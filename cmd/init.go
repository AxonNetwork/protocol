package main

import (
	"context"
)

func initRepo(repoID string, path string, name string, email string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// @@TODO: give context a timeout and make it configurable
	err = client.InitRepo(context.Background(), repoID, path, name, email)
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
