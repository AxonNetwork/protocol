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
	repos map[string]*Repo
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
		repos: make(map[string]*Repo),
	}
}

func (rm *RepoManager) AddRepo(repoPath string) (*Repo, error) {
	gitRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	repo := &Repo{
		Repository: gitRepo,
		Path:       repoPath,
	}

	repoID, err := repo.RepoID()
	if err != nil {
		return nil, err
	}

	rm.repos[repoID] = repo
	return repo, nil
}

func (rm *RepoManager) Repo(repoID string) *Repo {
	repo, ok := rm.repos[repoID]
	if !ok {
		return nil
	}
	return repo
}

func (rm *RepoManager) ForEachRepo(fn func(*Repo) error) error {
	for _, entry := range rm.repos {
		err := fn(entry)
		if err != nil {
			return err
		}
	}
	return nil
}

// Returns true if the object is known, false otherwise.
func (rm *RepoManager) HasObject(repoID string, objectID []byte) bool {
	repo, ok := rm.repos[repoID]
	if !ok {
		return false
	}

	if len(objectID) == CONSCIENCE_HASH_LENGTH {
		p := filepath.Join(repo.Path, ".git", CONSCIENCE_DATA_SUBDIR, hex.EncodeToString(objectID))
		_, err := os.Stat(p)
		return err == nil || !os.IsNotExist(err)

	} else if len(objectID) == GIT_HASH_LENGTH {
		hash := gitplumbing.Hash{}
		copy(hash[:], objectID)
		err := repo.Storer.HasEncodedObject(hash)
		return err == nil
	}

	return false
}

// Open an object for reading.  It is the caller's responsibility to .Close() the object when finished.
func (rm *RepoManager) OpenObject(repoID string, objectID []byte) (ObjectReader, error) {
	repo, ok := rm.repos[repoID]
	if !ok {
		return nil, errors.WithStack(ErrRepoNotFound)
	}

	if len(objectID) == CONSCIENCE_HASH_LENGTH {
		// Open a Conscience object
		p := filepath.Join(repo.Path, ".git", CONSCIENCE_DATA_SUBDIR, hex.EncodeToString(objectID))
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

type Repo struct {
	*git.Repository

	// RepoID string
	Path string
	// The string key is not the hexadecimal representation of the object ID (which is always a
	// []byte).  It's just a string typecast of the []byte because Go maps can't be keyed by byte
	// slices.
	//Objects map[string]struct{}
}

func (r *Repo) RepoID() (string, error) {
	cfg, err := r.Config()
	if err != nil {
		return "", err
	}

	section := cfg.Raw.Section("conscience")
	if section == nil {
		return "", fmt.Errorf("repo config doesn't have conscience section")
	}
	repoID := section.Option("repoid")
	if repoID == "" {
		return "", fmt.Errorf("repo config doesn't have conscience.repoid key")
	}

	return repoID, nil
}

func (r *Repo) ForEachObjectID(fn func([]byte) error) error {
	// First crawl the Git objects
	oIter, err := r.Repository.Objects()
	if err != nil {
		return err
	}

	err = oIter.ForEach(func(obj gitobject.Object) error {
		id := obj.ID()
		return fn(id[:])
	})
	if err != nil {
		return err
	}

	// Then crawl the Conscience objects
	dataDir, err := os.Open(filepath.Join(r.Path, ".git", CONSCIENCE_DATA_SUBDIR))
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

			err = fn(id)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
