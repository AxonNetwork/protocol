package repo

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"gopkg.in/src-d/go-git.v4"
	gitconfig "gopkg.in/src-d/go-git.v4/config"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitconfigformat "gopkg.in/src-d/go-git.v4/plumbing/format/config"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"

	"../util"
)

type Repo struct {
	*git.Repository

	// RepoID string
	Path string
	// The string key is not the hexadecimal representation of the object ID (which is always a
	// []byte).  It's just a string typecast of the []byte because Go maps can't be keyed by byte
	// slices.
	//Objects map[string]struct{}
}

const (
	CONSCIENCE_DATA_SUBDIR = "data"
	CONSCIENCE_HASH_LENGTH = 32
	GIT_HASH_LENGTH        = 20
)

var (
	ErrRepoNotFound   = fmt.Errorf("repo not found")
	ErrObjectNotFound = fmt.Errorf("object not found")
	ErrBadChecksum    = fmt.Errorf("object error: bad checksum")
)

func Init(path string) (*Repo, error) {
	gitRepo, err := git.PlainInit(path, false)
	if err != nil {
		return nil, err
	}
	_, err = os.Create(filepath.Join(path, ".git", "config"))
	if err != nil {
		return nil, err
	}

	return &Repo{
		Repository: gitRepo,
		Path:       path,
	}, nil
}

func Open(path string) (*Repo, error) {
	gitRepo, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}

	return &Repo{
		Repository: gitRepo,
		Path:       path,
	}, nil
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

func (r *Repo) HeadHash() (string, error) {
	head, err := r.Head()
	if err != nil {
		return "", err
	}
	return head.Hash().String(), nil
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

// Returns true if the object is known, false otherwise.
func (r *Repo) HasObject(repoID string, objectID []byte) bool {
	if len(objectID) == CONSCIENCE_HASH_LENGTH {
		p := filepath.Join(r.Path, ".git", CONSCIENCE_DATA_SUBDIR, hex.EncodeToString(objectID))
		_, err := os.Stat(p)
		return err == nil || !os.IsNotExist(err)

	} else if len(objectID) == GIT_HASH_LENGTH {
		hash := gitplumbing.Hash{}
		copy(hash[:], objectID)
		err := r.Storer.HasEncodedObject(hash)
		return err == nil
	}

	return false
}

// Open an object for reading.  It is the caller's responsibility to .Close() the object when finished.
func (r *Repo) OpenObject(objectID []byte) (*util.ObjectReader, error) {
	if len(objectID) == CONSCIENCE_HASH_LENGTH {
		// Open a Conscience object
		p := filepath.Join(r.Path, ".git", CONSCIENCE_DATA_SUBDIR, hex.EncodeToString(objectID))
		f, err := os.Open(p)
		if err != nil {
			return nil, errors.Wrapf(ErrObjectNotFound, "RepoManager")
		}
		stat, err := f.Stat()
		if err != nil {
			return nil, err
		}

		or := &util.ObjectReader{
			Reader:     f,
			Closer:     f,
			ObjectType: 0,
			ObjectLen:  stat.Size(),
		}
		return or, nil

	} else if len(objectID) == GIT_HASH_LENGTH {
		hash := gitplumbing.Hash{}
		copy(hash[:], objectID)
		obj, err := r.Storer.EncodedObject(gitplumbing.AnyObject, hash)
		if err != nil {
			return nil, err
		}

		r, err := obj.Reader()
		if err != nil {
			log.Errorf("WEIRD ERROR (@@todo: diagnose): %v", err)
			return nil, errors.WithStack(ErrObjectNotFound)
		}

		or := &util.ObjectReader{
			Reader:     r,
			Closer:     r,
			ObjectType: obj.Type(),
			ObjectLen:  obj.Size(),
		}
		return or, nil

	} else {
		return nil, fmt.Errorf("objectID is wrong size (%v)", len(objectID))
	}
}

func (r *Repo) SetupConfig(repoID string) error {
	cfg, err := r.Config()
	if err != nil {
		return err
	}

	raw := cfg.Raw
	changed := false
	section := raw.Section("conscience")

	if section.Option("repoid") != repoID {
		raw.SetOption("conscience", "", "repoid", repoID)
		changed = true
	}

	filter := raw.Section("filter").Subsection("conscience")
	if filter.Option("clean") != "conscience_encode" {
		raw.SetOption("filter", "conscience", "clean", "conscience_encode")
		changed = true
	}
	if filter.Option("smudge") != "conscience_decode" {
		raw.SetOption("filter", "conscience", "smudge", "conscience_decode")
		changed = true
	}

	if changed {
		p := filepath.Join(r.Path, ".git", "config")
		f, err := os.OpenFile(p, os.O_WRONLY, os.ModeAppend)
		if err != nil {
			return err
		}
		w := io.Writer(f)

		enc := gitconfigformat.NewEncoder(w)
		err = enc.Encode(raw)
		if err != nil {
			return err
		}
	}

	// Check the remotes
	{
		remotes, err := r.Remotes()
		if err != nil {
			return err
		}

		found := false
		hasOrigin := false
		for _, remote := range remotes {
			log.Printf("remote <%v> URLs: %v", remote.Config().Name, remote.Config().URLs)

			if remote.Config().Name == "origin" {
				hasOrigin = true
			}

			for _, url := range remote.Config().URLs {
				if url == "conscience://"+repoID {
					found = true
					break
				}
			}
		}

		if !found {
			remoteName := "origin"
			if hasOrigin {
				// @@TODO: what if this remote name already exists too?
				remoteName = repoID
			}

			_, err = r.CreateRemote(&gitconfig.RemoteConfig{
				Name:  remoteName,
				URLs:  []string{"conscience://" + repoID},
				Fetch: []gitconfig.RefSpec{gitconfig.RefSpec("+refs/heads/*:refs/remotes/" + remoteName + "/*")},
			})

			if err != nil {
				return err
			}
		}
	}

	return nil
}
