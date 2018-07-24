package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
	"gx/ipfs/QmYVNvtQkeZ6AKSwDrjQTs432QtL6umrrK41EBq3cu7iSP/go-cid"
	"gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"

	swarm ".."
	"../../config"
)

func main() {
	ctx := context.Background()

	// Read the config file in the user's homedir
	cfg, err := config.ReadConfig()
	if err != nil {
		panic(err)
	}

	// Allow overriding the P2P listen port from the command line
	if len(os.Args) >= 2 {
		listenPort, err := strconv.ParseUint(os.Args[1], 10, 64)
		if err != nil {
			panic("usage: swarm [p2p port] [rpc port]")
		}
		cfg.Node.P2PListenPort = int(listenPort)
	}

	// Allow overriding the RPC listen port from the command line
	if len(os.Args) >= 3 {
		listenPort, err := strconv.ParseUint(os.Args[2], 10, 64)
		if err != nil {
			panic("usage: swarm [p2p port] [rpc port]")
		}
		cfg.Node.RPCListenNetwork = "tcp"
		cfg.Node.RPCListenHost = fmt.Sprintf("127.0.0.1:%v", listenPort)
	}

	n, err := swarm.NewNode(ctx, cfg)
	if err != nil {
		panic(err)
	}

	inputLoop(ctx, n)
}

func inputLoop(ctx context.Context, n *swarm.Node) {
	fmt.Printf("> ")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		var err error

		switch parts[0] {
		case "addrs":
			logAddrs(n)

		case "repos":
			logRepos(n)

		case "peers":
			logPeers(n)

		case "config":
			logConfig(n)

		case "add-repo":
			if len(parts) < 2 {
				err = fmt.Errorf("not enough args")
				break
			}
			err = n.RepoManager.AddRepo(parts[1])

		case "add-peer":
			if len(parts) < 2 {
				err = fmt.Errorf("not enough args")
				break
			}
			err = n.AddPeer(ctx, parts[1])

		case "get":
			if len(parts) < 2 {
				err = fmt.Errorf("not enough args")
				break
			}
			key := parts[1]
			val, err := n.DHT.GetValue(ctx, key)
			if err != nil {
				log.Printf("%v: nil", key)
			} else {
				log.Printf("%v: %v", key, string(val))
			}

		case "set":
			if len(parts) < 3 {
				err = fmt.Errorf("not enough args")
				break
			}
			err = n.DHT.PutValue(ctx, parts[1], []byte(parts[2]))

		case "provide":
			if len(parts) < 2 {
				err = fmt.Errorf("not enough args")
				break
			}
			var c *cid.Cid
			pref := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256)
			c, err = pref.Sum([]byte(parts[1]))
			if err != nil {
				break
			}
			err = n.DHT.Provide(ctx, c, true)

		case "find-providers":
			if len(parts) < 2 {
				err = fmt.Errorf("not enough args")
				break
			}
			_, err = n.FindProviders(ctx, parts[1])

		default:
			err = fmt.Errorf("unknown command")
		}

		if err != nil {
			log.Errorln(err)
		}

		fmt.Printf("> ")
	}
}

func logPeers(n *swarm.Node) {
	log.Printf("total connected peers: %v", len(n.Host.Network().Conns()))

	for _, peerID := range n.Host.Peerstore().Peers() {
		log.Printf("  - %v (%v)", peerID.String(), peer.IDB58Encode(peerID))
		for _, addr := range n.Host.Peerstore().Addrs(peerID) {
			log.Printf("      - %v", addr)
		}
	}
}

func logAddrs(n *swarm.Node) {
	for _, addr := range n.Host.Addrs() {
		log.Println(addr.String() + "/ipfs/" + n.Host.ID().Pretty())
	}
}

func logRepos(n *swarm.Node) {
	log.Printf("Known repos:")
	for _, repoName := range n.RepoManager.RepoNames() {
		log.Printf("  - %v", repoName)
		for _, object := range n.RepoManager.ObjectsForRepo(repoName) {
			log.Printf("      - %v", object.IDString())
		}
	}
}

func logConfig(n *swarm.Node) {
	log.Printf("Config:")

	var doLog func(interface{})
	doLog = func(x interface{}) {
		s := reflect.ValueOf(x) //.Elem()
		configType := s.Type()

		for i := 0; i < s.NumField(); i++ {
			f := s.Field(i)
			if f.Kind() == reflect.Struct {
				log.Println()
				log.Printf("____ %v ________", configType.Field(i).Name)
				doLog(f.Interface())
			} else {
				log.Printf("%v = %v", configType.Field(i).Name, f.Interface())
			}
		}
	}

	doLog(n.Config)
}