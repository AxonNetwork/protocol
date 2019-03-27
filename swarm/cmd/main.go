package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"github.com/libgit2/git2go"
	"github.com/libp2p/go-libp2p-peer"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/config/env"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm"
	"github.com/Conscience/protocol/swarm/logger"
	"github.com/Conscience/protocol/swarm/nodehttp"
	"github.com/Conscience/protocol/swarm/nodep2p"
	"github.com/Conscience/protocol/swarm/noderpc"
)

func main() {
	// pprof server for profiling
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	log.SetField("App", "conscience-node")
	log.SetField("ReleaseStage", env.ReleaseStage)
	log.SetField("AppVersion", env.AppVersion)
	log.SetLevel(log.DebugLevel)

	app := cli.NewApp()
	app.Version = env.AppVersion

	configPath := filepath.Join(env.HOME, ".consciencerc")
	// for setting custom config path. Mainly used for testing with multiple nodes
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Value: configPath,
			Usage: "location of config file",
		},
	}

	app.Action = func(c *cli.Context) error {
		configPath := c.String("config")
		return run(configPath)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(configPath string) error {
	ctx, ctxCancel := context.WithCancel(context.Background())

	// Add our custom logger hook (used by nodehttp.Server)
	logger.InstallHook()

	// Read the config file
	cfg, err := config.ReadConfigAtPath(configPath)
	if err != nil {
		return err
	}
	config.AttachToLogger(cfg)

	n, err := swarm.NewNode(ctx, cfg)
	if err != nil {
		panic(err)
	}

	// Start the node HTTP server
	httpserver := nodehttp.New(n)
	go httpserver.Start()

	// Start the node RPC server
	rpcserver := noderpc.NewServer(n)
	go rpcserver.Start()

	// When the node shuts down, the HTTP and RPC servers should shut down as well
	go func() {
		<-n.Shutdown

		err := httpserver.Close()
		if err != nil {
			log.Errorf("error shutting down http server: %v", err)
		}

		err = rpcserver.Close()
		if err != nil {
			log.Errorf("error shutting down rpc server: %v", err)
		}
	}()

	// Catch ctrl+c so that we can gracefully shut down the Node
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		ctxCancel() // Stop any goroutines spawned by the node
		n.Close()
		os.Exit(1)
	}()

	go inputLoop(ctx, n)

	// Hang forever
	select {}

	return nil
}

var replCommands = map[string]struct {
	HelpText string
	Handler  func(ctx context.Context, args []string, n *swarm.Node) error
}{
	"addrs": {
		"list the p2p addresses this node is using to communicate with its swarm",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			for _, addr := range n.Addrs() {
				log.Println(addr.String() + "/p2p/" + n.ID().Pretty())
			}
			return nil
		},
	},

	"repos": {
		"list the local repositories this node is currently tracking and serving",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			log.Printf("Known repos:")

			return n.RepoManager().ForEachRepo(func(r *repo.Repo) error {
				repoID, err := r.RepoID()
				if err != nil {
					return err
				}

				log.Printf("  - %v", repoID)
				return nil
			})
		},
	},

	"peers": {
		"list the peers this node is currently connected to",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			log.Printf("total connected peers: %v", len(n.Conns()))

			for _, pinfo := range n.Peers() {
				log.Printf("  - %v (%v)", pinfo.ID.String(), peer.IDB58Encode(pinfo.ID))
				for _, addr := range pinfo.Addrs {
					log.Printf("      - %v", addr)
				}
			}
			return nil
		},
	},

	"config": {
		"display the node's configuration",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			log.Printf("Config:")

			var doLog func(interface{})
			doLog = func(x interface{}) {
				s := reflect.ValueOf(x) //.Elem()
				configType := s.Type()

				for i := 0; i < s.NumField(); i++ {
					f := s.Field(i)
					if f.Kind() == reflect.Ptr {
						log.Println()
						log.Printf("____ %v ________", configType.Field(i).Name)
						if f.CanInterface() {
							doLog(reflect.Indirect(f).Interface())
						}
					} else {
						if f.CanInterface() {
							log.Printf("%v = %v", configType.Field(i).Name, f.Interface())
						}
					}
				}
			}

			doLog(n.Config)
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

	"init": {
		"initialize a repo and add it to list of list of local repos",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 1 {
				return fmt.Errorf("not enough args")
			}
			_, err := n.TrackRepo(args[0], true)
			return err
		},
	},

	"add-repo": {
		"add a repo to the list of local repos this node is tracking and serving",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 1 {
				return fmt.Errorf("not enough args")
			}
			_, err := n.TrackRepo(args[0], true)
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

	"fetch-and-set-ref": {
		"update a git ref for the given repository",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			r := n.RepoManager().Repo(args[0])
			if r == nil {
				return errors.New("unknown repo")
			}

			_, err := nodep2p.FetchAndSetRef(ctx, &nodep2p.FetchOptions{Repo: r})
			return err
		},
	},

	"update-ref": {
		"update a git ref for the given repository",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 3 {
				return fmt.Errorf("not enough args")
			}

			oid1, err := git.NewOid(args[2])
			if err != nil {
				return err
			}
			oid2, err := git.NewOid(args[3])
			if err != nil {
				return err
			}

			tx, err := n.UpdateRef(ctx, args[0], args[1], *oid1, *oid2)
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

	"remote-refs": {
		"show the list of remote git refs for the given repository",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 3 {
				return fmt.Errorf("not enough args")
			}

			pageSize, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			page, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			refs, total, err := n.GetRemoteRefs(ctx, args[0], pageSize, page)
			if err != nil {
				return err
			}

			log.Printf("(%v total)", total)
			for refName, commitHash := range refs {
				log.Printf("ref: %v %v", refName, commitHash)
			}
			return nil
		},
	},

	"set-username": {
		"set your username",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 1 {
				return fmt.Errorf("not enough args")
			}

			username := args[0]

			tx, err := n.EnsureUsername(ctx, username)
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

	"clone": {
		"clone a repo",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 2 {
				return fmt.Errorf("not enough args")
			}
			repoID := args[0]
			repoRoot := args[1]

			_, err := nodep2p.Clone(context.TODO(), &nodep2p.CloneOptions{
				Node:      n,
				RepoID:    repoID,
				RepoRoot:  repoRoot,
				Bare:      false,
				UserName:  "bryn",
				UserEmail: "bryn@hi.com",
				ProgressCb: func(done, total uint64) error {
					log.Warnln("progress", done, total)
					return nil
				},
			})
			if err != nil {
				return err
			}

			log.Infoln("done.")

			return nil
		},
	},

	"push": {
		"push a repo",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 1 {
				return fmt.Errorf("not enough args")
			}
			repoID := args[0]
			r := n.Repo(repoID)
			if r == nil {
				return errors.New("repo not found")
			}

			_, err := nodep2p.Push(context.TODO(), &nodep2p.PushOptions{
				Node:       n,
				Repo:       r,
				BranchName: "master",
				ProgressCb: func(percent int) {
					log.Warnln("Push Progress: ", percent)
				},
			})
			if err != nil {
				return err
			}

			log.Infoln("push done.")

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
