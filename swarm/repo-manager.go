package swarm

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"
)

type RepoManager struct {
	repos map[string]RepoEntry
}

type RepoEntry struct {
	RepoID  string
	Path    string
	Objects map[string]ObjectEntry
}

type ObjectEntry struct {
	ID   []byte
	Type gitplumbing.ObjectType
	Len  int
}

func (oe ObjectEntry) IDString() string {
	return hex.EncodeToString(oe.ID)
}

const (
	CONSCIENCE_DATA_SUBDIR = "data"
	CONSCIENCE_HASH_LENGTH = 32
	GIT_HASH_LENGTH        = 20
)

var (
	ErrRepoNotFound = fmt.Errorf("repo not found")
	ErrBadChecksum  = fmt.Errorf("object error: bad checksum")
)

func NewRepoManager() *RepoManager {
	return &RepoManager{
		repos: make(map[string]RepoEntry),
	}
}

func (rm *RepoManager) AddRepo(repoPath string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	//
	// Get the repo's unique ID on the p2p network
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
	// Iterate over the objects and make note that we have them so the Node can .Provide them
	//

	objects := map[string]ObjectEntry{}

	// First crawl the Git objects
	oIter, err := repo.Objects()
	if err != nil {
		return err
	}

	err = oIter.ForEach(func(obj gitobject.Object) error {
		id := obj.ID()
		objects[string(id[:])] = ObjectEntry{ID: id[:], Type: obj.Type()}
		return nil
	})
	if err != nil {
		return err
	}

	// Then crawl the Conscience objects
	dataDir, err := os.Open(filepath.Join(repoPath, ".git", CONSCIENCE_DATA_SUBDIR))
	if err == nil {
		entries, err := dataDir.Readdir(-1)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			// @@TODO: read the contents of each object and compare its name to its hash?
			id, err := hex.DecodeString(entry.Name())
			if err != nil {
				log.Errorf("bad conscience data object name: %v", entry.Name())
				continue
			} else if len(id) != CONSCIENCE_HASH_LENGTH {
				log.Errorf("bad conscience data object name: %v", entry.Name())
				continue
			}
			objects[string(id)] = ObjectEntry{ID: id, Type: 0}
		}
	}

	rm.repos[repoID] = RepoEntry{
		RepoID:  repoID,
		Path:    repoPath,
		Objects: objects,
	}

	return nil
}

// @@TODO: make this a ForEach with a closure
func (rm *RepoManager) RepoNames() []string {
	repoNames := make([]string, len(rm.repos))
	i := 0
	for repoName := range rm.repos {
		repoNames[i] = repoName
		i++
	}
	return repoNames
}

// @@TODO: make this a ForEach with a closure
func (rm *RepoManager) ObjectsForRepo(repoName string) []ObjectEntry {
	repoEntry, ok := rm.repos[repoName]
	if !ok {
		return nil
	}

	objects := make([]ObjectEntry, len(repoEntry.Objects))
	i := 0
	for _, object := range repoEntry.Objects {
		objects[i] = object
		i++
	}
	return objects
}

// Returns true if the object is known, false otherwise.
func (rm *RepoManager) HasObject(repoID string, objectID []byte) bool {
	repoEntry, ok := rm.repos[repoID]
	if !ok {
		return false
	}
	_, ok = repoEntry.Objects[string(objectID)]
	return ok
}

func (rm *RepoManager) Object(repoID string, objectID []byte) (ObjectEntry, bool) {
	repoEntry, ok := rm.repos[repoID]
	if !ok {
		return ObjectEntry{}, false
	}

	entry, ok := repoEntry.Objects[string(objectID)]
	if !ok {
		return ObjectEntry{}, false
	}

	return entry, true
}

// Open a object for reading.  It is the caller's responsibility to .Close() the object when finished.
func (rm *RepoManager) OpenObject(repoID string, objectID []byte) (io.ReadCloser, gitplumbing.ObjectType, int64, error) {
	repoEntry, ok := rm.repos[repoID]
	if !ok {
		return nil, 0, 0, errors.WithStack(ErrRepoNotFound)
	}

	if len(objectID) == CONSCIENCE_HASH_LENGTH {
		// Open a Conscience object
		p := filepath.Join(repoEntry.Path, ".git", CONSCIENCE_DATA_SUBDIR, hex.EncodeToString(objectID))
		f, err := os.Open(p)
		if err != nil {
			return nil, 0, 0, errors.Wrapf(ErrObjectNotFound, "RepoManager - %v:%v", repoID, hex.EncodeToString(objectID))
		}
		stat, err := f.Stat()
		if err != nil {
			return nil, 0, 0, err
		}

		return f, 0, stat.Size(), nil

	} else if len(objectID) == GIT_HASH_LENGTH {
		// Open a Git object

		repo, err := git.PlainOpen(repoEntry.Path)
		if err != nil {
			return nil, 0, 0, err
		}

		// The object might be in a Packfile, so we use a more intelligent function to obtain a readable
		// handle to it.
		hash := gitplumbing.Hash{}
		copy(hash[:], objectID)
		obj, err := repo.Storer.EncodedObject(gitplumbing.AnyObject, hash)
		if err != nil {
			return nil, 0, 0, err
		}

		r, err := obj.Reader()
		if err != nil {
			log.Errorf("WEIRD ERROR (@@todo: diagnose): %v", err)
			return nil, 0, 0, errors.WithStack(ErrObjectNotFound)
		}

		return r, obj.Type(), obj.Size(), nil

	} else {
		return nil, 0, 0, fmt.Errorf("objectID is wrong size (%v)", len(objectID))
	}
}
