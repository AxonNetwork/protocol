package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/swarm/noderpc"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var ErrNotEnoughArgs = errors.New("not enough args")

func main() {
	app := cli.NewApp()

	app.Version = "0.0.1"
	app.Copyright = "(c) 2018 Conscience"
	app.Usage = "Utility for interacting with the Conscience network"

	flags := []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Value: "",
			Usage: "name for local repo config",
		},
		cli.StringFlag{
			Name:  "email",
			Value: "",
			Usage: "email for local repo config",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:      "init",
			Aliases:   []string{"i"},
			UsageText: "axon init <repo ID>",
			Usage:     "initialize a git repo to interact with the Conscience network",
			ArgsUsage: "[args usage]",
			Flags:     flags,
			Action: func(c *cli.Context) error {
				repoID := c.Args().Get(0)
				if repoID == "" {
					return ErrNotEnoughArgs
				}
				path := c.Args().Get(1)
				if path == "" {
					cwd, err := os.Getwd()
					if err != nil {
						return err
					}
					path = cwd
				}
				name := c.String("name")
				email := c.String("email")
				return initRepo(repoID, path, name, email)
			},
		},
		{
			Name:      "set-username",
			UsageText: "axon set-username <username>",
			Usage:     "set your username on the Conscience network",
			ArgsUsage: "[args usage]",
			Action: func(c *cli.Context) error {
				if len(c.Args()) < 1 {
					return ErrNotEnoughArgs
				}

				username := c.Args().Get(0)
				return setUsername(username)
			},
		},
		{
			Name:      "replicate",
			UsageText: "axon replicate <repo ID> <1 | 0>",
			Usage:     "set whether or not to replicate the given repo",
			ArgsUsage: "[args usage]",
			Action: func(c *cli.Context) error {
				if len(c.Args()) < 2 {
					return ErrNotEnoughArgs
				}

				repoID := c.Args().Get(0)
				_shouldReplicate := c.Args().Get(1)

				shouldReplicate, err := strconv.ParseBool(_shouldReplicate)
				if err != nil {
					return errors.New("Bad argument.  Must be 1 or 0.")
				}
				return setReplicationPolicy(repoID, shouldReplicate)
			},
		},
		{
			Name:      "repos",
			UsageText: "axon repos",
			Usage:     "returns a list of axon repositories hosted locally on this machine",
			ArgsUsage: "[args usage]",
			Action: func(c *cli.Context) error {
				repos, err := getLocalRepos()
				if err != nil {
					return err
				}
				for _, repo := range repos {
					fmt.Fprintf(c.App.Writer, "%s\n", repo)
				}

				return nil
			},
		},
		{
			Name:      "get-refs",
			UsageText: "axon get-refs <repo ID>",
			Usage:     "return all on-chain refs for the given repo",
			ArgsUsage: "[args usage]",
			Action: func(c *cli.Context) error {
				if len(c.Args()) < 1 {
					return ErrNotEnoughArgs
				}

				repoID := c.Args().Get(0)

				refs, err := getAllRefs(repoID)
				if err != nil {
					return err
				}
				for _, ref := range refs {
					fmt.Fprintf(c.App.Writer, "%s %s\n", ref.CommitHash, ref.RefName)
				}

				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getClient() (*noderpc.Client, error) {
	cfg, err := config.ReadConfig()
	if err != nil {
		return nil, err
	}
	return noderpc.NewClient(cfg.RPCClient.Host)
}
