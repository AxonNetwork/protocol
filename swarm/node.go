package swarm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/pkg/errors"

	cid "github.com/ipfs/go-cid"
	dstore "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	metrics "github.com/libp2p/go-libp2p-metrics"
	netp2p "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/nodeeth"
	"github.com/Conscience/protocol/swarm/nodep2p"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

type Node struct {
	host        host.Host
	dht         *dht.IpfsDHT
	eth         *nodeeth.Client
	repoManager *RepoManager
	Config      config.Config
	Shutdown    chan struct{}

	bandwidthCounter *metrics.BandwidthCounter
}

const (
	REPLICATION_PROTO       = "/conscience/replication/1.0.0"
	BECOME_REPLICATOR_PROTO = "/conscience/become-replicator/1.0.0"
)

func NewNode(ctx context.Context, cfg *config.Config) (*Node, error) {
	if cfg == nil {
		cfg = &config.DefaultConfig
	}

	privkey, err := obtainKey(cfg)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(cfg.Node.ReplicationRoot, os.ModePerm)
	if err != nil {
		return nil, err
	}

	bandwidthCounter := metrics.NewBandwidthCounter()

	// Initialize the p2p host
	h, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/%v/tcp/%v", cfg.Node.P2PListenAddr, cfg.Node.P2PListenPort),
		),
		libp2p.Identity(privkey),
		libp2p.BandwidthReporter(bandwidthCounter),
		libp2p.NATPortMap(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize libp2p host")
	}

	// Initialize the DHT
	d := dht.NewDHT(ctx, h, dsync.MutexWrap(dstore.NewMapDatastore()))
	d.Validator = blankValidator{} // Set a pass-through validator

	// Initialize the Ethereum client
	eth, err := nodeeth.NewClient(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize Ethereum client")
	}

	username, err := eth.GetUsername(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch username from Ethereum smart contract")
	}
	log.SetField("username", username)

	n := &Node{
		host:             h,
		dht:              d,
		eth:              eth,
		repoManager:      NewRepoManager(cfg),
		Config:           *cfg,
		Shutdown:         make(chan struct{}),
		bandwidthCounter: bandwidthCounter,
	}

	go n.periodicallyAnnounceContent(ctx) // Start a goroutine for announcing which repos and objects this Node can provide
	go n.periodicallyRequestContent(ctx)  // Start a goroutine for pulling content from repos we are replicating

	ns := nodep2p.NewServer(n)
	// n.host.SetStreamHandler(nodep2p.OBJECT_PROTO, ns.HandleObjectRequest)
	n.host.SetStreamHandler(nodep2p.MANIFEST_PROTO, ns.HandleManifestRequest)
	n.host.SetStreamHandler(nodep2p.OBJECT_PROTO, ns.HandleObjectStreamRequest)
	n.host.SetStreamHandler(nodep2p.PACKFILE_PROTO, ns.HandlePackfileStreamRequest)
	n.host.SetStreamHandler(REPLICATION_PROTO, n.handleReplicationRequest)
	n.host.SetStreamHandler(BECOME_REPLICATOR_PROTO, n.handleBecomeReplicatorRequest)

	// Connect to our list of bootstrap peers
	go func() {
		for _, peeraddr := range cfg.Node.BootstrapPeers {
			err = n.AddPeer(ctx, peeraddr)
			if err != nil {
				log.Errorf("[node] could not reach boostrap peer %v", peeraddr)
			}
		}
	}()

	return n, nil
}

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

func obtainKey(cfg *config.Config) (crypto.PrivKey, error) {
	f, err := os.Open(cfg.Node.PrivateKeyFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err

	} else if err == nil {
		defer f.Close()

		data, err := ioutil.ReadFile(cfg.Node.PrivateKeyFile)
		if err != nil {
			return nil, err
		}
		return crypto.UnmarshalPrivateKey(data)
	}

	privkey, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 0)
	if err != nil {
		return nil, err
	}

	bs, err := privkey.Bytes()
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(cfg.Node.PrivateKeyFile, bs, 0400)
	if err != nil {
		return nil, err
	}

	return privkey, nil
}

func (n *Node) Close() error {
	close(n.Shutdown)

	err := n.host.Close()
	if err != nil {
		return errors.Wrap(err, "could not .Close libp2p host")
	}

	err = n.dht.Close()
	if err != nil {
		return errors.Wrap(err, "could not .Close libp2p DHT")
	}

	err = n.eth.Close()
	if err != nil {
		return errors.Wrap(err, "could not .Close Ethereum client")
	}

	return nil
}

func (n *Node) periodicallyRequestContent(ctx context.Context) {
	c := time.Tick(time.Duration(n.Config.Node.ContentRequestInterval))
	for range c {
		log.Debugf("[content request] starting content request")

		for _, repoID := range n.Config.Node.ReplicateRepos {
			log.Debugf("[content request] requesting repo '%v'", repoID)
			err := n.PullRepo(repoID)
			if err != nil {
				log.Errorf("[content request] error pulling repo (%v): %+v", repoID, err)
			}
		}
	}
}

// Periodically announces our repos and objects to the network.
func (n *Node) periodicallyAnnounceContent(ctx context.Context) {
	c := time.Tick(time.Duration(n.Config.Node.ContentAnnounceInterval))
	for range c {
		log.Debugf("[content announce] starting content announce")

		// Announce what we're willing to replicate.
		for _, repoID := range n.Config.Node.ReplicateRepos {
			log.Debugf("[content announce] announcing repo '%v'", repoID)

			err := n.announceRepoReplicator(ctx, repoID)
			if err != nil {
				log.Errorf("[content announce] %+v", err)
				continue
			}
		}

		// Announce the repos we have locally
		err := n.repoManager.ForEachRepo(func(r *repo.Repo) error {
			repoID, err := r.RepoID()
			if err != nil {
				return err
			}

			err = n.announceRepo(ctx, repoID)
			if err != nil {
				return err
			}

			// return r.ForEachObjectID(func(objectID []byte) error {
			//  return n.announceObject(ctx, repoID, objectID)
			// })
			return nil
		})
		if err != nil {
			log.Errorf("[content announce] %+v", err)
		}
	}
}

// This method is called via the RPC connection when a user git-pushes new content to the network.
// A push is actually a request to be pulled from, and in order for peers to pull from us, they need
// to know that we have the content in question.  The content is new, and therefore hasn't been
// announced before; hence, the reason for this.
func (n *Node) AnnounceRepoContent(ctx context.Context, repoID string) error {
	repo := n.repoManager.Repo(repoID)
	if repo == nil {
		return errors.Errorf("repo '%v' not found", repoID)
	}

	err := n.announceRepo(ctx, repoID)
	if err != nil {
		return err
	}

	// return repo.ForEachObjectID(func(objectID []byte) error {
	//  return n.announceObject(ctx, repoID, objectID)
	// })
	return nil
}

// Announce to the swarm that this Node can provide objects from the given repository.
func (n *Node) announceRepo(ctx context.Context, repoID string) error {
	c, err := util.CidForString(repoID)
	if err != nil {
		return err
	}

	err = n.dht.Provide(ctx, c, true)
	if err != nil && err != kbucket.ErrLookupFailure {
		return errors.Wrapf(err, "could not dht.Provide repo '%v'", repoID)
	}
	return nil
}

// Announce to the swarm that this Node is willing to replicate objects from the given repository.
func (n *Node) announceRepoReplicator(ctx context.Context, repoID string) error {
	c, err := util.CidForString("replicate:" + repoID)
	if err != nil {
		return err
	}

	err = n.dht.Provide(ctx, c, true)
	if err != nil && err != kbucket.ErrLookupFailure {
		return errors.Wrapf(err, "could not dht.Provide replicator for repo '%v'", repoID)
	}
	return nil
}

// Announce to the swarm that this Node can provide a specific object from a given repository.
func (n *Node) announceObject(ctx context.Context, repoID string, objectID []byte) error {
	c, err := util.CidForObject(repoID, objectID)
	if err != nil {
		return err
	}

	err = n.dht.Provide(ctx, c, true)
	if err != nil && err != kbucket.ErrLookupFailure {
		return errors.Wrapf(err, "could not dht.Provide object '%0x' in repo '%v'", objectID, repoID)
	}
	return nil
}

// Adds a peer to the Node's address book and attempts to .Connect to it using the libp2p Host.
func (n *Node) AddPeer(ctx context.Context, multiaddrString string) error {
	// The following code extracts the peer ID from the given multiaddress
	addr, err := ma.NewMultiaddr(multiaddrString)
	if err != nil {
		return errors.Wrapf(err, "could not parse multiaddr '%v'", multiaddrString)
	}

	pinfo, err := pstore.InfoFromP2pAddr(addr)
	if err != nil {
		return errors.Wrapf(err, "could not parse PeerInfo from multiaddr '%v'", multiaddrString)
	}

	err = n.host.Connect(ctx, *pinfo)
	if err != nil {
		return errors.Wrapf(err, "could not connect to peer '%v'", multiaddrString)
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

func (n *Node) FetchFromCommit(ctx context.Context, repoID string, path string, commit string) (<-chan nodep2p.MaybeFetchFromCommitPacket, int64) {
	log.Debugln("FETCHING FOR:", path, "/ repoID:", repoID)

	// repo, err := n.repoManager.RepoAtPathOrID(path, repoID)
	repo := n.repoManager.Repo(repoID)
	// if repo == nil {
	// 	log.Errorln("[Node.FetchFromCommit] error getting repo")
	// 	return nil, 0
	// }

	c := nodep2p.NewSmartPackfileClient(n, repo, &n.Config)

	return c.FetchFromCommit(ctx, repoID, commit)
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

func (n *Node) RequestBecomeReplicator(ctx context.Context, repoID string) error {
	for _, pubkeyStr := range n.Config.Node.KnownReplicators {
		peerID, err := peer.IDB58Decode(pubkeyStr)
		if err != nil {
			log.Errorf("RequestBecomeReplicator: bad pubkey string '%v': %v", pubkeyStr, err)
			continue
		}

		stream, err := n.host.NewStream(ctx, peerID, BECOME_REPLICATOR_PROTO)
		if err != nil {
			log.Errorf("RequestBecomeReplicator: error connecting to peer %v: %v", peerID, err)
			continue
		}

		err = WriteStructPacket(stream, &BecomeReplicatorRequest{RepoID: repoID})
		if err != nil {
			log.Errorf("RequestBecomeReplicator: error writing request: %v", err)
			continue
		}

		resp := BecomeReplicatorResponse{}
		err = ReadStructPacket(stream, &resp)
		if err != nil {
			log.Errorf("RequestBecomeReplicator: error reading response: %v", err)
			continue
		}

		if resp.Error == "" {
			log.Infof("RequestBecomeReplicator: peer %v agreed to replicate %v", peerID, repoID)
		} else {
			log.Infof("RequestBecomeReplicator: peer %v refused to replicate %v (err: %v)", peerID, repoID, resp.Error)
		}
	}
	return nil
}

// Finds replicator nodes on the network that are hosting the given repository and issues requests
// to them to pull from our local copy.
func (n *Node) RequestReplication(ctx context.Context, repoID string) error {
	c, err := util.CidForString("replicate:" + repoID)
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

func (n *Node) handleBecomeReplicatorRequest(stream netp2p.Stream) {
	log.Printf("[become replicator] receiving 'become replicator' request")
	defer stream.Close()

	req := BecomeReplicatorRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[become replicator] error: %v", err)
		return
	}
	log.Debugf("[become replicator] repoID: %v", req.RepoID)

	if n.Config.Node.ReplicateEverything {
		err = n.SetReplicationPolicy(req.RepoID, true)
		if err != nil {
			log.Errorf("[become replicator] error: %v", err)
			_ = WriteStructPacket(stream, &BecomeReplicatorResponse{Error: err.Error()})
			return
		}

		// Acknowledge that we will now replicate the repo
		err = WriteStructPacket(stream, &BecomeReplicatorResponse{Error: ""})
		if err != nil {
			log.Errorf("[become replicator] error: %v", err)
			return
		}

	} else {
		err = WriteStructPacket(stream, &BecomeReplicatorResponse{Error: "no"})
		if err != nil {
			log.Errorf("[become replicator] error: %v", err)
			return
		}
	}
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
	log.Debugf("[replication] repoID: %v", req.RepoID)

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

	err = n.PullRepo(req.RepoID)
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

func (n *Node) PullRepo(repoID string) error {
	r, err := n.repoManager.EnsureLocalCheckoutExists(repoID)
	if err != nil {
		return err
	}

	// Start a git-pull process
	// @@TODO: make timeout configurable
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "pull", "origin", "master")
	cmd.Dir = r.Path
	cmd.Env = util.CopyEnv()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		log.Errorf("[pull repo] error running git pull: %v", string(stderr.Bytes()))
		return err
	}

	scan := bufio.NewScanner(stdout)
	for scan.Scan() {
		log.Debugf("[pull repo] git (stdout): %v", scan.Text())
	}
	if err = scan.Err(); err != nil {
		return err
	}

	scan = bufio.NewScanner(stderr)
	for scan.Scan() {
		log.Debugf("[pull repo] git (stderr): %v", scan.Text())
	}
	if err = scan.Err(); err != nil {
		return err
	}

	return nil
}

func (n *Node) GetConfig() config.Config {
	return n.Config
}

func (n *Node) RepoManager() *RepoManager {
	return n.repoManager
}

func (n *Node) Repo(repoID string) *repo.Repo {
	return n.repoManager.Repo(repoID)
}

func (n *Node) ID() peer.ID {
	return n.host.ID()
}

func (n *Node) Addrs() []ma.Multiaddr {
	return n.host.Addrs()
}

func (n *Node) NewStream(ctx context.Context, peerID peer.ID, pids ...protocol.ID) (netp2p.Stream, error) {
	return n.host.NewStream(ctx, peerID, pids...)
}

func (n *Node) FindProvidersAsync(ctx context.Context, key cid.Cid, count int) <-chan pstore.PeerInfo {
	return n.dht.FindProvidersAsync(ctx, key, count)
}

func (n *Node) Peers() []pstore.PeerInfo {
	return pstore.PeerInfos(n.host.Peerstore(), n.host.Peerstore().Peers())
}

func (n *Node) Conns() []netp2p.Conn {
	return n.host.Network().Conns()
}

func (n *Node) EthAddress() nodeeth.Address {
	return n.eth.Address()
}

func (n *Node) GetUsername(ctx context.Context) (string, error) {
	return n.eth.GetUsername(ctx)
}

func (n *Node) EnsureUsername(ctx context.Context, username string) (*nodeeth.Transaction, error) {
	return n.eth.EnsureUsername(ctx, username)
}

func (n *Node) EnsureRepoIDRegistered(ctx context.Context, repoID string) (*nodeeth.Transaction, error) {
	return n.eth.EnsureRepoIDRegistered(ctx, repoID)
}

func (n *Node) AddrFromSignedHash(data, sig []byte) (nodeeth.Address, error) {
	return n.eth.AddrFromSignedHash(data, sig)
}

func (n *Node) AddressHasPullAccess(ctx context.Context, user nodeeth.Address, repoID string) (bool, error) {
	return n.eth.AddressHasPullAccess(ctx, user, repoID)
}

func (n *Node) GetLocalRefs(ctx context.Context, repoID string, path string) (map[string]Ref, string, error) {
	r, err := n.repoManager.RepoAtPathOrID(path, repoID)
	if err != nil {
		return nil, "", err
	}

	refs, err := r.GetLocalRefs(ctx)
	if err != nil {
		return nil, "", err
	}
	return refs, r.Path, nil
}

func (n *Node) IsBehindRemote(ctx context.Context, repoID string, path string) (bool, error) {
	r, err := n.repoManager.RepoAtPathOrID(path, repoID)
	if err != nil {
		return false, err
	}

	remote, err := n.eth.GetRef(ctx, repoID, "refs/heads/master")
	if err != nil {
		return false, err
	}

	if len(remote) == 0 {
		return false, nil
	}

	remoteHash, err := hex.DecodeString(remote)
	if err != nil {
		return false, err
	}
	return !r.HasObject(remoteHash), nil
}

func (n *Node) GetNumRefs(ctx context.Context, repoID string) (uint64, error) {
	return n.eth.GetNumRefs(ctx, repoID)
}

func (n *Node) GetRemoteRefs(ctx context.Context, repoID string, pageSize uint64, page uint64) (map[string]Ref, uint64, error) {
	return n.eth.GetRefs(ctx, repoID, pageSize, page)
}

func (n *Node) UpdateRef(ctx context.Context, repoID string, refName string, commitHash string) (*nodeeth.Transaction, error) {
	return n.eth.UpdateRef(ctx, repoID, refName, commitHash)
}

func (n *Node) GetRepoUsers(ctx context.Context, repoID string, userType nodeeth.UserType, pageSize uint64, page uint64) ([]string, uint64, error) {
	return n.eth.GetRepoUsers(ctx, repoID, userType, pageSize, page)
}

func (n *Node) GetRefLogs(ctx context.Context, repoID string) (map[string]uint64, error) {
	return n.eth.GetRefLogs(ctx, repoID)
}

func (n *Node) SignHash(data []byte) ([]byte, error) {
	return n.eth.SignHash(data)
}

func (n *Node) SetUserPermissions(ctx context.Context, repoID string, username string, perms nodeeth.UserPermissions) (*nodeeth.Transaction, error) {
	return n.eth.SetUserPermissions(ctx, repoID, username, perms)
}

func (n *Node) GetBandwidthForPeer(p peer.ID) metrics.Stats {
	return n.bandwidthCounter.GetBandwidthForPeer(p)
}

func (n *Node) GetBandwidthForProtocol(proto protocol.ID) metrics.Stats {
	return n.bandwidthCounter.GetBandwidthForProtocol(proto)
}

func (n *Node) GetBandwidthTotals() metrics.Stats {
	return n.bandwidthCounter.GetBandwidthTotals()
}
