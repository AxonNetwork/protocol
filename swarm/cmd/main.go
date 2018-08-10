package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"

	tm "github.com/buger/goterm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"

	swarm ".."
	"../../config"
	"../../repo"
	"../statsserver"
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

	// Start the node stats HTTP server
	go statsserver.Start(":8081", n)

	// Catch ctrl+c so that we can gracefully shut down the Node
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		n.Close()
		os.Exit(1)
	}()

	inputLoop(ctx, n)
}

var replCommands = map[string]struct {
	HelpText string
	Handler  func(ctx context.Context, args []string, n *swarm.Node) error
}{
	"state": {
		"show an overview of the current state of the node",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			logState(n)
			return nil
		},
	},

	"addrs": {
		"list the p2p addresses this node is using to communicate with its swarm",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			logAddrs(n)
			return nil
		},
	},

	"repos": {
		"list the local repositories this node is currently tracking and serving",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			logRepos(n)
			return nil
		},
	},

	"peers": {
		"list the peers this node is currently connected to",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			logPeers(n)
			return nil
		},
	},

	"config": {
		"display the node's configuration",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			logConfig(n)
			return nil
		},
	},

	"replicate-repo": {
		"change the node's policy on replicating the given repo",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 2 {
				return fmt.Errorf("not enough args")
			}
			repoID := args[0]
			shouldReplicate, err := strconv.ParseBool(args[1])
			if err != nil {
				return err
			}
			err = n.SetReplicationPolicy(repoID, shouldReplicate)
			return err
		},
	},

	"add-repo": {
		"add a repo to the list of local repos this node is tracking and serving",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 1 {
				return fmt.Errorf("not enough args")
			}
			_, err := n.RepoManager.AddRepo(args[0])
			return err
		},
	},

	"add-peer": {
		"connect to a new peer",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 1 {
				return fmt.Errorf("not enough args")
			}
			return n.AddPeer(ctx, args[0])
		},
	},

	"update-ref": {
		"update a git ref for the given repository",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 3 {
				return fmt.Errorf("not enough args")
			}
			tx, err := n.Eth.UpdateRef(ctx, args[0], args[1], args[2])
			if err != nil {
				return err
			}
			log.Printf("update ref tx sent: %v", tx.Hash().Hex())
			txResult := <-tx.Await(ctx)
			if txResult.Err != nil {
				return txResult.Err
			}
			log.Printf("update ref tx resolved: %v", tx.Hash().Hex())
			return nil
		},
	},

	"get-refs": {
		"show the list of git refs for the current repository",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 2 {
				return fmt.Errorf("not enough args")
			}

			page, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}

			refs := n.Eth.GetRefs(ctx, args[0], page)
			for _, ref := range refs {
				log.Printf("ref: %v %v", ref.Name, ref.Commit)
			}
			return nil
		},
	},

	"set-username": {
		"set your username",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			var username string
			if len(args) == 0 {
				username = n.Config.User.Username
			} else {
				username = args[0]
			}

			tx, err := n.Eth.EnsureUsername(ctx, username)
			if err != nil {
				return err
			} else if tx == nil && err == nil {
				return fmt.Errorf("username already set")
			}

			log.Printf("set username tx sent: %v", tx.Hash().Hex())
			txResult := <-tx.Await(ctx)
			if txResult.Err != nil {
				return txResult.Err
			} else if txResult.Receipt.Status == 0 {
				return errors.New("SetUsername transaction failed")
			}
			log.Printf("set username tx resolved: %v", tx.Hash().Hex())
			return nil
		},
	},
}

func inputLoop(ctx context.Context, n *swarm.Node) {
	fmt.Println("Type \"help\" for a list of commands.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("> ")

		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		parts := strings.Split(line, " ")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		if len(parts) < 1 {
			log.Errorln("enter a command")
			continue
		} else if parts[0] == "help" {
			log.Println("___ Commands _________")
			log.Println()
			for cmd, info := range replCommands {
				log.Printf("%v\t\t- %v", cmd, info.HelpText)
			}
			continue
		}

		cmd, exists := replCommands[parts[0]]
		if !exists {
			log.Errorln("unknown command")
			continue
		}

		err := cmd.Handler(ctx, parts[1:], n)
		if err != nil {
			log.Errorln(err)
		}
	}
}

func logState(n *swarm.Node) {
	tm.Clear()
	tm.MoveCursor(1, 1)
	box := tm.NewBox(50|tm.PCT, 3, 0)
	_, err := box.Write([]byte("Conscience"))
	if err != nil {
		panic(err)
	}
	tm.Println(box.String())
	state, err := n.GetNodeState()
	if err != nil {
		tm.Println("There has been some error")
		tm.Println(err.Error())
		tm.Flush()
		return
	}
	tm.Printf("%v %v\n", tm.Bold("Username:"), state.User)
	tm.Printf("%v %v\n", tm.Bold("Ethereum Address:"), state.EthAccount)
	tm.Printf("\n%v\n", tm.Bold("Node ('addrs' for more info):"))
	tm.Println(state.Addrs[1])
	tm.Printf("\n%v ('peers' for more info):\n", tm.Bold("Peers"))
	if len(state.Peers) < 2 {
		tm.Printf("  No peers at the moment\n")
	}
	for peer, addrs := range state.Peers {
		if len(addrs) > 1 {
			tm.Printf("  -%s/ipfs/%s\n", addrs[1], peer)
		}
	}
	tm.Printf("\n%v ('repos' for more info)\n", tm.Bold("\nRepos"))
	for repo := range state.Repos {
		tm.Printf("  - %s\n", repo)
	}

	tm.Flush()
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

	n.RepoManager.ForEachRepo(func(r *repo.Repo) error {
		repoID, err := r.RepoID()
		if err != nil {
			return err
		}

		log.Printf("  - %v", repoID)

		err = r.ForEachObjectID(func(objectID []byte) error {
			log.Printf("      - %v", hex.EncodeToString(objectID))
			return nil
		})
		return err
	})
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
