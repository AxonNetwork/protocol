package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	RepoID  string
	Path    string
	Objects map[string]ObjectEntry
}

type ObjectEntry struct {
	ID   []byte
	Type gitplumbing.ObjectType
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
	ErrBadChecksum  = fmt.Errorf("chunk error: bad checksum")
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

			// @@TODO: read the contents of each chunk and compare its name to its hash?
			id, err := hex.DecodeString(entry.Name())
			if err != nil {
				log.Errorf("bad conscience data chunk name: %v", entry.Name())
				continue
			} else if len(id) != CONSCIENCE_HASH_LENGTH {
				log.Errorf("bad conscience data chunk name: %v", entry.Name())
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

// Returns true if the chunk is known, false otherwise.
func (rm *RepoManager) HasObject(repoID string, chunkID []byte) bool {
	repoEntry, ok := rm.repos[repoID]
	if !ok {
		return false
	}
	_, ok = repoEntry.Objects[string(chunkID)]
	return ok
}

func (rm *RepoManager) Object(repoID string, chunkID []byte) (ObjectEntry, bool) {
	repoEntry, ok := rm.repos[repoID]
	if !ok {
		return ObjectEntry{}, false
	}

	entry, ok := repoEntry.Objects[string(chunkID)]
	if !ok {
		return ObjectEntry{}, false
	}

	return entry, true
}

// Open a chunk for reading.  It is the caller's responsibility to .Close() the chunk when finished.
func (rm *RepoManager) OpenChunk(repoID string, chunkID []byte) (io.ReadCloser, error) {
	repoEntry, ok := rm.repos[repoID]
	if !ok {
		return nil, ErrRepoNotFound
	}

	if len(chunkID) == CONSCIENCE_HASH_LENGTH {
		// Open a Conscience chunk
		p := filepath.Join(repoEntry.Path, ".git", CONSCIENCE_DATA_SUBDIR, hex.EncodeToString(chunkID))
		f, err := os.Open(p)
		if err != nil {
			return nil, ErrChunkNotFound
		}
		return f, nil

	} else if len(chunkID) == GIT_HASH_LENGTH {
		// Open a Git chunk

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
	} else {
		return nil, fmt.Errorf("chunkID is wrong size (%v)", len(chunkID))
	}
}

// Create a new chunk and fill it with the data from the provided io.Reader.  The chunk is saved to
// disk.  If the hash of the data does not equal the provided chunkID, this function returns an
// error.  When creating a Conscience chunk, gitObjectType can simply be set to 0.
func (rm *RepoManager) CreateChunk(repoID string, chunkID []byte, gitObjectType gitplumbing.ObjectType, r io.Reader) (int64, error) {
	repoEntry, ok := rm.repos[repoID]
	if !ok {
		return 0, ErrRepoNotFound
	}

	var n int64

	if len(chunkID) == CONSCIENCE_HASH_LENGTH {
		// If this is a Conscience chunk, just write directly to the file system.

		// Make sure the .git/data dir exists.
		err := os.MkdirAll(filepath.Join(repoEntry.Path, ".git", CONSCIENCE_DATA_SUBDIR), 0777)
		if err != nil {
			return 0, err
		}

		p := filepath.Join(repoEntry.Path, ".git", CONSCIENCE_DATA_SUBDIR, hex.EncodeToString(chunkID))
		f, err := os.Create(p)
		if err != nil {
			return 0, ErrRepoNotFound
		}

		hasher := sha256.New()
		reader := io.TeeReader(r, hasher)
		n, err = io.Copy(f, reader)
		if err != nil {
			f.Close()
			os.Remove(p)
			return 0, err
		}

		if !bytes.Equal(chunkID, hasher.Sum(nil)) {
			f.Close()
			os.Remove(p)
			return 0, ErrBadChecksum
		}

	} else if len(chunkID) == GIT_HASH_LENGTH {
		// if this is a Git chunk, store it in the objects folder
		repo, err := git.PlainOpen(repoEntry.Path)
		if err != nil {
			return 0, err
		}

		obj := repo.Storer.NewEncodedObject() // returns a &plumbing.MemoryObject{}
		obj.SetType(gitObjectType)

		w, err := obj.Writer()
		if err != nil {
			return 0, err
		}

		n, err = io.Copy(w, r)
		if err != nil {
			return 0, err
		}

		err = w.Close()
		if err != nil {
			return 0, err
		}

		// Check the checksum
		h := obj.Hash()
		if !bytes.Equal(chunkID, h[:]) {
			return 0, ErrBadChecksum
		}

		// Write the object to disk
		_, err = repo.Storer.SetEncodedObject(obj)
		if err != nil {
			return 0, err
		}

	} else {
		return 0, fmt.Errorf("chunkID is wrong size (%v)", len(chunkID))
	}

	rm.repos[repoID].Objects[string(chunkID)] = ObjectEntry{ID: chunkID, Type: gitObjectType}

	return n, nil
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
