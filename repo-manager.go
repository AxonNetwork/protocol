package main

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

type RepoManager struct {
	root  string
	repos map[string]map[string]bool
}

func NewRepoManager(root string) (*RepoManager, error) {
	err := os.MkdirAll(root, 0777)
	if err != nil {
		return nil, err
	}

	rootDir, err := os.Open(root)
	if err != nil {
		return nil, err
	}
	defer rootDir.Close()

	repos, err := rootDir.Readdir(-1)
	if err != nil {
		return nil, err
	}

	rm := &RepoManager{
		root:  root,
		repos: make(map[string]map[string]bool),
	}

	for _, repo := range repos {
		if repo.IsDir() {
			rm.repos[repo.Name()] = make(map[string]bool)

			repoDir, err := os.Open(filepath.Join(root, repo.Name()))
			if err != nil {
				return nil, err
			}

			chunks, err := repoDir.Readdir(-1)
			if err != nil {
				repoDir.Close()
				return nil, err
			}

			for _, chunk := range chunks {
				if !chunk.IsDir() {
					rm.repos[repo.Name()][chunk.Name()] = true
				}
			}

			repoDir.Close()
		}
	}

	rm.LogRepos()

	return rm, nil
}

func (rm *RepoManager) RepoNames() []string {
	repoNames := make([]string, len(rm.repos))
	i := 0
	for repoName := range rm.repos {
		repoNames[i] = repoName
		i++
	}
	return repoNames
}

func (rm *RepoManager) LogRepos() {
	log.Printf("Known repos:")
	for repoName, repo := range rm.repos {
		log.Printf("  - %v", repoName)
		for chunk := range repo {
			log.Printf("      - %v", chunk)
		}
	}
}

func (rm *RepoManager) HasChunk(repoName, chunkID string) bool {
	repo, ok := rm.repos[repoName]
	if !ok {
		return false
	}

	return repo[chunkID]
}

func (rm *RepoManager) OpenChunk(repoName, chunkID string) (*os.File, error) {
	chunkPath := filepath.Join(rm.root, repoName, chunkID)
	return os.Open(chunkPath)
}

func (rm *RepoManager) CreateChunk(repoName, chunkID string) (*os.File, error) {
	err := os.MkdirAll(filepath.Join(rm.root, repoName), 0777)
	if err != nil {
		return nil, err
	}

	chunkPath := filepath.Join(rm.root, repoName, chunkID)
	return os.Create(chunkPath)
}
