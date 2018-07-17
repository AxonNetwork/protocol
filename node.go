package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

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
	"gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
	dstore "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"
	dsync "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore/sync"
)

type Node struct {
	Host host.Host
	DHT  *dht.IpfsDHT
}

const (
	STREAM_PROTO   = "/conscience/chunk-stream/1.0.0"
	CHUNK_HASH_LEN = 32
)

func NewNode(ctx context.Context, listenPort string) (*Node, error) {
	h, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", listenPort)),
	)
	if err != nil {
		return nil, err
	}

	n := &Node{
		Host: h,
		DHT:  dht.NewDHT(ctx, h, dsync.MutexWrap(dstore.NewMapDatastore())),
	}

	// set a pass-through validator
	n.DHT.Validator = blankValidator{}

	// set the handler function for when we get a new incoming chunk stream
	n.Host.SetStreamHandler(STREAM_PROTO, n.chunkStreamHandler)

	return n, nil
}

func (n *Node) Bootstrap(ctx context.Context) error {
	return n.DHT.Bootstrap(ctx)
}

func (n *Node) LogPeers() error {
	log.Printf("total connected peers: %v", len(n.Host.Network().Conns()))

	for _, peerID := range n.Host.Peerstore().Peers() {
		log.Printf("  - %v (%v)", peerID.String(), peer.IDB58Encode(peerID))
		for _, addr := range n.Host.Peerstore().Addrs(peerID) {
			log.Printf("      - %v", addr)
		}
	}
	return nil
}

func (n *Node) AddPeer(ctx context.Context, multiaddrString string) error {
	// The following code extracts the peer ID from the
	// given multiaddress
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

func (n *Node) GetValue(ctx context.Context, key string) error {
	val, err := n.DHT.GetValue(ctx, key)
	if err != nil {
		log.Printf("%v: nil", key)
	} else {
		log.Printf("%v: %v", key, string(val))
	}
	return nil
}

func (n *Node) SetValue(ctx context.Context, key, val string) error {
	return n.DHT.PutValue(ctx, key, []byte(val))
}

func (n *Node) Provide(ctx context.Context, repoName string) error {
	c, err := cidFromRepoName(repoName)
	if err != nil {
		return err
	}

	return n.DHT.Provide(ctx, c, true)
}

func (n *Node) FindProviders(ctx context.Context, repoName string) error {
	c, err := cidFromRepoName(repoName)
	if err != nil {
		return err
	}

	chProviders := n.DHT.FindProvidersAsync(ctx, c, 100)

	timeout, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	select {
	case provider := <-chProviders:
		if provider.ID == "" {
			log.Printf("got nil provider for %v")
		} else {
			log.Printf("got provider: %+v", provider)
		}

	case <-timeout.Done():
		return fmt.Errorf("timed out, could not find provider of %v", repoName)
	}

	return nil
}

func (n *Node) SendChunkStream(ctx context.Context, peerIDB58 string, chunkIDString string) error {
	log.Printf("[stream] sending chunk...")

	peerID, err := peer.IDB58Decode(peerIDB58)
	if err != nil {
		return err
	}

	// open the stream
	s, err := n.Host.NewStream(ctx, peerID, STREAM_PROTO)
	if err != nil {
		return err
	}
	defer s.Close()

	// write the chunk name to the stream (this also allows us to checksum)
	chunkID, err := hex.DecodeString(chunkIDString)
	if err != nil {
		return err
	}

	written, err := s.Write(chunkID)
	if err != nil {
		return err
	} else if written < 32 {
		return fmt.Errorf("chunk name: wrote the wrong number of bytes: %v", n)
	}

	// write the file data
	f, err := os.Open("./" + chunkIDString)
	if err != nil {
		return err
	}

	copied, err := io.Copy(s, f)
	if err != nil {
		return err
	}

	log.Printf("[stream] sent %v bytes", copied)

	return nil
}

func (n *Node) chunkStreamHandler(stream net.Stream) {
	log.Println("[stream] beginning")

	// read the chunk hash name
	chunkID := make([]byte, 32)
	recvd, err := stream.Read(chunkID)
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	} else if recvd < 32 {
		log.Errorf("[stream] didn't receive enough bytes for chunk name header")
		return
	}

	file, err := os.Create("/tmp/" + hex.EncodeToString(chunkID))
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}
	defer file.Close()

	// copy the data from the p2p stream into the file and the SHA256 hasher simultaneously.
	// use the sha256 hash of the data for checksumming.
	hasher := sha256.New()
	reader := io.TeeReader(stream, hasher)
	recvd2, err := io.Copy(file, reader)
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}

	log.Printf("[stream] chunk bytes received: %v", recvd2)

	hash := hasher.Sum(nil)

	if !bytes.Equal(chunkID, hash) {
		log.Errorf("[stream] checksums are not equal (%v and %v)", hex.EncodeToString(chunkID), hex.EncodeToString(hash))
	}

	log.Printf("[stream] finished")
}

//
// blankValidator just works as a pass-through
//
type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }
