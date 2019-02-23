package p2pserver

import (
	"bufio"
	"context"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/libgit2/git2go"
	netp2p "github.com/libp2p/go-libp2p-net"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	. "github.com/Conscience/protocol/swarm/wire"
)

// Handles incoming requests for commit manifests
func (s *Server) HandleManifestRequest(stream netp2p.Stream) {
	defer stream.Close()

	// Read the request packet
	req := GetManifestRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[manifest server] %+v", errors.WithStack(err))
		return
	}

	addr, err := s.node.AddrFromSignedHash(req.Commit[:], req.Signature)
	if err != nil {
		log.Errorf("[manifest server] %+v", errors.WithStack(err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hasAccess, err := s.node.AddressHasPullAccess(ctx, addr, req.RepoID)
	if err != nil {
		log.Errorf("[manifest server] %+v", errors.WithStack(err))
		return
	}

	if hasAccess == false {
		log.Warnf("[manifest server] address 0x%0x does not have pull access", addr.Bytes())
		err := WriteStructPacket(stream, &GetManifestResponse{ErrUnauthorized: true})
		if err != nil {
			log.Errorf("[manifest server] %+v", errors.WithStack(err))
			return
		}
		return
	}

	r := s.node.Repo(req.RepoID)
	if r == nil {
		log.Warnf("[manifest server] cannot find repo %v", req.RepoID)
		err := WriteStructPacket(stream, &GetManifestResponse{ErrMissingCommit: true})
		if err != nil {
			log.Errorf("[manifest server] %+v", errors.WithStack(err))
			return
		}
		return
	}

	err = WriteStructPacket(stream, &GetManifestResponse{SendingManifest: true})
	if err != nil {
		log.Errorf("[manifest server] %+v", errors.WithStack(err))
		return
	}

	manifest := getManifestStream(r, req.Commit, CheckoutType(req.CheckoutType))
	_, err = io.Copy(stream, manifest)
	if err != nil {
		log.Errorf("[manifest server] %+v", errors.WithStack(err))
		return
	}

	err = WriteStructPacket(stream, &ManifestObject{End: true})
	if err != nil {
		log.Errorf("[manifest server] %+v", errors.WithStack(err))
		return
	}

	log.Printf("[manifest server] sent manifest for %v %v", req.RepoID, req.Commit)
}

func getManifestStream(r *repo.Repo, commitHash git.Oid, checkoutType CheckoutType) io.Reader {
	seenObj := make(map[git.Oid]bool)
	seenChunks := make(map[string]bool)
	stack := []git.Oid{commitHash}

	rPipe, wPipe := io.Pipe()

	go func() {
		var err error
		defer func() { wPipe.CloseWithError(err) }()

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
					innerErr = writeGitOid(wPipe, r, *entry.Id, seenObj)
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
					(checkoutType == Full || (checkoutType == Working && commitHash == *commit.Id())) {

					innerErr = writeChunkIDsIfChunked(r, *entry.Id, seenChunks, wPipe)
					if innerErr != nil {
						return -1
					}
				}

				return 0
			})
			if innerErr != nil {
				err = innerErr
				return
			} else if err != nil {
				return
			}

			err = writeGitOid(wPipe, r, *commit.Id(), seenObj)
			if err != nil {
				return
			}

			err = writeGitOid(wPipe, r, *commit.TreeId(), seenObj)
			if err != nil {
				return
			}
		}
	}()

	return rPipe
}

func writeGitOid(stream io.Writer, r *repo.Repo, oid git.Oid, seenObj map[git.Oid]bool) error {
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

	reader, err := odb.NewReadStream(&oid)
	if err != nil {
		return err
	}
	defer reader.Close()

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

	return nil
}
