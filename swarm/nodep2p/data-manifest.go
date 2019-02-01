package nodep2p

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"io"
	"io/ioutil"
	"path/filepath"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/Conscience/protocol/repo"
	"github.com/pkg/errors"
)

type Chunk [repo.CONSCIENCE_HASH_LENGTH]byte

func getChunkManifest(r *repo.Repo, commitHash gitplumbing.Hash, fullHistory bool) (io.Reader, error) {
	if fullHistory {
		return getAllChunks(r)
	} else {
		return getChunksForCommit(r, commitHash)
	}
}

func getAllChunks(r *repo.Repo) (io.Reader, error) {
	p := filepath.Join(r.Path, ".git", repo.CONSCIENCE_DATA_SUBDIR)
	files, err := ioutil.ReadDir(p)
	if err != nil {
		return nil, err
	}

	rPipe, wPipe := io.Pipe()

	go func() {
		defer wPipe.Close()

		strLen := 2 * repo.CONSCIENCE_HASH_LENGTH
		for _, f := range files {
			name := f.Name()
			if len(name) != strLen {
				continue
			}
			hex, err := hex.DecodeString(name)
			if err != nil {
				wPipe.CloseWithError(err)
				break
			}
			n, err := wPipe.Write(hex)
			if err != nil {
				wPipe.CloseWithError(err)
				break
			} else if n != repo.CONSCIENCE_HASH_LENGTH {
				wPipe.CloseWithError(errors.Errorf("Did not send full chunk hash"))
				break
			}
		}
	}()

	return rPipe, nil
}

func getChunksForCommit(r *repo.Repo, commitHash gitplumbing.Hash) (io.Reader, error) {
	seen := make(map[gitplumbing.Hash]bool)

	commit, err := r.CommitObject(commitHash)
	if err != nil {
		return nil, err
	}

	tree, err := r.TreeObject(commit.TreeHash)
	if err != nil {
		return nil, err
	}

	rPipe, wPipe := io.Pipe()

	go func() {
		defer wPipe.Close()

		walker := gitobject.NewTreeWalker(tree, true, seen)
		for {
			_, entry, err := walker.Next()
			if err == io.EOF {
				walker.Close()
				break
			} else if err != nil {
				walker.Close()
				wPipe.CloseWithError(err)
				break
			}
			obj, err := r.Object(gitplumbing.AnyObject, entry.Hash)
			if err != nil {
				wPipe.CloseWithError(err)
				break
			}
			if obj.Type() != gitplumbing.BlobObject {
				continue
			}

			encoded, err := r.Storer.EncodedObject(gitplumbing.BlobObject, entry.Hash)
			if err != nil {
				wPipe.CloseWithError(err)
				break
			}

			reader, err := encoded.Reader()
			if err != nil {
				wPipe.CloseWithError(err)
				break
			}
			defer reader.Close()

			br := bufio.NewReader(reader)
			line, _, err := br.ReadLine()
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// empty file
				continue
			} else if err != nil {
				wPipe.CloseWithError(err)
				break
			}

			if bytes.Compare(line, []byte("CONSCIENCE_ENCODED")) != 0 {
				// not a chunked file
				continue
			}

			for {
				line, _, err := br.ReadLine()
				if err == io.EOF {
					break
				} else if err != nil {
					wPipe.CloseWithError(err)
					break
				}

				hex, err := hex.DecodeString(string(line))
				if err != nil {
					wPipe.CloseWithError(err)
					break
				}

				n, err := wPipe.Write(hex)
				if err != nil {
					wPipe.CloseWithError(err)
					break
				} else if n != repo.CONSCIENCE_HASH_LENGTH {
					wPipe.CloseWithError(errors.Errorf("Did not send full chunk hash"))
					break
				}
			}
			if err != nil {
				break
			}
		}
	}()

	return rPipe, nil
}
