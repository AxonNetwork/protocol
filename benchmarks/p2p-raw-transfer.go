package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

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
	ma "github.com/multiformats/go-multiaddr"
	multihash "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config"
	. "github.com/Conscience/protocol/swarm/wire"
)

const MB = 1024 * 1024
const PACK_MULTIPLIER = 64
const PACK_SIZE = MB * PACK_MULTIPLIER
const CHUNK_MULTIPLIER = 1
const CHUNK_SIZE = MB * CHUNK_MULTIPLIER

type StreamChunk struct {
	End     bool
	DataLen int `struc:"sizeof=Data"`
	Data    []byte
}

func main() {
	var server bool
	if len(os.Args) < 2 || os.Args[1] == "" {
		server = true
	}

	ctx := context.Background()

	bandwidthCounter := metrics.NewBandwidthCounter()

	var key crypto.PrivKey
	var port int
	if server {
		key = obtainKey()
		port = 9991
	} else {
		key = makeKey()
		port = 9992
	}

	// Initialize the p2p host
	host, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/%v/tcp/%v", "0.0.0.0", port),
		),
		libp2p.Identity(key),
		libp2p.NATPortMap(),
		libp2p.BandwidthReporter(bandwidthCounter),
	)
	if err != nil {
		panic(err)
	}

	// Initialize the DHT
	d := dht.NewDHT(ctx, host, dsync.MutexWrap(dstore.NewMapDatastore()))
	d.Validator = blankValidator{} // Set a pass-through validator

	if server {
		for _, addr := range host.Addrs() {
			fmt.Println(addr.String() + "/ipfs/" + host.ID().Pretty())
		}

		host.SetStreamHandler("test-file-proto", testFileServer)
		host.SetStreamHandler("test-chunk-proto", testFileServer)
		host.SetStreamHandler("test-struct-proto", testStructServer)

		go periodicallyAnnounceContent(ctx, d)

		select {}

	} else {
		addr, err := ma.NewMultiaddr(os.Args[1])
		if err != nil {
			panic(errors.Wrapf(err, "could not parse multiaddr '%v'", os.Args[1]).Error())
		}

		pinfo, err := pstore.InfoFromP2pAddr(addr)
		if err != nil {
			panic(errors.Wrapf(err, "could not parse PeerInfo from multiaddr '%v'", os.Args[1]).Error())
		}

		err = host.Connect(ctx, *pinfo)
		if err != nil {
			panic(errors.Wrapf(err, "could not connect to peer '%v'", os.Args[1]).Error())
		}

		fileTime, err := testFileClient(host, pinfo.ID)
		if err != nil {
			panic(err)
		}

		chunkTime, err := testChunkClient(host, pinfo.ID)
		if err != nil {
			panic(err)
		}

		structTime, err := testStructClient(host, pinfo.ID)
		if err != nil {
			panic(err)
		}

		fmt.Println("[client] done")

		fmt.Printf("File Size: %vMB\n", PACK_MULTIPLIER)
		fmt.Printf("Chunk Size: %vMB\n", CHUNK_MULTIPLIER)
		fmt.Println("Full File Transfer time: ", fileTime)
		fmt.Println("Chunked File Transfer time: ", chunkTime)
		fmt.Println("Struct Chunked File Transfer time: ", structTime)
	}
}

func testFileServer(stream netp2p.Stream) {
	defer stream.Close()
	fmt.Println("[server] transfer starting")

	data := make([]byte, PACK_SIZE)
	var total int
	for {
		n, err := io.ReadFull(stream, data)
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			continue
		} else if err != nil {
			panic(err)
		}

		total += n
	}

	fmt.Printf("[server] read %v bytes\n", total)
	fmt.Println("[server] finished")
}

func testStructServer(stream netp2p.Stream) {
	defer stream.Close()
	fmt.Println("[server] transfer starting")

	var total int
	for {
		chunk := StreamChunk{}
		err := ReadStructPacket(stream, &chunk)
		if err != nil {
			fmt.Println("err: ", err)
			return
		}
		total += len(chunk.Data)
		if chunk.End {
			break
		}
	}
	fmt.Printf("[server] read %v bytes\n", total)
	fmt.Println("[server] finished")
}

func testFileClient(host host.Host, id peer.ID) (time.Duration, error) {
	start := time.Now()

	ctx := context.Background()
	stream, err := host.NewStream(ctx, id, "test-file-proto")
	if err != nil {
		return 0, err
	}
	defer stream.Close()

	fmt.Println("[client] connected to server")

	f, err := os.Open("/dev/urandom")
	if err != nil {
		return 0, err
	}

	r := io.LimitReader(f, PACK_SIZE)

	n, err := io.Copy(stream, r)
	if err != nil && err != io.EOF {
		return 0, err
	} else if err == io.EOF {
	}
	fmt.Printf("[client] transferred %v bytes\n", n)
	fmt.Println("[client] done")

	total := time.Now().Sub(start)
	return total, nil
}

func testChunkClient(host host.Host, id peer.ID) (time.Duration, error) {
	start := time.Now()

	ctx := context.Background()
	stream, err := host.NewStream(ctx, id, "test-chunk-proto")
	if err != nil {
		return 0, err
	}
	defer stream.Close()

	fmt.Println("[client] connected to server")

	f, err := os.Open("/dev/urandom")
	if err != nil {
		return 0, err
	}

	r := io.LimitReader(f, PACK_SIZE)
	data := make([]byte, CHUNK_SIZE)
	transferred := 0
	for {
		n, err := io.ReadFull(r, data)
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			data = data[:n]
		} else if err != nil {
			return 0, err
		}
		transferred += n

		buf := bytes.NewBuffer(data)
		_, err = buf.WriteTo(stream)
		if err != nil {
			return 0, err
		}
	}
	fmt.Printf("[client] transferred %v bytes\n", transferred)

	total := time.Now().Sub(start)
	return total, nil
}

func testStructClient(host host.Host, id peer.ID) (time.Duration, error) {
	start := time.Now()

	ctx := context.Background()
	stream, err := host.NewStream(ctx, id, "test-struct-proto")
	if err != nil {
		return 0, err
	}
	defer stream.Close()

	fmt.Println("[client] connected to server")

	f, err := os.Open("/dev/urandom")
	if err != nil {
		return 0, err
	}

	r := io.LimitReader(f, PACK_SIZE)
	data := make([]byte, CHUNK_SIZE)
	transferred := 0
	end := false
	for {
		n, err := io.ReadFull(r, data)
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			data = data[:n]
			end = true
		} else if err != nil {
			return 0, err
		}
		err = WriteStructPacket(stream, &StreamChunk{End: end, Data: data})
		if err != nil {
			return 0, err
		}
		transferred += n
	}
	fmt.Printf("[client] transferred %v bytes\n", transferred)

	total := time.Now().Sub(start)
	return total, nil
}

var fakeRepos = []string{"one", "two", "three", "four"}

func periodicallyAnnounceContent(ctx context.Context, dht *dht.IpfsDHT) {
	c := time.Tick(15 * time.Second)
	for range c {
		fmt.Println("[content announce] starting content announce")

		for i := range fakeRepos {
			c, err := cidForString(fakeRepos[i])
			if err != nil {
				panic(err)
			}

			err = dht.Provide(ctx, c, true)
			if err != nil && err != kbucket.ErrLookupFailure {
				panic(err)
			}
		}
	}
}

func cidForString(s string) (cid.Cid, error) {
	pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
	c, err := pref.Sum([]byte(s))
	if err != nil {
		return cid.Cid{}, errors.Wrap(err, "could not create cid")
	}
	return c, nil
}

func obtainKey() crypto.PrivKey {
	data, err := ioutil.ReadFile(filepath.Join(config.HOME, ".conscience.key"))
	if err != nil {
		return makeKey()
	}

	k, err := crypto.UnmarshalPrivateKey(data)
	if err != nil {
		panic(err)
	}

	return k
}

func makeKey() crypto.PrivKey {
	privkey, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 0)
	if err != nil {
		panic(err)
	}
	return privkey
}

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }
