package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/libgit2/git2go"

	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

func main() {
	r, err := repo.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	head, err := r.Head()
	if err != nil {
		panic(err)
	}

	start := time.Now()
	gitObjects := []ManifestObject{}
	chunkObjects := []ManifestObject{}
	var totalSize int64

	stream := getManifestStream(r, *head.Target(), CheckoutTypeFull)
	for {
		obj := ManifestObject{}
		err = ReadStructPacket(stream, &obj)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if obj.End == true {
			break
		}
		if len(obj.Hash) == repo.GIT_HASH_LENGTH {
			gitObjects = append(gitObjects, obj)

		} else {
			chunkObjects = append(chunkObjects, obj)
		}
		totalSize += obj.UncompressedSize
	}

	total := time.Now().Sub(start)
	fmt.Println("gitObjects =", len(gitObjects))
	fmt.Println("chunkObjects =", len(chunkObjects))
	fmt.Println("total size =", util.HumanizeBytes(float64(totalSize)))
	fmt.Println("total time =", total)
	runtime.KeepAlive(r)
}

func getManifestStream(r *repo.Repo, commitHash git.Oid, checkoutType CheckoutType) io.Reader {
	seenObj := make(map[git.Oid]bool)
	seenChunks := make(map[string]bool)
	stack := []git.Oid{commitHash}

	rPipe, wPipe := io.Pipe()

	go func() {
		var err error
		defer func() { wPipe.CloseWithError(err) }()

		odb, err := r.Odb()
		if err != nil {
			panic(err)
		}

		for len(stack) > 0 {
			if seenObj[stack[0]] {
				stack = stack[1:]
				continue
			}

			commit, err := r.LookupCommit(&stack[0])
			if err != nil {
				return
			}

			parentCount := commit.ParentCount()
			parentHashes := []git.Oid{}
			for i := uint(0); i < parentCount; i++ {
				hash := commit.ParentId(i)
				if _, wasSeen := seenObj[*hash]; !wasSeen {
					parentHashes = append(parentHashes, *hash)
				}
			}

			stack = append(stack[1:], parentHashes...)

			tree, err := commit.Tree()
			if err != nil {
				return
			}

			var obj *git.Object
			var innerErr error
			err = tree.Walk(func(name string, entry *git.TreeEntry) int {
				obj, innerErr = r.Lookup(entry.Id)
				if innerErr != nil {
					return -1
				}

				switch obj.Type() {
				case git.ObjectTree, git.ObjectBlob:
					innerErr = writeGitOid(wPipe, r, odb, *entry.Id, seenObj)
					if innerErr != nil {
						return -1
					}
				default:
					log.Printf("found weird object: %v (%v)\n", entry.Id.String(), obj.Type().String())
					return 0
				}

				// Only write large file chunk hashes to the stream if:
				// 1. this is a full checkout, or...
				// 2. this is a working checkout and we're on the first (i.e. checked out) commit
				// @@TODO: can we assume that the requested commit is the one that will be checked out?
				if obj.Type() == git.ObjectBlob &&
					(checkoutType == CheckoutTypeFull || (checkoutType == CheckoutTypeWorking && commitHash == *commit.Id())) {

					// innerErr = writeChunkIDsIfChunked(r, odb, *entry.Id, seenChunks, wPipe)
					// if innerErr != nil {
					// 	return -1
					// }
				}

				return 0
			})
			if innerErr != nil {
				err = innerErr
				return
			} else if err != nil {
				return
			}

			err = writeGitOid(wPipe, r, odb, *commit.Id(), seenObj)
			if err != nil {
				return
			}

			err = writeGitOid(wPipe, r, odb, *commit.TreeId(), seenObj)
			if err != nil {
				return
			}
		}
	}()

	return rPipe
}

func writeGitOid(stream io.Writer, r *repo.Repo, odb *git.Odb, oid git.Oid, seenObj map[git.Oid]bool) error {
	if seenObj[oid] == true {
		return nil
	}
	seenObj[oid] = true

	odb, err := r.Odb()
	if err != nil {
		return err
	}

	size, _, err := odb.ReadHeader(&oid)
	if err != nil {
		return err
	}

	object := ManifestObject{
		End:              false,
		Hash:             oid[:],
		UncompressedSize: int64(size),
	}

	return WriteStructPacket(stream, &object)
}

func writeChunkIDsIfChunked(r *repo.Repo, oid git.Oid, seenChunks map[string]bool, stream io.Writer) error {
	odb, err := r.Odb()
	if err != nil {
		return err
	}

	odbObj, err := odb.Read(&oid)
	if err != nil {
		return err
	}

	reader, err := odb.NewReadStream(&oid)
	if err != nil {
		return err
	}

	br := bufio.NewReader(reader)

	header, err := br.Peek(18)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return nil
	} else if err != nil {
		return err
	} else if string(header) != "CONSCIENCE_ENCODED" {
		// not a chunked file
		return nil
	}

	// Discard the first line, it's just the header
	_, _, err = br.ReadLine()
	if err != nil {
		return err
	}

	for {
		line, _, err := br.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		seenChunks[string(line)] = true

		chunkID, err := hex.DecodeString(string(line))
		if err != nil {
			return err
		}

		p := filepath.Join(r.Path(), ".git", repo.CONSCIENCE_DATA_SUBDIR, string(line))
		stat, err := os.Stat(p)
		if err != nil {
			return err
		}

		object := ManifestObject{
			End:              false,
			Hash:             chunkID,
			UncompressedSize: stat.Size(),
		}

		err = WriteStructPacket(stream, &object)
		if err != nil {
			return err
		}
	}

	runtime.KeepAlive(odbObj)

	return nil
}
