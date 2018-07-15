package main

import (
	"context"
	"fmt"
	//"io"
	"log"
	"os"
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
	//peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
	//ic "gx/ipfs/Qme1knMqwt1hKZbc1BmQFmnm9f36nyQGwXxPGVpVJ9rMK5/go-libp2p-crypto"
	dstore "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore"
	dsync "gx/ipfs/QmeiCcJfDW1GJnWUArudsv5rQsihpi4oyddPhdqo3CfX6i/go-datastore/sync"
)

var bsaddrs = []string{
	/*"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
	"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
	"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
	"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",*/
}

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

func setupNode(ctx context.Context, listenPort string, talkTo []string) (host.Host, *dht.IpfsDHT) {
	h, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%s", listenPort)),
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
	for _, bsa := range talkTo {
		wg.Add(1)
		go func(bsa string) {
			defer wg.Done()
			a, err := ma.NewMultiaddr(bsa)
			if err != nil {
				panic(err)
			}

			pinfo, err := pstore.InfoFromP2pAddr(a)
			if err != nil {
				panic(err)
			}

			bef := time.Now()
			if err := h.Connect(ctx, *pinfo); err != nil {
				fmt.Println("connect to bootstrapper failed: ", err)
			}
			fmt.Printf("Connect(%s) took %s\n", pinfo.ID.Pretty(), time.Since(bef))
		}(bsa)
	}
	wg.Wait()
	return h, rt
}

func main() {
	/*el, err := NewEventsLogger("events")
	if err != nil {
		panic(err)
	}

	r, w := io.Pipe()

	go el.handleEvents(r)
	writer.WriterGroup.AddWriter(w)*/

	ctx := context.Background()

	listenPort := os.Args[1]

	talkTo := []string{}
	if len(os.Args) > 2 {
		for _, port := range os.Args[2:] {
			talkTo = append(talkTo, "/ip4/127.0.0.1/tcp/" + port)
		}
	}

	h, rt := setupNode(ctx, listenPort, talkTo)
	go logPeers(h)
	go logValue(ctx, rt, "/v/key-1")

	fmt.Println("peerID is: ", h.ID().Pretty())

	if len(talkTo) > 0 {
		tryAddingValue(ctx, rt)
	}
	select{}
}

func logPeers(h host.Host) {
	c := time.Tick(5 * time.Second)
	for range c {
		fmt.Println("total connected peers: ", len(h.Network().Conns()))
	}
}

func logValue(ctx context.Context, rt *dht.IpfsDHT, key string) {
	c := time.Tick(5 * time.Second)
	for range c {
		val, err := rt.GetValue(ctx, key)
		if err != nil {
			fmt.Printf("key(%v) = nil\n", key)
		} else {
			fmt.Printf("key(%v) = %v\n", key, val)
		}
	}
}

func tryAddingValue(ctx context.Context, rt *dht.IpfsDHT) {
	//var snaps [][]*PeerDialLog
	var times []time.Duration

	for i := 0; i < 5; i++ {
		log.Println("starting put")
		bef := time.Now()
		if err := rt.PutValue(ctx, fmt.Sprintf("/v/key-%v", i), []byte("brynnish")); err != nil {
			log.Println("put failed: ", err)
		}
		took := time.Since(bef)
		fmt.Println("took: ", time.Since(bef))
		times = append(times, took)

		/*dials := el.Dials
		fmt.Println("total dials: ", len(dials))
		snaps = append(snaps, dials)
		el.Dials = nil
		el.DialsByParent = make(map[uint64][]DialAttempt) */
	}

	/*for i, s := range snaps {
		fmt.Println(times[i], len(s))
	}*/


	valout, err := rt.GetValue(ctx, "/v/key-1")
	if err != nil {
		fmt.Println("getvalue after put failed: ", err)
	}
	fmt.Println(valout)
}
