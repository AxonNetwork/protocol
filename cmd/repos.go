package main

import (
	"context"

	"../swarm/noderpc2"
	"../swarm/wire"
)

func getAllRefs(repoID string) (map[string]wire.Ref, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}

	// @@TODO: give context a timeout and make it configurable
	return client.GetAllRefs(context.Background(), repoID)
}

func getLocalRepos() (chan noderpc.MaybeLocalRepo, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}

	// @@TODO: give context a timeout and make it configurable
	return client.GetLocalRepos(context.Background())
}
