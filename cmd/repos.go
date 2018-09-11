package main

import (
	"context"
	"fmt"

	"../swarm/wire"
)

func getAllRefs(repoID string) (map[string]wire.Ref, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// @@TODO: give context a timeout and make it configurable
	return client.GetAllRefs(context.Background(), repoID)
}

type Repo struct {
	RepoID string
	Path   string
}

func getLocalRepos() ([]string, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	repos := make([]string, 0)
	// @@TODO: give context a timeout and make it configurable
	ch, err := client.GetLocalRepos(context.Background())
	if err != nil {
		return nil, err
	}
	for {
		maybeRepo := <-ch
		if maybeRepo.LocalRepo.RepoID == "" {
			break
		}
		repo := maybeRepo.LocalRepo
		repos = append(repos, fmt.Sprintf("%s %s", repo.RepoID, repo.Path))
	}

	return repos, nil
}
