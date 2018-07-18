package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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
	STREAM_PROTO   = "/conscience/chunk-stream/1.0.0"
	CHUNK_HASH_LEN = 32
)

func NewNode(ctx context.Context, listenPort string) (*Node, error) {
	h, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", listenPort)),
		libp2p.NATPortMap(),
	)
	if err != nil {
		return nil, err
	}

	rm, err := NewRepoManager("./repos")
	if err != nil {
		return nil, err
	}

	n := &Node{
		Host:        h,
		DHT:         dht.NewDHT(ctx, h, dsync.MutexWrap(dstore.NewMapDatastore())),
		RepoManager: rm,
	}

	// start a goroutine for announcing which repos this Node can provide every few seconds
	// @@TODO: make announce interval configurable
	go func() {
		c := time.Tick(5 * time.Second)
		for range c {
			repoNames := rm.RepoNames()
			for _, repoName := range repoNames {
				log.Printf("[announce] %v", repoName)

				err := n.Provide(ctx, repoName)
				if err != nil && err != kbucket.ErrLookupFailure {
					log.Errorf("[announce] %v", err)
				}
			}
		}
	}()

	// set a pass-through validator
	n.DHT.Validator = blankValidator{}

	// set the handler function for when we get a new incoming chunk stream
	n.Host.SetStreamHandler(STREAM_PROTO, n.chunkStreamHandler)

	// Register Node on RPC to listen to procedure from git push/pull hooks
	// Only listen to calls from localhost
	port, err := incrementPort(listenPort)
	if err != nil {
		panic(err)
	}
	err = RegisterRPC(n, port)
	if err != nil {
		return nil, err
	}

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

func (n *Node) FindProviders(ctx context.Context, repoName string) ([]pstore.PeerInfo, error) {
	c, err := cidFromRepoName(repoName)
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

func (n *Node) GetRepo(ctx context.Context, peerIDB58 string, repoName string) (bool, error) {
	log.Printf("[stream] requesting repo...")

	recentHash := "0983d41ec8d4732136f4b3a5c2cb45a27a18436b" // TDOO: get this from single source of truth

	success, err := n.CrawlGitTree(recentHash, ctx, peerIDB58, repoName)

	return success, err
}

func (n *Node) CrawlGitTree(sha1 string, ctx context.Context, peerIDB58 string, repoName string) (bool, error) {
	log.Printf("crawling hash: %s", sha1)

	objType, err := n.RepoManager.GitCatKind(sha1, repoName)
	if err != nil {
		return false, err
	}
	
	log.Printf("object Type: %s", objType)
	if objType == "tree" {
		log.Printf("is a tree!")
		objects, err := n.RepoManager.GitListObjects(sha1, repoName)

		log.Printf("objects: %v: ", objects)
		if err != nil {
			return false, err
		}

		// 	Recurse
		for _, obj := range objects {
			if (obj != sha1) {
				log.Printf("object: %v", obj)
				n.CrawlGitTree(obj, ctx, peerIDB58, repoName)
				return true, nil
			}
		}
	}

	n.GetChunk(ctx, peerIDB58, repoName, sha1)

	return true, nil
}

func (n *Node) GetChunk(ctx context.Context, peerIDB58 string, repoName string, chunkIDString string) (bool, error) {
	log.Printf("[stream] requesting chunk...")

	peerID, err := peer.IDB58Decode(peerIDB58)
	if err != nil {
		return false, err
	}

	// open the stream
	stream, err := n.Host.NewStream(ctx, peerID, STREAM_PROTO)
	if err != nil {
		return false, err
	}
	defer stream.Close()

	//
	// 1. write the repo name and chunk ID to the stream
	//
	chunkID, err := hex.DecodeString(chunkIDString)
	if err != nil {
		return false, err
	}

	msg := append([]byte(repoName), 0x0)
	msg = append(msg, chunkID...)
	msg = append(msg, 0x0)
	_, err = stream.Write(msg)
	if err != nil {
		return false, err
	}

	//
	// 2. if the reply is 0x0, the peer doesn't have the chunk.  if it's 0x1, stream the chunk and save to disk.
	//
	reply := make([]byte, 1)
	recvd, err := stream.Read(reply)
	if err != nil {
		return false, err
	} else if recvd < 1 {
		return false, fmt.Errorf("empty reply from chunk request")
	}

	if reply[0] == 0x0 {
		return false, nil
	} else if reply[0] != 0x1 {
		return false, fmt.Errorf("bad reply from chunk request: %v", reply[0])
	}

	chunk, err := n.RepoManager.CreateChunk(repoName, hex.EncodeToString(chunkID))
	if err != nil {
		return false, err
	}
	defer chunk.Close()

	// copy the data from the p2p stream into the file and the SHA256 hasher simultaneously.
	// use the sha256 hash of the data for checksumming.
	hasher := sha256.New()
	reader := io.TeeReader(stream, hasher)
	recvd2, err := io.Copy(chunk, reader)
	if err != nil {
		return false, err
	}

	log.Printf("[stream] chunk bytes received: %v", recvd2)

	hash := hasher.Sum(nil)

	if !bytes.Equal(chunkID, hash) {
		log.Errorf("[stream] checksums are not equal (%v and %v)", hex.EncodeToString(chunkID), hex.EncodeToString(hash))
	}

	log.Printf("[stream] finished")
	return true, nil
}

func (n *Node) chunkStreamHandler(stream netp2p.Stream) {
	defer stream.Close()

	// create a buffered reader so we can read up until certain byte delimiters
	bufstream := bufio.NewReader(stream)

	var repoName, chunkIDStr string

	//
	// read the repo name, terminated by a null byte
	//
	{
		repoNameBytes, err := bufstream.ReadBytes(0x0)
		if err == io.EOF {
			log.Errorf("[stream] terminated early")
			return
		} else if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}
		repoNameBytes = repoNameBytes[:len(repoNameBytes)-1] // hack off the null byte at the end
		repoName = string(repoNameBytes)
	}

	//
	// read the chunk ID, terminated by a null byte
	//
	{
		chunkID, err := bufstream.ReadBytes(0x0)
		if err == io.EOF {
			log.Errorf("[stream] terminated early")
			return
		} else if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}
		chunkID = chunkID[:len(chunkID)-1] // hack off the null byte at the end
		chunkIDStr = hex.EncodeToString(chunkID)
	}

	log.Printf("[stream] peer requested %v %v", repoName, chunkIDStr)

	//
	// send our response:
	// 1. we don't have the chunk:
	//    - 0x0
	//    - <close connection>
	// 2. we do have the chunk:
	//    - 0x1
	//    - [chunk bytes...]
	//    - <close connection>
	//
	if !n.RepoManager.HasChunk(repoName, chunkIDStr) {
		log.Printf("[stream] we don't have %v %v", repoName, chunkIDStr)

		// tell the peer we don't have the chunk and then close the connection
		_, err := stream.Write([]byte{0x0})
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}
		return
	}

	_, err := stream.Write([]byte{0x1})
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}

	chunk, err := n.RepoManager.OpenChunk(repoName, chunkIDStr)
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}
	defer chunk.Close()

	sent, err := io.Copy(stream, chunk)
	if err != nil {
		log.Errorf("[stream] %v", err)
	}

	log.Printf("[stream] sent %v %v (%v bytes)", repoName, chunkIDStr, sent)
}

func (this *Node) GitPush(remoteName string, remoteUrl string, branch string, commit string) error {
	fmt.Printf("\n******************\n")
	fmt.Printf("Git Push:\n")
	fmt.Println("remoteName: ", remoteName)
	fmt.Println("remoteUrl: ", remoteUrl)
	fmt.Println("branch: ", branch)
	fmt.Println("commit: ", commit)
	fmt.Printf("******************\n")
	return nil
}

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }
