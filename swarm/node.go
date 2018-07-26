package swarm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	netp2p "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
	"gx/ipfs/QmQYwRL1T9dJtdCScoeRQwwvScbJTcWqnXhq4dYQ6Cu5vX/go-libp2p-kad-dht"
	ma "gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
	"gx/ipfs/QmZ86eLPtXkQ1Dfa992Q8NpXArUoWWh3y728JDcWvzRrvC/go-libp2p"
	pstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	"gx/ipfs/Qmb8T6YBBsjYsVGfrihQLfCJveczZnneSBqBKkYEBWDjge/go-libp2p-host"
	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
	dstore "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"
	dsync "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore/sync"
	"gx/ipfs/QmesQqwonP618R7cJZoFfA4ioYhhMKnDmtUxcAvvxEEGnw/go-libp2p-kbucket"

	"../config"
)

type Node struct {
	Host        host.Host
	DHT         *dht.IpfsDHT
	RepoManager *RepoManager
	Config      config.Config
	rpcListener net.Listener
	chShutdown  chan struct{}
}

const (
	OBJECT_STREAM_PROTO = "/conscience/object-stream/1.0.0"
	PULL_PROTO          = "/conscience/pull/1.0.0"
)

var (
	ErrObjectNotFound = fmt.Errorf("object not found")
	ErrProtocol       = fmt.Errorf("protocol error")
)

func NewNode(ctx context.Context, cfg *config.Config) (*Node, error) {
	if cfg == nil {
		cfg = &config.DefaultConfig
	}

	h, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/%v/tcp/%v", cfg.Node.P2PListenAddr, cfg.Node.P2PListenPort),
		),
		libp2p.NATPortMap(),
	)
	if err != nil {
		return nil, err
	}

	n := &Node{
		Host:        h,
		DHT:         dht.NewDHT(ctx, h, dsync.MutexWrap(dstore.NewMapDatastore())),
		RepoManager: NewRepoManager(),
		Config:      *cfg,
		rpcListener: nil,
		chShutdown:  make(chan struct{}),
	}

	// Set a pass-through validator
	n.DHT.Validator = blankValidator{}

	// Start a goroutine for announcing which repos and objects this Node can provide (happens every few seconds)
	go n.announceContent(ctx)

	// Set the handler function for when we get a new incoming object stream
	n.Host.SetStreamHandler(OBJECT_STREAM_PROTO, n.objectStreamHandler)
	n.Host.SetStreamHandler(PULL_PROTO, n.pullHandler)

	// Setup the RPC interface so our git helpers can push and pull objects to the network
	err = n.initRPC(cfg.Node.RPCListenNetwork, cfg.Node.RPCListenHost)
	if err != nil {
		return nil, err
	}

	return n, nil
}

func (n *Node) Close() error {
	close(n.chShutdown)

	err := n.Host.Close()
	if err != nil {
		return err
	}

	err = n.DHT.Close()
	if err != nil {
		return err
	}

	return n.rpcListener.Close()
}

func (n *Node) announceContent(ctx context.Context) {
	c := time.Tick(time.Duration(n.Config.Node.AnnounceInterval))
	for range c {
		err := n.RepoManager.ForEachRepo(func(repo RepoEntry) error {
			err := n.ProvideRepo(ctx, repo.RepoID)
			if err != nil {
				return err
			}

			return repo.ForEachObject(func(objectID []byte) error {
				return n.ProvideObject(ctx, repo.RepoID, objectID)
			})
		})
		if err != nil && err != kbucket.ErrLookupFailure {
			log.Errorf("[announce] %v", err)
		}
	}
}

// Adds a peer to the Node's address book and attempts to .Connect to it using the libp2p Host.
func (n *Node) AddPeer(ctx context.Context, multiaddrString string) error {
	// The following code extracts the peer ID from the given multiaddress
	addr, err := ma.NewMultiaddr(multiaddrString)
	if err != nil {
		return err
	}

	pinfo, err := pstore.InfoFromP2pAddr(addr)
	if err != nil {
		return err
	}

	err = n.Host.Connect(ctx, *pinfo)
	if err != nil {
		return fmt.Errorf("connect to peer failed: %v", err)
	}

	return nil
}

// Announce to the swarm that this Node is tracking/replicating a given repository.
func (n *Node) ProvideRepo(ctx context.Context, repoID string) error {
	c, err := cidForString(repoID)
	if err != nil {
		return err
	}
	return n.DHT.Provide(ctx, c, true)
}

// Announce to the swarm that this Node can provide a given object from a given repository.
func (n *Node) ProvideObject(ctx context.Context, repoID string, objectID []byte) error {
	c, err := cidForObject(repoID, objectID)
	if err != nil {
		return err
	}
	return n.DHT.Provide(ctx, c, true)
}

// Attempts to find a peer providing the given object.  If a peer is found, the Node tries to
// download the object from that peer.
func (n *Node) GetObjectReader(ctx context.Context, repoID string, objectID []byte) (ObjectReader, error) {
	// If we detect that we already have the object locally, just open a regular file stream
	if n.RepoManager.HasObject(repoID, objectID) {
		return n.openLocalObjectReader(repoID, objectID)
	}

	c, err := cidForObject(repoID, objectID)
	if err != nil {
		return nil, err
	}

	// Try to find 1 provider of the object within the given timeout
	// @@TODO: reach out to multiple peers, take first responder
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(n.Config.Node.FindProviderTimeout))
	defer cancel()

	provider, found := <-n.DHT.FindProvidersAsync(ctxTimeout, c, 1)
	if !found {
		return nil, errors.New("can't find peer with object " + repoID + ":" + hex.EncodeToString(objectID))
	}

	if provider.ID == n.Host.ID() {
		// We have the object locally (perhaps in another clone of the same repository)
		return n.openLocalObjectReader(repoID, objectID)

	} else {
		// We found a peer with the object
		return n.openPeerObjectReader(ctx, provider.ID, repoID, objectID)
	}
}

func (n *Node) openLocalObjectReader(repoID string, objectID []byte) (ObjectReader, error) {
	return n.RepoManager.OpenObject(repoID, objectID)
}

func (n *Node) AddRepo(ctx context.Context, repoPath string) error {
	err := n.RepoManager.AddRepo(repoPath)
	if err != nil {
		return err
	}
	repo, err := n.RepoManager.GetGitRepo(repoPath)
	if err != nil {
		return err
	}
	username, repoName, err := n.RepoManager.GetRepoInfo(repo)
	if err != nil {
		return err
	}
	repoID := formatRepoID(username, repoName)

	refs, err := n.GetRefs(ctx, repoID)
	if len(refs) > 0 {
		// If refs exist, don't need to add HEAD
		return nil
	}
	head, err := n.RepoManager.GetHEAD(repo)
	if err != nil {
		return err
	}
	_, err = n.AddRef(ctx, repoID, head, "refs/heads/master")
	if err != nil {
		return err
	}
	_, err = n.AddRef(ctx, repoID, "@refs/heads/master", "HEAD")
	if err != nil {
		return err
	}
	return nil
}

func (n *Node) AddRef(ctx context.Context, repoID string, target string, name string) (map[string]string, error) {
	refs, err := n.GetRefs(ctx, repoID)
	if err != nil {
		return nil, err
	}
	refs[name] = target
	refStr, err := json.Marshal(refs)
	err = n.DHT.PutValue(ctx, repoID, refStr)
	return refs, err
}

func (n *Node) GetRefs(ctx context.Context, repoID string) (map[string]string, error) {
	refsBin, err := n.DHT.GetValue(ctx, repoID)
	if err != nil {
		// refs don't exist
		return map[string]string{}, nil
	}
	refs := map[string]string{}
	err = json.Unmarshal(refsBin, &refs)
	if err != nil {
		return nil, err
	}
	return refs, nil
}

// Finds replicator nodes on the network that are hosting the given repository and issues requests
// to them to pull from our local copy.
func (n *Node) requestReplication(ctx context.Context, repoID string) error {
	log.Printf("requesting replication of '%v'", repoID)
	c, err := cidForString(repoID)
	if err != nil {
		return err
	}

	// @@TODO: configurable timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	chProviders := n.DHT.FindProvidersAsync(ctxTimeout, c, 8)

	wg := &sync.WaitGroup{}
	for provider := range chProviders {
		log.Printf("[pull] my ID: %v / their ID: %v", n.Host.ID(), provider.ID)
		if provider.ID == n.Host.ID() {
			log.Printf("[pull] skipping %v", provider.ID)
			continue
		}

		wg.Add(1)

		go func(peerID peer.ID) {
			err = n.requestPull(ctx, peerID, repoID)
			if err != nil {
				log.Errorf("[pull] error: %v", err)
			}
			wg.Done()
		}(provider.ID)
	}
	wg.Wait()

	return nil
}

// Issues a request to a single replicator peer to pull from the given repository.
func (n *Node) requestPull(ctx context.Context, peerID peer.ID, repoID string) error {
	log.Printf("[pull] requesting pull of %v from %v", repoID, peerID.String())
	stream, err := n.Host.NewStream(ctx, peerID, PULL_PROTO)
	if err != nil {
		return err
	}
	defer stream.Close()

	err = writeStructPacket(stream, &PullRequest{RepoID: repoID})
	if err != nil {
		return err
	}

	resp := PullResponse{}
	err = readStructPacket(stream, &resp)
	if err != nil {
		return err
	}

	if resp.OK {
		log.Printf("[pull] request accepted by peer %v", peerID.String())
	} else {
		log.Printf("[pull] request rejected by peer %v", peerID.String())
	}

	return nil
}

// Handles an incoming request to replicate (pull changes from) a given repository.
func (n *Node) pullHandler(stream netp2p.Stream) {
	log.Printf("[pull] receiving pull request")
	defer stream.Close()

	req := PullRequest{}
	err := readStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[pull] error: %v", err)
		return
	}
	log.Printf("[pull] repoID: %v", req.RepoID)

	found := false
	for _, repo := range n.Config.Node.ReplicateRepos {
		if repo == req.RepoID {
			found = true
			break
		}
	}

	if !found {
		err = writeStructPacket(stream, &PullResponse{OK: false})
		if err != nil {
			log.Errorf("[pull] error: %v", err)
		}
		return
	}

	repo := n.RepoManager.Repo(req.RepoID)
	if repo == nil {
		log.Errorf("[pull] error: repo not found")
		return
	}

	// Start a git-pull process
	cmd := exec.Command("git", "pull", "origin", "master")
	cmd.Dir = repo.Path
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	err = cmd.Run()
	if err != nil {
		log.Errorf("[pull] error: %v", err)
		return
	}

	scan := bufio.NewScanner(buf)
	for scan.Scan() {
		log.Printf("[pull] git: %v", scan.Text())
	}
	if err = scan.Err(); err != nil {
		log.Errorf("[pull] error: %v", err)
		return
	}

	err = writeStructPacket(stream, &PullResponse{OK: true})
	if err != nil {
		log.Errorf("[pull] error: %v", err)
		return
	}
}

//
// Everything below here is fairly unimportant.
//

func (n *Node) FindProviders(ctx context.Context, contentID string) ([]pstore.PeerInfo, error) {
	c, err := cidForString(contentID)
	if err != nil {
		return nil, err
	}

	timeout, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	chProviders := n.DHT.FindProvidersAsync(timeout, c, 8)

	providers := []pstore.PeerInfo{}
ForLoop:
	for {
		select {
		case provider, ok := <-chProviders:
			if !ok {
				break ForLoop
			}

			if provider.ID == "" {
				log.Printf("got nil provider for %v")
			} else {
				log.Printf("got provider: %+v", provider)
				providers = append(providers, provider)
			}

		case <-timeout.Done():
			break ForLoop
		}
	}

	return providers, nil
}
