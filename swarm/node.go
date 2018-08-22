package swarm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
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
	"../repo"
	"../util"
	"./nodeeth"
	. "./wire"
)

type Node struct {
	host        host.Host
	dht         *dht.IpfsDHT
	eth         *nodeeth.Client
	RepoManager *RepoManager
	Config      config.Config
	Shutdown    chan struct{}
}

const (
	OBJECT_PROTO      = "/conscience/object/1.0.0"
	REPLICATION_PROTO = "/conscience/replication/1.0.0"
)

func NewNode(ctx context.Context, cfg *config.Config) (*Node, error) {
	if cfg == nil {
		cfg = &config.DefaultConfig
	}

	// Initialize the p2p host
	h, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/%v/tcp/%v", cfg.Node.P2PListenAddr, cfg.Node.P2PListenPort),
		),
		libp2p.NATPortMap(),
	)
	if err != nil {
		return nil, err
	}

	// Initialize the DHT
	d := dht.NewDHT(ctx, h, dsync.MutexWrap(dstore.NewMapDatastore()))
	d.Validator = blankValidator{} // Set a pass-through validator

	// Initialize the Ethereum client
	eth, err := nodeeth.NewClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	n := &Node{
		host:        h,
		dht:         d,
		eth:         eth,
		RepoManager: NewRepoManager(cfg),
		Config:      *cfg,
		Shutdown:    make(chan struct{}),
	}

	go n.periodicallyAnnounceContent(ctx) // Start a goroutine for announcing which repos and objects this Node can provide
	go n.periodicallyRequestContent(ctx)  // Start a goroutine for pulling content from repos we are replicating

	// Set the handler function for when we get a new incoming object stream
	n.host.SetStreamHandler(OBJECT_PROTO, n.handleObjectRequest)
	n.host.SetStreamHandler(REPLICATION_PROTO, n.handleReplicationRequest)

	// Connect to our list of bootstrap peers
	for _, peeraddr := range cfg.Node.BootstrapPeers {
		err = n.AddPeer(ctx, peeraddr)
		if err != nil {
			log.Errorf("[node] could not reach boostrap peer %v", peeraddr)
		}
	}

	return n, nil
}

func (n *Node) Close() error {
	close(n.Shutdown)

	err := n.host.Close()
	if err != nil {
		return err
	}

	err = n.dht.Close()
	if err != nil {
		return err
	}

	err = n.eth.Close()
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) ID() peer.ID {
	return n.host.ID()
}

func (n *Node) Addrs() []ma.Multiaddr {
	return n.host.Addrs()
}

func (n *Node) Peers() []pstore.PeerInfo {
	return pstore.PeerInfos(n.host.Peerstore(), n.host.Peerstore().Peers())
}

func (n *Node) Conns() []netp2p.Conn {
	return n.host.Network().Conns()
}

type NodeState struct {
	User       string
	EthAccount string
	Addrs      []string
	Peers      map[string][]string
	Repos      map[string]repo.RepoInfo
}

func (n *Node) GetNodeState() (*NodeState, error) {
	user := n.Config.User.Username
	ethAccount := n.eth.Address().Hex()

	addrs := make([]string, 0)
	for _, addr := range n.host.Addrs() {
		addrs = append(addrs, fmt.Sprintf("%v/p2p/%v", addr.String(), n.host.ID().Pretty()))
	}

	peers := make(map[string][]string)
	for _, peerID := range n.host.Peerstore().Peers() {
		if peerID == n.host.ID() {
			continue
		}

		pid := peerID.Pretty()
		peers[pid] = make([]string, 0)
		for _, addr := range n.host.Peerstore().Addrs(peerID) {
			peers[pid] = append(peers[pid], addr.String())
		}
	}

	repos, err := n.RepoManager.GetReposInfo()
	if err != nil {
		log.Printf(err.Error())
		return nil, err
	}

	return &NodeState{
		user,
		ethAccount,
		addrs,
		peers,
		repos,
	}, nil
}

func (n *Node) periodicallyRequestContent(ctx context.Context) {
	c := time.Tick(time.Duration(n.Config.Node.ContentRequestInterval))
	for range c {
		log.Infof("[content request] starting content request")

		for _, repoID := range n.Config.Node.ReplicateRepos {
			log.Infof("[content request] requesting repo '%v'", repoID)
			err := n.pullRepo(repoID)
			if err != nil {
				log.Errorf("[content request] error pulling repo (%v): %v", repoID, err)
			}
		}
	}
}

// Periodically announces our repos and objects to the network.
func (n *Node) periodicallyAnnounceContent(ctx context.Context) {
	c := time.Tick(time.Duration(n.Config.Node.ContentAnnounceInterval))
	for range c {
		log.Infof("[content announce] starting content announce")

		// Announce what we're willing to replicate.
		for _, repoID := range n.Config.Node.ReplicateRepos {
			log.Infof("[content announce] announcing repo '%v'", repoID)

			err := n.announceRepoReplicator(ctx, repoID)
			if err != nil {
				log.Errorf("[content announce] %v", err)
				continue
			}
		}

		// Announce the repos we have locally
		err := n.RepoManager.ForEachRepo(func(r *repo.Repo) error {
			repoID, err := r.RepoID()
			if err != nil {
				return err
			}

			err = n.announceRepo(ctx, repoID)
			if err != nil {
				return err
			}

			return r.ForEachObjectID(func(objectID []byte) error {
				return n.announceObject(ctx, repoID, objectID)
			})
		})
		if err != nil {
			log.Errorf("[content announce] %v", err)
		}
	}
}

// This method is called via the RPC connection when a user git-pushes new content to the network.
// A push is actually a request to be pulled from, and in order for peers to pull from us, they need
// to know that we have the content in question.  The content is new, and therefore hasn't been
// announced before; hence, the reason for this.
func (n *Node) AnnounceRepoContent(ctx context.Context, repoID string) error {
	repo := n.RepoManager.Repo(repoID)
	if repo == nil {
		return errors.New("repo not found")
	}

	err := n.announceRepo(ctx, repoID)
	if err != nil {
		return err
	}

	return repo.ForEachObjectID(func(objectID []byte) error {
		return n.announceObject(ctx, repoID, objectID)
	})
}

// Announce to the swarm that this Node can provide objects from the given repository.
func (n *Node) announceRepo(ctx context.Context, repoID string) error {
	c, err := cidForString(repoID)
	if err != nil {
		return err
	}

	err = n.dht.Provide(ctx, c, true)
	if err != nil && err != kbucket.ErrLookupFailure {
		return err
	}
	return nil
}

// Announce to the swarm that this Node is willing to replicate objects from the given repository.
func (n *Node) announceRepoReplicator(ctx context.Context, repoID string) error {
	c, err := cidForString("replicate:" + repoID)
	if err != nil {
		return err
	}

	err = n.dht.Provide(ctx, c, true)
	if err != nil && err != kbucket.ErrLookupFailure {
		return err
	}
	return nil
}

// Announce to the swarm that this Node can provide a specific object from a given repository.
func (n *Node) announceObject(ctx context.Context, repoID string, objectID []byte) error {
	c, err := cidForObject(repoID, objectID)
	if err != nil {
		return err
	}

	err = n.dht.Provide(ctx, c, true)
	if err != nil && err != kbucket.ErrLookupFailure {
		return err
	}
	return nil
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

	err = n.host.Connect(ctx, *pinfo)
	if err != nil {
		return fmt.Errorf("connect to peer failed: %v", err)
	}

	return nil
}

func (n *Node) RemovePeer(peerID peer.ID) error {
	if len(n.host.Network().ConnsToPeer(peerID)) > 0 {
		err := n.host.Network().ClosePeer(peerID)
		if err != nil {
			return err
		}
	}
	n.host.Peerstore().ClearAddrs(peerID)
	return nil
}

// Attempts to open a stream to the given object.  If we have it locally, the object is read from
// the filesystem.  Otherwise, we look for a peer and stream it over a p2p connection.
func (n *Node) GetObjectReader(ctx context.Context, repoID string, objectID []byte) (*util.ObjectReader, error) {
	r := n.RepoManager.Repo(repoID)

	// If we detect that we already have the object locally, just open a regular file stream
	if r != nil && r.HasObject(repoID, objectID) {
		return r.OpenObject(objectID)
	}

	c, err := cidForObject(repoID, objectID)
	if err != nil {
		return nil, err
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(n.Config.Node.FindProviderTimeout))
	defer cancel()

	for provider := range n.dht.FindProvidersAsync(ctxTimeout, c, 10) {
		if provider.ID == n.host.ID() {
			// We have the object locally (perhaps in another clone of the same repository)
			objectReader, err := r.OpenObject(objectID)
			if err != nil {
				continue
			}
			return objectReader, nil

		} else {
			// We found a peer with the object
			objectReader, err := n.requestObject(ctx, provider.ID, repoID, objectID)
			if err != nil {
				continue
			}
			return objectReader, nil
		}
	}
	return nil, errors.Errorf("could not find provider for %v : %v", repoID, objectID)
}

func (n *Node) SetReplicationPolicy(repoID string, shouldReplicate bool) error {
	return n.Config.Update(func() error {
		if shouldReplicate {
			n.Config.Node.ReplicateRepos = util.StringSetAdd(n.Config.Node.ReplicateRepos, repoID)
		} else {
			n.Config.Node.ReplicateRepos = util.StringSetRemove(n.Config.Node.ReplicateRepos, repoID)
		}
		return nil
	})
}

// Finds replicator nodes on the network that are hosting the given repository and issues requests
// to them to pull from our local copy.
func (n *Node) RequestReplication(ctx context.Context, repoID string) error {
	log.Printf("requesting replication of '%v'", repoID)
	c, err := cidForString("replicate:" + repoID)
	if err != nil {
		return err
	}

	// @@TODO: configurable timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	chProviders := n.dht.FindProvidersAsync(ctxTimeout, c, 8)

	wg := &sync.WaitGroup{}
	for provider := range chProviders {
		if provider.ID == n.host.ID() {
			continue
		}

		wg.Add(1)

		go func(peerID peer.ID) {
			defer wg.Done()

			log.Printf("[pull] requesting pull of %v from %v", repoID, peerID.String())
			stream, err := n.host.NewStream(ctx, peerID, REPLICATION_PROTO)
			if err != nil {
				log.Errorf("[pull] error: %v", err)
				return
			}
			defer stream.Close()

			err = WriteStructPacket(stream, &ReplicationRequest{RepoID: repoID})
			if err != nil {
				log.Errorf("[pull] error: %v", err)
				return
			}

			resp := ReplicationResponse{}
			err = ReadStructPacket(stream, &resp)
			if err != nil {
				log.Errorf("[pull] error: %v", err)
				return
			}

			if resp.Error != "" {
				log.Printf("[pull] request rejected by peer %v (error: %v)", peerID.String(), resp.Error)
				return
			}
			log.Printf("[pull] request accepted by peer %v", peerID.String())

		}(provider.ID)
	}
	wg.Wait()

	return nil
}

// Handles an incoming request to replicate (pull changes from) a given repository.
func (n *Node) handleReplicationRequest(stream netp2p.Stream) {
	log.Printf("[replication] receiving replication request")
	defer stream.Close()

	req := ReplicationRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[replication] error: %v", err)
		return
	}
	log.Printf("[replication] repoID: %v", req.RepoID)

	// Ensure that the repo has been whitelisted for replication.
	whitelisted := false
	for _, repo := range n.Config.Node.ReplicateRepos {
		if repo == req.RepoID {
			whitelisted = true
			break
		}
	}

	if !whitelisted {
		err = WriteStructPacket(stream, &ReplicationResponse{Error: "not a whitelisted repo"})
		if err != nil {
			log.Errorf("[replication] error: %v", err)
		}
		return
	}

	err = n.pullRepo(req.RepoID)
	if err != nil {
		log.Errorf("[replication] error: %v", err)

		err = WriteStructPacket(stream, &ReplicationResponse{Error: err.Error()})
		if err != nil {
			log.Errorf("[replication] error: %v", err)
			return
		}
		return
	}

	err = WriteStructPacket(stream, &ReplicationResponse{Error: ""})
	if err != nil {
		log.Errorf("[replication] error: %v", err)
		return
	}
}

func (n *Node) pullRepo(repoID string) error {
	r, err := n.RepoManager.EnsureLocalCheckoutExists(repoID)
	if err != nil {
		return err
	}

	// Start a git-pull process
	cmd := exec.Command("git", "pull", "origin", "master")
	cmd.Dir = r.Path
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	err = cmd.Run()
	if err != nil {
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Errorf("[pullRepo] error running git pull, then error trying to get CombinedOutput")
			return err
		}
		log.Errorf("[pullRepo] error running git pull: %v", string(output))
		return err
	}

	scan := bufio.NewScanner(buf)
	for scan.Scan() {
		log.Printf("[replication] git: %v", scan.Text())
	}
	if err = scan.Err(); err != nil {
		return err
	}
	return nil
}

func (n *Node) EnsureUsername(ctx context.Context, username string) (*nodeeth.Transaction, error) {
	return n.eth.EnsureUsername(ctx, username)
}

func (n *Node) EnsureRepoIDRegistered(ctx context.Context, repoID string) (*nodeeth.Transaction, error) {
	return n.eth.EnsureRepoIDRegistered(ctx, repoID)
}

func (n *Node) GetNumRefs(ctx context.Context, repoID string) (int64, error) {
	return n.eth.GetNumRefs(ctx, repoID)
}

func (n *Node) GetRefs(ctx context.Context, repoID string, page int64) (map[string]Ref, error) {
	return n.eth.GetRefs(ctx, repoID, page)
}

func (n *Node) UpdateRef(ctx context.Context, repoID string, refName string, commitHash string) (*nodeeth.Transaction, error) {
	return n.eth.UpdateRef(ctx, repoID, refName, commitHash)
}
