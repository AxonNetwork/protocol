package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

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
	"gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	dstore "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"
	dsync "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore/sync"
)

const (
	STREAM_PROTO   = "/conscience/chunk-stream/1.0.0"
	CHUNK_HASH_LEN = 32
)

func main() {
	listenPort := os.Args[1]

	talkTo := []string{}
	if len(os.Args) > 2 {
		for _, nodeAddr := range os.Args[2:] {
			talkTo = append(talkTo, nodeAddr)
		}
	}

	ctx := context.Background()

	h, rt := setupNode(ctx, listenPort, talkTo)
	log.Println("peerID is: ", h.ID().Pretty())

	inputLoop(ctx, rt, h)
}

//
// we have to have a validator, so blankValidator just works as a pass-through.
//
type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

func setupNode(ctx context.Context, listenPort string, talkTo []string) (host.Host, *dht.IpfsDHT) {
	h, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", listenPort)),
	)
	if err != nil {
		panic(err)
	}

	ds := dsync.MutexWrap(dstore.NewMapDatastore())
	rt := dht.NewDHT(ctx, h, ds)

	/*validator := record.NamespacedValidator{
	    "pk":   record.PublicKeyValidator{},
	}*/
	rt.Validator = blankValidator{}

	var wg sync.WaitGroup
	for _, multiaddr := range talkTo {
		wg.Add(1)

		go func(multiaddr string) {
			defer wg.Done()

			a, err := ma.NewMultiaddr(multiaddr)
			if err != nil {
				panic(err)
			}

			pinfo, err := pstore.InfoFromP2pAddr(a)
			if err != nil {
				panic(err)
			}

			bef := time.Now()

			err = h.Connect(ctx, *pinfo)
			if err != nil {
				log.Println("connect to bootstrapper failed: ", err)
			}
			log.Printf("Connect(%s) took %s\n", pinfo.ID.Pretty(), time.Since(bef))
		}(multiaddr)
	}
	wg.Wait()

	h.SetStreamHandler(STREAM_PROTO, chunkStreamHandler)

	return h, rt
}

func inputLoop(ctx context.Context, rt *dht.IpfsDHT, h host.Host) {
	fmt.Printf("> ")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		switch parts[0] {
		case "bootstrap":
			bootstrap(ctx, rt)

		case "peers":
			logPeers(h)

		case "get":
			if len(parts) < 2 {
				fmt.Println("not enough args")
				continue
			}
			logValue(ctx, rt, parts[1])

		case "set":
			if len(parts) < 3 {
				fmt.Println("not enough args")
				continue
			}
			setValue(ctx, rt, parts[1], parts[2])

		case "provide":
			if len(parts) < 2 {
				fmt.Println("not enough args")
				continue
			}
			provide(ctx, rt, parts[1])

		case "find-provider":
			if len(parts) < 2 {
				fmt.Println("not enough args")
				continue
			}
			findProvider(ctx, rt, parts[1])

		case "stream":
			if len(parts) < 3 {
				fmt.Println("not enough args")
				continue
			}
			openStream(ctx, h, parts[1], parts[2])

		default:
			fmt.Println("unknown command")
		}

		fmt.Printf("> ")
	}
}

func bootstrap(ctx context.Context, rt *dht.IpfsDHT) {
	err := rt.Bootstrap(ctx)
	if err != nil {
		panic(err)
	}
}

func logPeers(h host.Host) {
	log.Println("total connected peers: ", len(h.Network().Conns()))
	for _, peerID := range h.Peerstore().Peers() {
		fmt.Printf("  - %v (%v)\n", peerID.String(), peer.IDB58Encode(peerID))
		for _, addr := range h.Peerstore().Addrs(peerID) {
			fmt.Println("      -", addr)
		}
	}
}

func logValue(ctx context.Context, rt *dht.IpfsDHT, key string) {
	val, err := rt.GetValue(ctx, key)
	if err != nil {
		log.Printf("key(%v) = nil\n", key)
	} else {
		log.Printf("key(%v) = %v\n", key, string(val))
	}
}

func setValue(ctx context.Context, rt *dht.IpfsDHT, key, val string) {
	bef := time.Now()

	err := rt.PutValue(ctx, key, []byte(val))
	if err != nil {
		log.Println("set failed: ", err)
	}
	fmt.Println("took: ", time.Since(bef))
}

func provide(ctx context.Context, rt *dht.IpfsDHT, repoName string) {
	c, err := cidFromRepoName(repoName)
	if err != nil {
		panic(err)
	}

	err = rt.Provide(ctx, c, true)
	if err != nil {
		panic(err)
	}

	fmt.Println("ok")
}

func findProvider(ctx context.Context, rt *dht.IpfsDHT, repoName string) {
	c, err := cidFromRepoName(repoName)
	if err != nil {
		panic(err)
	}

	chProviders := rt.FindProvidersAsync(ctx, c, 1)
	if err != nil {
		panic(err)
	}

	timeout, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	select {
	case provider := <-chProviders:
		if provider.ID == "" {
			fmt.Println("got nil provider for " + repoName)
		} else {
			fmt.Printf("got provider: %+v\n", provider)
		}

	case <-timeout.Done():
		fmt.Println("timed out, could not find provider of " + repoName)
	}
}

func openStream(ctx context.Context, h host.Host, peerIDB58 string, chunkIDString string) {
	log.Println("[stream] sending chunk...")

	peerID, err := peer.IDB58Decode(peerIDB58)
	if err != nil {
		panic(err)
	}

	// open the stream
	s, err := h.NewStream(ctx, peerID, STREAM_PROTO)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	// write the chunk name to the stream (this also allows us to checksum)
	chunkID, err := hex.DecodeString(chunkIDString)
	if err != nil {
		panic(err)
	}

	n, err := s.Write(chunkID)
	if err != nil {
		panic(err)
	} else if n < 32 {
		panic(fmt.Sprintf("chunk name: wrote the wrong number of bytes: %v", n))
	}

	// write the file data
	f, err := os.Open("./" + chunkIDString)
	if err != nil {
		panic(err)
	}

	nn, err := io.Copy(s, f)
	if err != nil {
		panic(err)
	}

	log.Printf("[stream] sent %v bytes\n", nn)
}

func cidFromRepoName(repoName string) (*cid.Cid, error) {
	pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
	return pref.Sum([]byte(repoName))
}

func chunkStreamHandler(s net.Stream) {
	log.Println("[stream] beginning")

	// read the chunk hash name
	chunkID := make([]byte, 32)
	n, err := s.Read(chunkID)
	if err != nil {
		panic(err)
	} else if n < 32 {
		panic("didn't receive enough bytes for chunk name header")
	}

	f, err := os.Create("/tmp/" + hex.EncodeToString(chunkID))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// copy the data from the p2p stream into the file while calculating
	// the sha256 hash of the data for checksumming
	hasher := sha256.New()
	reader := io.TeeReader(s, hasher)
	nn, err := io.Copy(f, reader)
	if err != nil {
		panic(err)
	}

	fmt.Println("chunk bytes received:", nn)

	hash := hasher.Sum(nil)

	if !bytes.Equal(chunkID, hash) {
		fmt.Printf("checksums are not equal (%v and %v)\n", hex.EncodeToString(chunkID), hex.EncodeToString(hash))
	}

	log.Println("[stream] finished")
}
