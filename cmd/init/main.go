package main

import (
	"os"

	"../../config"
	"../../repo"
	"../../swarm"

	log "github.com/sirupsen/logrus"
)

func main() {
	if len(os.Args) < 2 {
		panic("usage: conscience-init <repo id>")
	}

	repoID := os.Args[1]

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	r, err := repo.Open(cwd)
	if err != nil {
		panic(err)
	}

	err = r.SetupConfig(repoID)
	if err != nil {
		panic(err)
	}

	cfg, err := config.ReadConfig()
	if err != nil {
		panic(err)
	}

	client, err := swarm.NewRPCClient(cfg.RPCClient.Network, cfg.RPCClient.Host)
	if err != nil {
		panic(err)
	}

	err = client.AddRepo(cwd)
	if err != nil {
		panic(err)
	}

	log.Printf("Creating Repo On-chain")
	err = client.CreateRepo(repoID)
	if err != nil {
		panic(err)
	}

	// _, err = n.eth.CreateRepository(ctx, repoID)
	// if err != nil {
	// 	return err
	// }

	// head, err := repo.Head()
	// if err != nil {
	// 	return err
	// }

	// _, err = n.eth.UpdateRef(ctx, repoID, "refs/heads/master", head.Hash().String())
	// if err != nil {
	// 	return err
	// }
}
