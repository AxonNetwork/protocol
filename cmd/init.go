package main

import (
	"os"

	"../config"
	"../repo"
	"../swarm/noderpc"

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

	client, err := noderpc.NewClient(cfg.RPCClient.Network, cfg.RPCClient.Host)
	if err != nil {
		return err
	}

	err = client.AddRepo(cwd)
	if err != nil {
		return err
	}

	err = client.RegisterRepoID(repoID)
	if err != nil {
		return err
	}
	return nil
}

func setUsername(username string) error {
	cfg, err := config.ReadConfig()
	if err != nil {
		return err
	}

	client, err := noderpc.NewClient(cfg.RPCClient.Network, cfg.RPCClient.Host)
	if err != nil {
		return err
	}

	err = client.SetUsername(username)
	if err != nil {
		return err
	}
	return nil
}

func setReplicationPolicy(repoID string, shouldReplicate bool) error {
	cfg, err := config.ReadConfig()
	if err != nil {
		return err
	}

	client, err := noderpc.NewClient(cfg.RPCClient.Network, cfg.RPCClient.Host)
	if err != nil {
		return err
	}

	err = client.SetReplicationPolicy(repoID, shouldReplicate)
	if err != nil {
		return err
	}
	return nil
}
