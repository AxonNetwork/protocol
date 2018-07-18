package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type RepoManager struct {
	root  string
	repos map[string]map[string]bool
}

func NewRepoManager() (*RepoManager, error) {
	rm := &RepoManager{
		repos: make(map[string]map[string]bool),
	}
	return rm, nil
}

func (rm *RepoManager) AddRepo(repoPath string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	//
	// get the repo's announce name (its unique ID on the p2p network)
	//
	cfg, err := repo.Config()
	if err != nil {
		return err
	}

	section := cfg.Raw.Section("conscience")
	if section == nil {
		return fmt.Errorf("repo config doesn't have conscience section")
	}
	announceName := section.Option("announcename")
	if announceName == "" {
		return fmt.Errorf("repo config doesn't have conscience.announcename key")
	}

	rm.repos[announceName] = make(map[string]bool)

	//
	// iterate over the objects and make note that we have them so the Node can .Provide them
	//
	oIter, err := repo.Objects()
	if err != nil {
		return err
	}

	err = oIter.ForEach(func(obj object.Object) error {
		rm.repos[announceName][obj.ID().String()] = true
		return nil
	})
	if err != nil {
		return err
	}

	return nil
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

func (rm *RepoManager) ChunksForRepo(repoName string) []string {
	repo, ok := rm.repos[repoName]
	if !ok {
		return nil
	}

	chunks := make([]string, len(repo))
	i := 0
	for chunk := range repo {
		chunks[i] = chunk
		i++
	}
	return chunks
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
