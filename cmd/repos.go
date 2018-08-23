package main

import (
	"../swarm/wire"
)

func getAllRefs(repoID string) (map[string]wire.Ref, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}

	return client.GetAllRefs(repoID)
}

func getRepos() ([]wire.Repo, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}

	return client.GetLocalRepos()
}
