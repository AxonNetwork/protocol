package swarm

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"reflect"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"gx/ipfs/QmQYwRL1T9dJtdCScoeRQwwvScbJTcWqnXhq4dYQ6Cu5vX/go-libp2p-kad-dht"
	//"gx/ipfs/QmVsp2KdPYE6M8ryzCk5KHLo3zprcY5hBDaYx6uPCFUdxA/go-libp2p-record"
	ma "gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
	//proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	"gx/ipfs/QmZ86eLPtXkQ1Dfa992Q8NpXArUoWWh3y728JDcWvzRrvC/go-libp2p"
	pstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	"gx/ipfs/Qmb8T6YBBsjYsVGfrihQLfCJveczZnneSBqBKkYEBWDjge/go-libp2p-host"
	//writer "gx/ipfs/QmcVVHfdyv15GVPk7NrxdWjh2hLVccXnoD8j2tyQShiXJb/go-log/writer"
	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
	//ic "gx/ipfs/Qme1knMqwt1hKZbc1BmQFmnm9f36nyQGwXxPGVpVJ9rMK5/go-libp2p-crypto"
	inet "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
	netp2p "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
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

	// set a pass-through validator
	n.DHT.Validator = blankValidator{}

	// set the handler function for when we get a new incoming object stream
	n.Host.SetStreamHandler(OBJECT_STREAM_PROTO, n.objectStreamHandler)

	// Register Node on RPC to listen to procedure from git push/pull hooks
	// Only listen to calls from localhost
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
func (n *Node) GetObjectReader(ctx context.Context, repoID string, objectID []byte) (io.ReadCloser, gitplumbing.ObjectType, int64, error) {
	// If we detect that we already have the object locally, just open a regular file stream
	if n.RepoManager.HasObject(repoID, objectID) {
		return n.openLocalObjectReader(repoID, objectID)
	}

	c, err := cidForObject(repoID, objectID)
	if err != nil {
		return nil, 0, 0, err
	}

	// Try to find 1 provider of the object within 10 seconds
	// @@TODO: make timeout configurable
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	provider, found := <-n.DHT.FindProvidersAsync(ctxTimeout, c, 1)
	if !found {
		return nil, 0, 0, errors.New("can't find peer with object " + repoID + ":" + hex.EncodeToString(objectID))
	}

	if provider.ID == n.Host.ID() {
		// We have the object locally (perhaps in another clone of the same repository)
		return n.openLocalObjectReader(repoID, objectID)

	} else {
		// We found a peer with the object
		return n.openPeerObjectReader(ctx, provider.ID, repoID, objectID)
	}
}

func (n *Node) openLocalObjectReader(repoID string, objectID []byte) (io.ReadCloser, gitplumbing.ObjectType, int64, error) {
	return n.RepoManager.OpenObject(repoID, objectID)
}

func (n *Node) openPeerObjectReader(ctx context.Context, peerID peer.ID, repoID string, objectID []byte) (io.ReadCloser, gitplumbing.ObjectType, int64, error) {
	log.Printf("[stream] requesting object...")

	// Open the stream
	stream, err := n.Host.NewStream(ctx, peerID, OBJECT_STREAM_PROTO)
	if err != nil {
		return nil, 0, 0, err
	}

	//
	// 1. Write the repo name and object ID to the stream.
	//

	repoIDLen := make([]byte, 8)
	objectIDLen := make([]byte, 8)
	binary.LittleEndian.PutUint64(repoIDLen, uint64(len(repoID)))
	binary.LittleEndian.PutUint64(objectIDLen, uint64(len(objectID)))

	msg := append(repoIDLen, []byte(repoID)...)
	msg = append(msg, objectIDLen...)
	msg = append(msg, objectID...)
	// msg = append(msg, 0x0)
	_, err = stream.Write(msg)
	if err != nil {
		return nil, 0, 0, err
	}

	//
	// 2. If the reply is 0x0, the peer doesn't have the object.  If the reply is 0x1, it does.
	//
	reply := make([]byte, 1)
	recvd, err := stream.Read(reply)
	if err != nil {
		return nil, 0, 0, err
	} else if recvd < 1 {
		return nil, 0, 0, errors.WithStack(ErrProtocol)
	}

	if reply[0] == 0x0 {
		return nil, 0, 0, errors.New("peer doesn't have object " + repoID + ":" + hex.EncodeToString(objectID))
	} else if reply[0] != 0x1 {
		return nil, 0, 0, errors.WithStack(ErrProtocol)
	}

	//
	// 3. Read the object type.  This only matters for Git objects.  Conscience objects use 0x0.
	//
	recvd, err = stream.Read(reply)
	if err != nil {
		return nil, 0, 0, err
	} else if recvd < 1 {
		return nil, 0, 0, errors.WithStack(ErrProtocol)
	}

	objectType := gitplumbing.ObjectType(reply[0])
	if objectType < 0 || objectType > 7 {
		return nil, 0, 0, errors.WithStack(ErrProtocol)
	}

	objectLen, err := readUint64(stream)
	if err != nil {
		return nil, 0, 0, err
	}

	return stream, objectType, int64(objectLen), nil
}

func (n *Node) objectStreamHandler(stream netp2p.Stream) {
	defer stream.Close()

	// create a buffered reader so we can read up until certain byte delimiters
	// bufstream := bufio.NewReader(stream)

	var repoID, objectIDStr string
	var objectID []byte
	var err error

	//
	// read the repo ID
	//
	{
		repoIDLen, err := readUint64(stream)
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}

		repoIDBytes := make([]byte, repoIDLen)
		_, err = io.ReadFull(stream, repoIDBytes)
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}

		repoID = string(repoIDBytes)
	}

	//
	// read the object ID
	//
	{
		objectIDLen, err := readUint64(stream)
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}

		objectID = make([]byte, objectIDLen)
		_, err = io.ReadFull(stream, objectID)
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}

		objectIDStr = hex.EncodeToString(objectID)
	}

	log.Printf("[stream] peer requested %v %v", repoID, objectIDStr)

	//
	// send our response:
	// 1. we don't have the object:
	//    - 0x0
	//    - <close connection>
	// 2. we do have the object:
	//    - 0x1
	//    - [object type byte, only matters for Git objects]
	//    - [object length]
	//    - [object bytes...]
	//    - <close connection>
	//
	object, exists := n.RepoManager.Object(repoID, objectID)
	if !exists {
		log.Printf("[stream] we don't have %v %v", repoID, objectIDStr)

		// tell the peer we don't have the object and then close the connection
		_, err := stream.Write([]byte{0x0})
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}
		return
	}

	_, err = stream.Write([]byte{0x1, byte(object.Type)})
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}

	objectData, _, objectLen, err := n.RepoManager.OpenObject(repoID, objectID)
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}
	defer objectData.Close()

	err = writeUint64(stream, uint64(objectLen))
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}

	sent, err := io.Copy(stream, objectData)
	if err != nil {
		log.Errorf("[stream] %v", err)
	}

	if sent < objectLen {
		log.Errorf("[stream] terminated while sending")
	}

	log.Printf("[stream] sent %v %v (%v bytes)", repoID, objectIDStr, sent)
}

func (n *Node) initRPC(network, addr string) error {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Errorf("[rpc stream] %v", err)
			} else {
				go n.rpcStreamHandler(conn)
			}
		}
	}()

	return nil
}

func (n *Node) rpcStreamHandler(stream io.ReadWriteCloser) {
	log.Printf("[rpc stream] opening new stream")

	defer func() {
		stream.Close()
		log.Printf("[rpc stream] closing stream")
	}()

	msgType, err := readUint64(stream)
	if err != nil {
		panic(err)
	}

	log.Printf("[rpc stream] msgType = %v", msgType)

	switch RPCMessageType(msgType) {
	case RPCMessageType_GetObject:
		log.Printf("[rpc stream] GetObject")
		req := GetObjectRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc stream] %+v", req)

		objectReader, objectType, objectLen, err := n.GetObjectReader(context.Background(), req.RepoID, req.ObjectID)
		// @@TODO: maybe don't assume err == 404
		if err != nil {
			log.Printf("[rpc stream] don't have object: %v", err)
			err = writeStructPacket(stream, &GetObjectResponse{HasObject: false})
			if err != nil {
				panic(err)
			}
			return
		}

		log.Printf("[rpc stream] do have object")
		err = writeStructPacket(stream, &GetObjectResponse{
			HasObject:  true,
			ObjectType: objectType,
			ObjectLen:  objectLen,
		})
		if err != nil {
			panic(err)
		}

		// Write the object
		n, err := io.Copy(stream, objectReader)
		if err != nil {
			panic(err)
		} else if n < objectLen {
			panic("n < objectLen")
		}

	default:
		log.Errorf("Node.rpcStreamHandler: bad message type from peer (%v)", msgType)
	}
}

func (this *Node) PushHook(remoteName string, remoteUrl string, branch string, commit string) error {
	fmt.Printf("\n******************\n")
	fmt.Printf("Push Hook:\n")
	fmt.Println("remoteName: ", remoteName)
	fmt.Println("remoteUrl: ", remoteUrl)
	fmt.Println("branch: ", branch)
	fmt.Println("commit: ", commit)
	fmt.Printf("******************\n")
	return nil
}

func (this *Node) ListHelper(repoID string, objectID []byte) (inet.Stream, error) {
	fmt.Printf("\n******************\n")
	fmt.Printf("ListHelper:\n")
	fmt.Println("repoID: ", repoID)
	fmt.Println("objectID: ", objectID)

	//@TODO: refactor with GetObject
	c, err := cidForObject(repoID, objectID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	provider, found := <-this.DHT.FindProvidersAsync(ctxTimeout, c, 1)
	if !found {
		return nil, errors.WithStack(ErrObjectNotFound)
	}

	objectType, stream, _, err := this.openPeerObjectReader(ctx, provider.ID, repoID, objectID)

	fmt.Printf("objectType: %v", objectType)
	fmt.Printf("stream: %v", reflect.TypeOf(stream))

	fmt.Printf("******************\n")
	return nil, nil
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
