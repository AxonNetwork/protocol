package main

import (
	"os"

	"../config"
	"../repo"
	"../swarm"

	"gopkg.in/src-d/go-git.v4"
)

func initRepo(repoID string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	r, err := repo.Open(cwd)
	if err == git.ErrRepositoryNotExists {
		r, err = repo.Init(cwd)
	}
	if err != nil {
		return err
	}

	err = r.SetupConfig(repoID)
	if err != nil {
		return err
	}

	cfg, err := config.ReadConfig()
	if err != nil {
		return err
	}

	client, err := swarm.NewRPCClient(cfg.RPCClient.Network, cfg.RPCClient.Host)
	if err != nil {
		return err
	}

	err = client.AddRepo(cwd)
	if err != nil {
		return err
	}

	err = client.CreateRepo(repoID)
	if err != nil {
		return err
	}
	return nil
}
