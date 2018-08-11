package main

import (
	"log"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var ErrNotEnoughArgs = errors.New("not enough args")

func main() {
	app := cli.NewApp()

	app.Version = "0.0.1"
	app.Copyright = "(c) 2018 Conscience"
	app.Usage = "Utility for interacting with the Consience network"

	app.Commands = []cli.Command{
		{
			Name:      "init",
			Aliases:   []string{"i"},
			UsageText: "conscience init <repo ID>",
			Usage:     "initialize a git repo to interact with the Conscience network",
			ArgsUsage: "[args usage]",
			Action: func(c *cli.Context) error {
				repoID := c.Args().Get(0)
				if repoID == "" {
					return ErrNotEnoughArgs
				}
				return initRepo(repoID)
			},
		},
		{
			Name:      "set-username",
			UsageText: "conscience set-username <username>",
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
			UsageText: "conscience replicate <repo ID> <1 | 0>",
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
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
