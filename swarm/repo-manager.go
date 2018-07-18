package main

import (
	// "bufio"
	// "bytes"
	"encoding/hex"
	"fmt"
	"io"
	// "os"
	// "os/exec"
	// "path/filepath"
	// "strings"

	// "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"
)

type RepoManager struct {
	repos map[string]RepoEntry
}

// @@TODO: save chunks as byte strings
type RepoEntry struct {
	RepoID string
	Path   string
	Chunks map[string]bool
}

var (
	ErrRepoNotFound = fmt.Errorf("repo not found")
)

func NewRepoManager() (*RepoManager, error) {
	rm := &RepoManager{
		repos: make(map[string]RepoEntry),
	}
	return rm, nil
}

func (rm *RepoManager) AddRepo(repoPath string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	//
	// get the repo's unique ID on the p2p network
	//
	cfg, err := repo.Config()
	if err != nil {
		return err
	}

	section := cfg.Raw.Section("conscience")
	if section == nil {
		return fmt.Errorf("repo config doesn't have conscience section")
	}
	repoID := section.Option("repoid")
	if repoID == "" {
		return fmt.Errorf("repo config doesn't have conscience.repoid key")
	}

	//
	// iterate over the objects and make note that we have them so the Node can .Provide them
	//
	oIter, err := repo.Objects()
	if err != nil {
		return err
	}

	chunks := map[string]bool{}
	err = oIter.ForEach(func(obj gitobject.Object) error {
		chunks[obj.ID().String()] = true
		return nil
	})
	if err != nil {
		return err
	}

	rm.repos[repoID] = RepoEntry{
		RepoID: repoID,
		Path:   repoPath,
		Chunks: chunks,
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
	repoEntry, ok := rm.repos[repoName]
	if !ok {
		return nil
	}

	chunks := make([]string, len(repoEntry.Chunks))
	i := 0
	for chunk := range repoEntry.Chunks {
		if repoEntry.Chunks[chunk] == true {
			chunks[i] = chunk
			i++
		}
	}
	return chunks
}

func (rm *RepoManager) HasChunk(repoID string, chunkID []byte) bool {
	repoEntry, ok := rm.repos[repoID]
	if !ok {
		return false
	}
	return repoEntry.Chunks[hex.EncodeToString(chunkID)]
}

func (rm *RepoManager) OpenChunk(repoID string, chunkID []byte) (io.ReadCloser, error) {
	repoEntry, ok := rm.repos[repoID]
	if !ok {
		return nil, ErrRepoNotFound
	}

	repo, err := git.PlainOpen(repoEntry.Path)
	if err != nil {
		return nil, err
	}

	// The object might be in a Packfile, so we use a more intelligent function to obtain a readable
	// handle to it.
	hash := gitplumbing.Hash{}
	copy(hash[:], chunkID)
	obj, err := repo.Storer.EncodedObject(gitplumbing.AnyObject, hash)
	if err != nil {
		return nil, err
	}

	r, err := obj.Reader()
	if err != nil {
		log.Errorf("WEIRD ERROR (@@todo: diagnose): %v", err)
		return nil, ErrChunkNotFound
	}
	return r, nil
}

func (rm *RepoManager) CreateChunk(repoID string, r io.Reader) ([]byte, int64, error) {
	repoEntry, ok := rm.repos[repoID]
	if !ok {
		return nil, 0, ErrRepoNotFound
	}

	repo, err := git.PlainOpen(repoEntry.Path)
	if err != nil {
		return nil, 0, err
	}

	obj := repo.Storer.NewEncodedObject() // returns a &plumbing.MemoryObject{}
	obj.SetType(gitplumbing.BlobObject)

	w, err := obj.Writer()
	if err != nil {
		return nil, 0, err
	}

	n, err := io.Copy(w, r)
	if err != nil {
		return nil, 0, err
	}

	err = w.Close()
	if err != nil {
		return nil, 0, err
	}

	hash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return nil, 0, err
	}

	return hash[:], n, nil

	// objectsRoot := filepath.Join(repoEntry.Path, ".git", "objects")

	// // make sure the objects root dir exists on disk
	// _, err := os.Stat(objectsRoot)
	// if err != nil {
	//  return nil, ErrRepoNotFound
	// }

	// // the chunk belongs in a folder named with the first 2 characters of the chunkID
	// objectDir := filepath.Join(objectsRoot, chunkID[:2])
	// err = os.MkdirAll(objectDir, 0777)
	// if err != nil {
	//  return nil, err
	// }

	// chunkPath := filepath.Join(objectDir, chunkID[2:])
	// f, err := os.Create(chunkPath)
	// if err != nil {
	//  return nil, ErrRepoNotFound
	// }
	// return f, nil
}

// func (rm *RepoManager) GitCatKind(sha1 string, repoName string) (string, error) {
//     catFile := exec.Command("git", "cat-file", "-t", sha1)

//     thisGitRepo := filepath.Join(rm.root, repoName)
//     catFile.Dir = thisGitRepo
//     out, err := catFile.CombinedOutput()

//     log.Printf(strings.TrimSpace(string(out)))
//     return strings.TrimSpace(string(out)), err
// }

// func (rm *RepoManager) GitListObjects(ref string, repoName string) ([]string, error) {
//     args := []string{"rev-list", "--objects", ref}
//     revList := exec.Command("git", args...)

//     thisGitRepo := filepath.Join(rm.root, repoName)
//     if strings.HasSuffix(thisGitRepo, ".git") {
//         thisGitRepo = filepath.Dir(thisGitRepo)
//     }
//     revList.Dir = thisGitRepo // GIT_DIR
//     out, err := revList.CombinedOutput()
//     if err != nil {
//         return nil, errors.Wrapf(err, "rev-list failed: %s\n%q", err, string(out))
//     }
//     var objs []string
//     s := bufio.NewScanner(bytes.NewReader(out))
//     for s.Scan() {
//         objs = append(objs, strings.Split(s.Text(), " ")[0])
//     }
//     if err := s.Err(); err != nil {
//         return nil, errors.Wrapf(err, "scanning rev-list output failed: %s", err)
//     }

//     return objs, nil
// }
