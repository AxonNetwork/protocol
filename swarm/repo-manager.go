package swarm

import (
	"encoding/hex"
	"fmt"
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
	Username string
	RepoID   string
	Path     string
	Objects  map[string]struct{}
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
	username := section.Option("username")
	if username == "" {
		return fmt.Errorf("repo config doesn't have conscience.username key")
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
		objects[string(id[:])] = struct{}{}
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
			objects[string(id)] = struct{}{}
		}
	}

	rm.repos[repoID] = RepoEntry{
		Username: username,
		RepoID:   repoID,
		Path:     repoPath,
		Objects:  objects,
	}

	return nil
}

func (rm *RepoManager) ForEachRepo(fn func(RepoEntry) error) error {
	for _, entry := range rm.repos {
		err := fn(entry)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r RepoEntry) ForEachObject(fn func([]byte) error) {
	for key := range r.Objects {
		err := fn([]byte(key))
		if err != nil {
			return err
		}
	}
	return nil
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

// Open a object for reading.  It is the caller's responsibility to .Close() the object when finished.
func (rm *RepoManager) OpenObject(repoID string, objectID []byte) (ObjectReader, error) {
	repoEntry, ok := rm.repos[repoID]
	if !ok {
		return nil, errors.WithStack(ErrRepoNotFound)
	}

	if len(objectID) == CONSCIENCE_HASH_LENGTH {
		// Open a Conscience object
		p := filepath.Join(repoEntry.Path, ".git", CONSCIENCE_DATA_SUBDIR, hex.EncodeToString(objectID))
		f, err := os.Open(p)
		if err != nil {
			return nil, errors.Wrapf(ErrObjectNotFound, "RepoManager - %v:%v", repoID, hex.EncodeToString(objectID))
		}
		stat, err := f.Stat()
		if err != nil {
			return nil, err
		}

		or := objectReader{
			Reader:     f,
			Closer:     f,
			objectType: 0,
			objectLen:  stat.Size(),
		}
		return or, nil

	} else if len(objectID) == GIT_HASH_LENGTH {
		// Open a Git object

		repo, err := git.PlainOpen(repoEntry.Path)
		if err != nil {
			return nil, err
		}

		// The object might be in a Packfile, so we use a more intelligent function to obtain a readable
		// handle to it.
		hash := gitplumbing.Hash{}
		copy(hash[:], objectID)
		obj, err := repo.Storer.EncodedObject(gitplumbing.AnyObject, hash)
		if err != nil {
			return nil, err
		}

		r, err := obj.Reader()
		if err != nil {
			log.Errorf("WEIRD ERROR (@@todo: diagnose): %v", err)
			return nil, errors.WithStack(ErrObjectNotFound)
		}

		or := objectReader{
			Reader:     r,
			Closer:     r,
			objectType: obj.Type(),
			objectLen:  obj.Size(),
		}
		return or, nil

	} else {
		return nil, fmt.Errorf("objectID is wrong size (%v)", len(objectID))
	}
}
