package main

import (
	"os"

	"../../config"
	"../../repo"
	"../../swarm"

	"gopkg.in/src-d/go-git.v4"
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
	if err == git.ErrRepositoryNotExists {
		r, err = repo.Init(cwd)
	}
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

	err = client.CreateRepo(repoID)
	if err != nil {
		panic(err)
	}
}
