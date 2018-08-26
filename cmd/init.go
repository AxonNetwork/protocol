package main

import (
	"context"
	"os"

	"../repo"

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

	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// @@TODO: give context a timeout and make it configurable
	err = client.TrackLocalRepo(context.Background(), cwd)
	if err != nil {
		return err
	}

	// @@TODO: give context a timeout and make it configurable
	err = client.RegisterRepoID(context.Background(), repoID)
	if err != nil {
		return err
	}
	return nil
}

func setUsername(username string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// @@TODO: give context a timeout and make it configurable
	err = client.SetUsername(context.Background(), username)
	if err != nil {
		return err
	}
	return nil
}

func setReplicationPolicy(repoID string, shouldReplicate bool) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// @@TODO: give context a timeout and make it configurable
	err = client.SetReplicationPolicy(context.Background(), repoID, shouldReplicate)
	if err != nil {
		return err
	}
	return nil
}
