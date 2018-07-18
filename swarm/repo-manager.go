package main

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
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

func (rm *RepoManager) GitCatKind(sha1 string, repoName string) (string, error) {
	catFile := exec.Command("git", "cat-file", "-t", sha1)

	thisGitRepo := filepath.Join(rm.root, repoName)
	catFile.Dir = thisGitRepo
	out, err := catFile.CombinedOutput()

	log.Printf(strings.TrimSpace(string(out)))
	return strings.TrimSpace(string(out)), err
}

func (rm *RepoManager) GitListObjects(ref string, repoName string) ([]string, error) {
	args := []string{"rev-list", "--objects", ref}
	revList := exec.Command("git", args...)

	thisGitRepo := filepath.Join(rm.root, repoName)
	if strings.HasSuffix(thisGitRepo, ".git") {
		thisGitRepo = filepath.Dir(thisGitRepo)
	}
	revList.Dir = thisGitRepo // GIT_DIR
	out, err := revList.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "rev-list failed: %s\n%q", err, string(out))
	}
	var objs []string
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		objs = append(objs, strings.Split(s.Text(), " ")[0])
	}
	if err := s.Err(); err != nil {
		return nil, errors.Wrapf(err, "scanning rev-list output failed: %s", err)
	}

	return objs, nil
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
