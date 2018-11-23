package repo

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"gopkg.in/src-d/go-git.v4"
	gitconfig "gopkg.in/src-d/go-git.v4/config"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitconfigformat "gopkg.in/src-d/go-git.v4/plumbing/format/config"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

type Repo struct {
	*git.Repository

	Path string
}

const (
	CONSCIENCE_DATA_SUBDIR = "data"
	CONSCIENCE_HASH_LENGTH = 32
	GIT_HASH_LENGTH        = 20
)

var (
	ErrRepoNotFound   = git.ErrRepositoryNotExists
	ErrObjectNotFound = fmt.Errorf("object not found")
	ErrBadChecksum    = fmt.Errorf("object error: bad checksum")
)

func EnsureExists(path string) (*Repo, error) {
	r, err := Open(path)
	if err == nil {
		return r, nil
	} else if errors.Cause(err) != ErrRepoNotFound {
		return nil, err
	}
	return Init(path)
}

func Init(path string) (*Repo, error) {
	gitRepo, err := git.PlainInit(path, false)
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize repo at path '%v'", path)
	}

	f, err := os.Create(filepath.Join(path, ".git", "config"))
	if err != nil {
		return nil, errors.Wrapf(err, "could not create .git/config at path '%v'", path)
	}
	defer f.Close()

	return &Repo{
		Repository: gitRepo,
		Path:       path,
	}, nil
}

func Open(path string) (*Repo, error) {
	gitRepo, err := git.PlainOpen(path)
	if err != nil {
		return nil, errors.Wrapf(err, "could not open repo at path '%v'", path)
	}

	return &Repo{
		Repository: gitRepo,
		Path:       path,
	}, nil
}

func (r *Repo) RepoID() (string, error) {
	cfg, err := r.Config()
	if err != nil {
		return "", errors.Wrapf(err, "could not open repo config at path '%v'", r.Path)
	}

	section := cfg.Raw.Section("conscience")
	if section == nil {
		return "", errors.Errorf("repo config doesn't have conscience section (path: %v)", r.Path)
	}
	repoID := section.Option("repoid")
	if repoID == "" {
		return "", errors.Errorf("repo config doesn't have conscience.repoid key (path: %v)", r.Path)
	}

	return repoID, nil
}

func (r *Repo) HeadHash() (string, error) {
	head, err := r.Head()
	if err != nil {
		return "", errors.Wrapf(err, "could not fetch HEAD for repo (path: %v)", r.Path)
	}
	return head.Hash().String(), nil
}

func (r *Repo) ForEachObjectID(fn func([]byte) error) error {
	// First crawl the Git objects
	oIter, err := r.Repository.Objects()
	if err != nil {
		return errors.Wrapf(err, "could not fetch repo object iterator (path: %v)", r.Path)
	}

	err = oIter.ForEach(func(obj gitobject.Object) error {
		id := obj.ID()
		return fn(id[:])
	})
	if err != nil {
		return errors.Wrapf(err, "could not iterate over repo objects (path: %v)", r.Path)
	}
	oIter.Close()

	// Then crawl the Conscience objects
	dataDir, err := os.Open(filepath.Join(r.Path, ".git", CONSCIENCE_DATA_SUBDIR))
	if err == nil {
		defer dataDir.Close()

		entries, err := dataDir.Readdir(-1)
		if err != nil {
			return errors.Wrapf(err, "could not crawl conscience objects (path: %v)", r.Path)
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
func (r *Repo) HasObject(objectID []byte) bool {
	if len(objectID) == CONSCIENCE_HASH_LENGTH {
		p := filepath.Join(r.Path, ".git", CONSCIENCE_DATA_SUBDIR, hex.EncodeToString(objectID))
		_, err := os.Stat(p)
		return err == nil || !os.IsNotExist(err)

	} else if len(objectID) == GIT_HASH_LENGTH {
		var hash gitplumbing.Hash
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
			return nil, errors.WithStack(ErrObjectNotFound)
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			return nil, errors.Wrapf(err, "could not stat file '%v'", p)
		}

		or := &util.ObjectReader{
			Reader:     f,
			Closer:     f,
			ObjectType: 0,
			ObjectLen:  uint64(stat.Size()),
		}
		return or, nil

	} else if len(objectID) == GIT_HASH_LENGTH {
		hash := gitplumbing.Hash{}
		copy(hash[:], objectID)
		obj, err := r.Storer.EncodedObject(gitplumbing.AnyObject, hash)
		if err != nil {
			return nil, errors.Wrapf(err, "error fetching encoded git object from repo (path: %v, object: %v)", r.Path, hash.String())
		}

		r, err := obj.Reader()
		if err != nil {
			log.Errorf("WEIRD ERROR (@@todo: diagnose): %v", err)
			return nil, errors.WithStack(ErrObjectNotFound)
		}
		// It is the caller's responsibility to `.Close()` this reader, so we don't do it here.

		or := &util.ObjectReader{
			Reader:     r,
			Closer:     r,
			ObjectType: obj.Type(),
			ObjectLen:  uint64(obj.Size()),
		}
		return or, nil

	} else {
		return nil, errors.Errorf("objectID is wrong size (%v)", len(objectID))
	}
}

func (r *Repo) GetLocalRefs(ctx context.Context) (map[string]wire.Ref, error) {
	refs := map[string]wire.Ref{}

	err := util.ExecAndScanStdout(ctx, []string{"git", "show-ref"}, r.Path, func(line string) error {
		parts := strings.Split(line, " ")
		refs[parts[1]] = wire.Ref{RefName: parts[1], CommitHash: parts[0]}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return refs, nil
}

func writeConfig(path string, rawCfg *gitconfigformat.Config) error {
	p := filepath.Join(path, ".git", "config")
	f, err := os.OpenFile(p, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return errors.Wrapf(err, "could not open .git/config (path: %v)", path)
	}
	defer f.Close()

	w := io.Writer(f)

	enc := gitconfigformat.NewEncoder(w)
	err = enc.Encode(rawCfg)
	if err != nil {
		return errors.Wrapf(err, "could not encode git config (path: %v)", path)
	}
	return nil
}

func (r *Repo) SetupConfig(repoID string) error {
	cfg, err := r.Config()
	if err != nil {
		return errors.Wrapf(err, "could not get repo config (repoID: %v, path: %v)", repoID, r.Path)
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
		writeConfig(r.Path, raw)
	}

	// Check the remotes
	{
		remotes, err := r.Remotes()
		if err != nil {
			return errors.Wrapf(err, "could not read git remote config (repoID: %v, path: %v)", repoID, r.Path)
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
				return errors.Wrapf(err, "could not create remote (repoID: %v, path: %v)", repoID, r.Path)
			}
		}
	}

	return nil
}

func (r *Repo) AddUserToConfig(name string, email string) error {
	cfg, err := r.Config()
	if err != nil {
		return errors.Wrapf(err, "could not get repo config (path: %v)", r.Path)
	}
	raw := cfg.Raw
	changed := false
	if len(name) > 0 {
		raw.SetOption("user", "", "name", name)
		changed = true
	}
	if len(email) > 0 {
		raw.SetOption("user", "", "email", email)
		changed = true
	}
	if changed {
		err = writeConfig(r.Path, raw)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) PackfileWriter() (io.WriteCloser, error) {
	pfw, ok := r.Storer.(storer.PackfileWriter)
	if !ok {
		return nil, errors.Errorf("Repository storer is not a storer.PackfileWriter")
	}

	return pfw.PackfileWriter()
}
