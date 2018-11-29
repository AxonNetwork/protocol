package main

import (
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
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	metrics "github.com/libp2p/go-libp2p-metrics"
	netp2p "github.com/libp2p/go-libp2p-net"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	multihash "github.com/multiformats/go-multihash"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config"
)

func main() {
	const PACK_SIZE = 1024 * 1024 * 20

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

		host.SetStreamHandler("test-proto", func(stream netp2p.Stream) {
			defer stream.Close()
			fmt.Println("[server] transfer starting")

			data, err := ioutil.ReadAll(stream)
			if err != nil {
				panic(err)
			}

			fmt.Printf("[server] transferred %v bytes\n", len(data))
			fmt.Println("[server] finished")
		})

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

		stream, err := host.NewStream(ctx, pinfo.ID, "test-proto")
		if err != nil {
			panic(err)
		}
		defer stream.Close()

		fmt.Println("[client] connected to server")

		f, err := os.Open("/dev/urandom")
		if err != nil {
			panic(err)
		}

		r := io.LimitReader(f, PACK_SIZE)

		n, err := io.Copy(stream, r)
		if err != nil && err != io.EOF {
			panic(err)
		} else if err == io.EOF {
		}
		fmt.Printf("[client] transferred %v bytes\n", n)
		fmt.Println("[client] done")
	}
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
