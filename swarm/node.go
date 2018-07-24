package swarm

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"gx/ipfs/QmQYwRL1T9dJtdCScoeRQwwvScbJTcWqnXhq4dYQ6Cu5vX/go-libp2p-kad-dht"
	ma "gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
	"gx/ipfs/QmZ86eLPtXkQ1Dfa992Q8NpXArUoWWh3y728JDcWvzRrvC/go-libp2p"
	pstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	"gx/ipfs/Qmb8T6YBBsjYsVGfrihQLfCJveczZnneSBqBKkYEBWDjge/go-libp2p-host"
	dstore "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"
	dsync "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore/sync"
	"gx/ipfs/QmesQqwonP618R7cJZoFfA4ioYhhMKnDmtUxcAvvxEEGnw/go-libp2p-kbucket"
)

type Node struct {
	Host        host.Host
	DHT         *dht.IpfsDHT
	RepoManager *RepoManager
}

const (
	OBJECT_STREAM_PROTO = "/conscience/object-stream/1.0.0"
)

var (
	ErrObjectNotFound = fmt.Errorf("object not found")
	ErrProtocol       = fmt.Errorf("protocol error")
)

func NewNode(ctx context.Context, listenPort int) (*Node, error) {
	h, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%v", listenPort),
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
	}

	// start a goroutine for announcing which repos and objects this Node can provide (happens every few seconds)
	// @@TODO: make announce interval configurable
	go func() {
		c := time.Tick(10 * time.Second)
		for range c {
			repoNames := n.RepoManager.RepoNames()
			for _, repoName := range repoNames {
				// log.Printf("[announce] %v", repoName)

				for _, object := range n.RepoManager.ObjectsForRepo(repoName) {
					err := n.ProvideObject(ctx, repoName, object.ID)
					if err != nil && err != kbucket.ErrLookupFailure {
						log.Errorf("[announce] %v", err)
					}
				}
			}
		}
	}()

	// Set a pass-through validator
	n.DHT.Validator = blankValidator{}

	// Set the handler function for when we get a new incoming object stream
	n.Host.SetStreamHandler(OBJECT_STREAM_PROTO, n.objectStreamHandler)

	// Setup the RPC interface so our git helpers can push and pull objects to the network
	// @@TODO: make listen addr configurable (including unix FDs for direct IPC)
	err = n.initRPC("tcp", fmt.Sprintf("127.0.0.1:%v", listenPort+1))
	// err = n.initRPC("unix", "/tmp/conscience-socket")
	if err != nil {
		return nil, err
	}

	return n, nil
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
		return fmt.Errorf("connect to bootstrapper failed: %v", err)
	}

	return nil
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

	// Try to find 1 provider of the object within 10 seconds
	// @@TODO: make timeout configurable
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
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

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

//
// Everything below here is fairly unimportant.
//

func (n *Node) FindProviders(ctx context.Context, contentID string) ([]pstore.PeerInfo, error) {
	c, err := cidForString(contentID)
	if err != nil {
		return nil, err
	}

	chProviders := n.DHT.FindProvidersAsync(ctx, c, 8)

	timeout, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	providers := []pstore.PeerInfo{}
	for {
		select {
		case provider, ok := <-chProviders:
			if !ok {
				break
			}

			if provider.ID == "" {
				log.Printf("got nil provider for %v")
			} else {
				log.Printf("got provider: %+v", provider)
				providers = append(providers, provider)
			}

		case <-timeout.Done():
			break
		}
	}

	return providers, nil
}
