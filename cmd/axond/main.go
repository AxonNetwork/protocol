package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"github.com/libgit2/git2go"
	"github.com/libp2p/go-libp2p-peer"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/config/env"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm"
	"github.com/Conscience/protocol/swarm/nodeexec"
	"github.com/Conscience/protocol/swarm/nodehttp"
	"github.com/Conscience/protocol/swarm/nodep2p"
	"github.com/Conscience/protocol/swarm/noderpc"
)

func main() {
	// pprof server for profiling
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	log.SetField("App", "axon-node")
	log.SetField("ReleaseStage", env.ReleaseStage)
	log.SetField("AppVersion", env.AppVersion)
	log.SetLevel(log.DebugLevel)

	app := cli.NewApp()
	app.Version = env.AppVersion

	configPath := filepath.Join(env.HOME, ".axonrc")
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
	nodehttp.InstallLogrusHook()

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
	"exec": {
		"asdf",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			var blahJS = []byte(`
	    const fs = require('fs')

	    const y = Math.floor(Math.random() * 1000)
	    fs.writeFileSync('/data/shared/nodejs-' + y + '.txt', 'from node')

	    let x = fs.readdirSync('/data')
	    console.log('/data ~>', x)
	    x = fs.readdirSync('/data/shared')
	    console.log('/data/shared ~>', x)

	    const _in = fs.createReadStream('/data/in')
	    const _out = fs.createWriteStream('/data/out')

	    _in.on('data', data => {
	        console.log('got some data ~>', data)
	        _out.write('here was your input: <'+data.toString()+'>')
	        _out.end()
	    })
	`)

			var blahPY = []byte(`
	import os
	from os.path import join, getsize

	with open('/data/shared/python.txt', 'w') as f:
	    f.write('from python')

	for root, dirs, files in os.walk('/data'):
	    for f in files:
	        print(root + '/' + f)

	with open('/data/in') as _in:
	    with open('/data/out', 'w') as _out:

	        data = _in.read(1024)
	        print('got data ~> %s' % str(data))
	        _out.write(data.upper())
	`)

			var inputStages = []nodeexec.InputStage{
				{
					"python",
					[]nodeexec.File{{"blah.py", int64(len(blahPY)), bytes.NewReader(blahPY)}},
					"blah.py",
					nil,
				},
				{
					"node",
					[]nodeexec.File{{"blah.js", int64(len(blahJS)), bytes.NewReader(blahJS)}},
					"blah.js",
					nil,
				},
				{
					"node",
					[]nodeexec.File{{"blah.js", int64(len(blahJS)), bytes.NewReader(blahJS)}},
					"blah.js",
					nil,
				},
			}

			pipelineIn, pipelineOut, err := nodeexec.StartPipeline(inputStages)
			if err != nil {
				fmt.Println(err)
				return err
			}
			log.Warnln("done starting pipeline!")

			wg := &sync.WaitGroup{}

			wg.Add(1)
			go func() {
				defer wg.Done()

				buf := make([]byte, 1024)
				for {
					log.Warnln("[read] reading...")
					n, err := io.ReadFull(pipelineOut, buf)
					if err == io.EOF {
						break
					} else if err == io.ErrUnexpectedEOF {

					} else if err != nil {
						panic(err)
					}
					log.Warnln("[read]", n, "bytes")

					log.Warnln("[out]", string(buf[:n]))
				}
			}()

			data := []byte("Us and them\nAnd after all we're only ordinary men\nMe, and you\nGod only knows it's not what we would choose to do\nForward he cried from the rear\nAnd the front rank died\nAnd the General sat, as the lines on the map\nMoved from side to side\nBlack and Blue\nAnd who knows which is which and who is who\nUp and down\nAnd in the end it's only round and round and round\nHaven't you heard it's a battle of words\nThe poster bearer cried\nListen son, said the man with the gun\nThere's room for you inside\nDown and out\nIt can't be helped but there's a lot of it about\nWith, without\nAnd who'll deny that's what the fighting's all about\nGet out of the way, it's a busy day\nAnd I've got things on my mind\nFor want of the price of tea and a slice\nThe old man died")
			log.Warnln("[write] writing...")
			num, err := io.Copy(pipelineIn, bytes.NewReader(data))
			if err != nil {
				return err
			}
			log.Warnln("[write] wrote", num, "bytes")
			pipelineIn.Close()

			wg.Wait()

			return nil
		},
	},

	"addrs": {
		"list the p2p addresses this node is using to communicate with its swarm",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			for _, addr := range n.P2PHost().Addrs() {
				log.Println(addr.String() + "/p2p/" + n.P2PHost().ID().Pretty())
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
			log.Printf("total connected peers: %v", len(n.P2PHost().Conns()))

			for _, pinfo := range n.P2PHost().Peers() {
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
				if s.Type().Kind() == reflect.Ptr && s.Type().Elem().Kind() == reflect.Struct {
					s = reflect.Indirect(s)
				}

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
			maxBytes, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			err = n.P2PHost().SetReplicationPolicy(repoID, maxBytes)
			return err
		},
	},

	"init": {
		"initialize a repo and add it to list of list of local repos",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 1 {
				return fmt.Errorf("not enough args")
			}
			_, err := n.RepoManager().TrackRepo(args[0], true)
			return err
		},
	},

	"add-repo": {
		"add a repo to the list of local repos this node is tracking and serving",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 1 {
				return fmt.Errorf("not enough args")
			}
			_, err := n.RepoManager().TrackRepo(args[0], true)
			return err
		},
	},

	"add-peer": {
		"connect to a new peer",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			if len(args) < 1 {
				return fmt.Errorf("not enough args")
			}
			return n.P2PHost().AddPeer(ctx, args[0])
		},
	},

	"fetch-and-set-ref": {
		"update a git ref for the given repository",
		func(ctx context.Context, args []string, n *swarm.Node) error {
			r := n.RepoManager().Repo(args[0])
			if r == nil {
				return errors.New("unknown repo")
			}

			_, err := n.P2PHost().FetchAndSetRef(ctx, &nodep2p.FetchOptions{Repo: r})
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

			tx, err := n.EthereumClient().UpdateRemoteRef(ctx, args[0], args[1], *oid1, *oid2)
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

			refs, total, err := n.EthereumClient().GetRemoteRefs(ctx, args[0], pageSize, page)
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

			tx, err := n.EthereumClient().EnsureUsername(ctx, username)
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

			_, err := n.P2PHost().Clone(context.TODO(), &nodep2p.CloneOptions{
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
			r := n.RepoManager().Repo(repoID)
			if r == nil {
				return errors.New("repo not found")
			}

			_, err := n.P2PHost().Push(context.TODO(), &nodep2p.PushOptions{
				Repo:       r,
				BranchName: "master",
				ProgressCb: func(percent int) {
					log.Warnln("Push progress: ", percent)
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

	var longestCommandLength int
	for cmd := range replCommands {
		if len(cmd) > longestCommandLength {
			longestCommandLength = len(cmd)
		}
	}

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
				difference := longestCommandLength - len(cmd)
				space := strings.Repeat(" ", difference+4)
				log.Printf("%v%v- %v", cmd, space, info.HelpText)
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
