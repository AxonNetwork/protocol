package nodep2p

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
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
	"github.com/Conscience/protocol/swarm/nodeevents"
	"github.com/Conscience/protocol/util"
)

type Host struct {
	libp2pHost  host.Host
	dht         *dht.IpfsDHT
	repoManager *repo.Manager
	ethClient   *nodeeth.Client
	eventBus    *nodeevents.EventBus
	Config      *config.Config
	*metrics.BandwidthCounter
}

func NewHost(ctx context.Context, repoManager *repo.Manager, ethClient *nodeeth.Client, eventBus *nodeevents.EventBus, cfg *config.Config) (*Host, error) {
	privkey, err := obtainP2PKey(cfg)
	if err != nil {
		return nil, err
	}

	bandwidthCounter := metrics.NewBandwidthCounter()

	// Initialize the libp2p host
	libp2pHost, err := libp2p.New(ctx,
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
	d := dht.NewDHT(ctx, libp2pHost, dsync.MutexWrap(dstore.NewMapDatastore()))
	d.Validator = blankValidator{} // Set a pass-through validator

	h := &Host{
		libp2pHost:       libp2pHost,
		dht:              d,
		repoManager:      repoManager,
		ethClient:        ethClient,
		eventBus:         eventBus,
		Config:           cfg,
		BandwidthCounter: bandwidthCounter,
	}

	err = RegisterTransport(ethClient, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not register axon:// git transport")
	}

	go h.periodicallyAnnounceContent(ctx) // Start a goroutine for announcing which repos and objects this node can provide
	go h.periodicallyRequestContent(ctx)  // Start a goroutine for pulling content from repos we are replicating

	h.libp2pHost.SetStreamHandler(MANIFEST_PROTO, h.handleManifestRequest)
	h.libp2pHost.SetStreamHandler(PACKFILE_PROTO, h.handlePackfileStreamRequest)
	h.libp2pHost.SetStreamHandler(CHUNK_PROTO, h.handleChunkStreamRequest)
	h.libp2pHost.SetStreamHandler(REPLICATION_PROTO, h.handleReplicationRequest)

	// Connect to our list of bootstrap peers
	go func() {
		for _, peeraddr := range cfg.Node.BootstrapPeers {
			err = h.AddPeer(ctx, peeraddr)
			if err != nil {
				log.Errorf("[node] could not reach boostrap peer %v: %v", peeraddr, err)
			}
		}
	}()

	return h, nil
}

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

func obtainP2PKey(cfg *config.Config) (crypto.PrivKey, error) {
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

func (h *Host) Close() error {
	err := h.libp2pHost.Close()
	if err != nil {
		return errors.Wrap(err, "could not .Close libp2p host")
	}

	err = h.dht.Close()
	if err != nil {
		return errors.Wrap(err, "could not .Close libp2p DHT")
	}

	return nil
}

func (h *Host) periodicallyRequestContent(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		log.Debugf("[content request] starting content request")

		for repoID, policy := range h.Config.Node.ReplicationPolicies {
			log.Debugf("[content request] requesting repo '%v'", repoID)

			// @@TODO: make context timeout configurable
			innerCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

			err := h.Replicate(innerCtx, repoID, policy, func(current, total uint64) error { return nil })
			if err != nil {
				log.Errorf("[content request]")
			}
			cancel()
		}

		time.Sleep(time.Duration(h.Config.Node.ContentRequestInterval))
	}
}

// Periodically announces our repos and objects to the network.
func (h *Host) periodicallyAnnounceContent(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		log.Debugf("[content announce] starting content announce")

		// Announce what we're willing to replicate.
		for repoID, policy := range h.Config.Node.ReplicationPolicies {
			if policy.MaxBytes <= 0 {
				continue
			}

			log.Debugf("[content announce] i'm a replicator for '%v'", repoID)

			ctxInner, cancel := context.WithTimeout(ctx, 10*time.Second)

			err := h.announceRepoReplicator(ctxInner, repoID)
			if err != nil {
				log.Warnf("[content announce] %+v", err)
				continue
			}

			cancel()
		}

		// Announce the repos we have locally
		_ = h.repoManager.ForEachRepo(func(r *repo.Repo) error {
			repoID, err := r.RepoID()
			if err != nil {
				return err
			}

			ctxInner, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			err = h.AnnounceRepo(ctxInner, repoID)
			if err != nil {
				log.Warnf("[content announce] error announcing repo: %+v", err)
			}
			return nil
		})

		time.Sleep(time.Duration(h.Config.Node.ContentAnnounceInterval))
	}
}

// Announce to the swarm that this Node can provide objects from the given repository.
func (h *Host) AnnounceRepo(ctx context.Context, repoID string) error {
	c, err := util.CidForString(repoID)
	if err != nil {
		return err
	}

	err = h.dht.Provide(ctx, c, true)
	if err != nil && err != kbucket.ErrLookupFailure {
		return errors.Wrapf(err, "could not dht.Provide repo '%v'", repoID)
	}
	return nil
}

// Announce to the swarm that this Node is willing to replicate objects from the given repository.
func (h *Host) announceRepoReplicator(ctx context.Context, repoID string) error {
	c, err := util.CidForString("replicate:" + repoID)
	if err != nil {
		return err
	}

	err = h.dht.Provide(ctx, c, true)
	if err != nil && err != kbucket.ErrLookupFailure {
		return errors.Wrapf(err, "could not dht.Provide replicator for repo '%v'", repoID)
	}
	return nil
}

// Adds a peer to the Node's address book and attempts to .Connect to it using the libp2p Host.
func (h *Host) AddPeer(ctx context.Context, multiaddrString string) error {
	// The following code extracts the peer ID from the given multiaddress
	addr, err := ma.NewMultiaddr(multiaddrString)
	if err != nil {
		return errors.Wrapf(err, "could not parse multiaddr '%v'", multiaddrString)
	}

	pinfo, err := pstore.InfoFromP2pAddr(addr)
	if err != nil {
		return errors.Wrapf(err, "could not parse PeerInfo from multiaddr '%v'", multiaddrString)
	}

	err = h.libp2pHost.Connect(ctx, *pinfo)
	if err != nil {
		return errors.Wrapf(err, "could not connect to peer '%v'", multiaddrString)
	}
	return nil
}

func (h *Host) RemovePeer(peerID peer.ID) error {
	if len(h.libp2pHost.Network().ConnsToPeer(peerID)) > 0 {
		err := h.libp2pHost.Network().ClosePeer(peerID)
		if err != nil {
			return err
		}
	}
	h.libp2pHost.Peerstore().ClearAddrs(peerID)
	return nil
}

func (h *Host) ID() peer.ID {
	return h.libp2pHost.ID()
}

func (h *Host) Addrs() []ma.Multiaddr {
	return h.libp2pHost.Addrs()
}

func (h *Host) NewStream(ctx context.Context, peerID peer.ID, pids ...protocol.ID) (netp2p.Stream, error) {
	return h.libp2pHost.NewStream(ctx, peerID, pids...)
}

func (h *Host) FindProvidersAsync(ctx context.Context, key cid.Cid, count int) <-chan pstore.PeerInfo {
	return h.dht.FindProvidersAsync(ctx, key, count)
}

func (h *Host) Peers() []pstore.PeerInfo {
	return pstore.PeerInfos(h.libp2pHost.Peerstore(), h.libp2pHost.Peerstore().Peers())
}

func (h *Host) Conns() []netp2p.Conn {
	return h.libp2pHost.Network().Conns()
}

func (h *Host) SetReplicationPolicy(repoID string, maxBytes int64) error {
	return h.Config.Update(func() error {
		h.Config.Node.ReplicationPolicies[repoID] = config.ReplicationPolicy{
			MaxBytes: maxBytes,
			Bare:     true,
		}
		return nil
	})
}
