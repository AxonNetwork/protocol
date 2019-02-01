package nodep2p

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"time"

	netp2p "github.com/libp2p/go-libp2p-net"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"

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
		log.Errorf("[p2p server] %v", err)
		return
	}

	addr, err := s.node.AddrFromSignedHash(req.Commit[:], req.Signature)
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hasAccess, err := s.node.AddressHasPullAccess(ctx, addr, req.RepoID)
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}

	if hasAccess == false {
		log.Warnf("[p2p server] address 0x%0x does not have pull access", addr.Bytes())
		err := WriteStructPacket(stream, &GetManifestResponse{ErrUnauthorized: true})
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}
		return
	}

	r := s.node.Repo(req.RepoID)
	if r == nil {
		log.Warnf("[p2p server] cannot find repo %v", req.RepoID)
		err := WriteStructPacket(stream, &GetManifestResponse{ErrMissingCommit: true})
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}
		return
	}

	err = WriteStructPacket(stream, &GetManifestResponse{SendingManifest: true})
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}

	manifest := getManifestStream(r, req.Commit, CheckoutType(req.CheckoutType))
	_, err = io.Copy(stream, manifest)
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}

	err = WriteStructPacket(stream, &ManifestObject{End: true})
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}

	log.Printf("[p2p server] sent manifest for %v %v", req.RepoID, req.Commit)
}

func getManifestStream(r *repo.Repo, commitHash gitplumbing.Hash, checkoutType CheckoutType) io.Reader {
	seenObj := make(map[gitplumbing.Hash]bool)
	seenChunks := make(map[string]bool)
	stack := []gitplumbing.Hash{commitHash}

	rPipe, wPipe := io.Pipe()

	go func() {
		defer wPipe.Close()
		for len(stack) > 0 {
			if seenObj[stack[0]] {
				stack = stack[1:]
				continue
			}

			commit, err := r.CommitObject(stack[0])
			if err != nil {
				wPipe.CloseWithError(err)
				return
			}

			parentHashes := []gitplumbing.Hash{}
			for _, h := range commit.ParentHashes {
				if _, wasSeen := seenObj[h]; !wasSeen {
					parentHashes = append(parentHashes, h)
				}
			}

			stack = append(stack[1:], parentHashes...)

			// Walk the tree for this commit
			tree, err := r.TreeObject(commit.TreeHash)
			if err != nil {
				wPipe.CloseWithError(err)
				return
			}

			walker := gitobject.NewTreeWalker(tree, true, seenObj)

			for {
				_, entry, err := walker.Next()
				if err == io.EOF {
					walker.Close()
					break
				} else if err != nil {
					walker.Close()
					wPipe.CloseWithError(err)
					return
				}
				obj, err := r.Object(gitplumbing.AnyObject, entry.Hash)
				if err != nil {
					log.Printf("[err] error on r.Object: %v\n", err)
					continue
				}
				switch obj.Type() {
				case gitplumbing.TreeObject:
				case gitplumbing.BlobObject:
					err = writeGitHashToStream(r, entry.Hash, seenObj, wPipe)
					if err != nil {
						wPipe.CloseWithError(err)
						return
					}

				default:
					log.Printf("found weird object: %v (%v)\n", entry.Hash.String(), obj.Type())
				}

				// full checkout or if this is a blob object for the first commit
				if checkoutType == Full || (checkoutType == Working && commitHash == commit.Hash) {
					if obj.Type() == gitplumbing.BlobObject {
						err = writeChunksForHash(r, entry.Hash, seenChunks, wPipe)
						if err != nil {
							wPipe.CloseWithError(err)
							return
						}
					}
				}
			}

			err = writeGitHashToStream(r, commit.Hash, seenObj, wPipe)
			if err != nil {
				wPipe.CloseWithError(err)
				return
			}

			err = writeGitHashToStream(r, commit.TreeHash, seenObj, wPipe)
			if err != nil {
				wPipe.CloseWithError(err)
				return
			}
		}
	}()

	return rPipe
}

func writeGitHashToStream(r *repo.Repo, hash gitplumbing.Hash, seenObj map[gitplumbing.Hash]bool, stream io.Writer) error {
	if seenObj[hash] == true {
		return nil
	}
	seenObj[hash] = true

	size, err := r.Storer.EncodedObjectSize(hash)
	if err != nil {
		return err
	}

	object := ManifestObject{
		End:              false,
		Hash:             hash[:],
		UncompressedSize: size,
	}

	return WriteStructPacket(stream, &object)
}

func writeChunksForHash(r *repo.Repo, hash gitplumbing.Hash, seenChunks map[string]bool, stream io.Writer) error {
	obj, err := r.Storer.EncodedObject(gitplumbing.BlobObject, hash)
	if err != nil {
		return err
	}

	reader, err := obj.Reader()
	if err != nil {
		return err
	}
	defer reader.Close()

	br := bufio.NewReader(reader)

	line, _, err := br.ReadLine()
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return nil
	} else if err != nil {
		return err
	}

	if bytes.Compare(line, []byte("CONSCIENCE_ENCODED")) != 0 {
		// not a chunked file
		return nil
	}

	for {
		line, _, err := br.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		seenChunks[string(line)] = true

		hex, err := hex.DecodeString(string(line))
		if err != nil {
			return err
		}

		p := filepath.Join(r.Path, ".git", repo.CONSCIENCE_DATA_SUBDIR)
		stat, err := os.Stat(p)
		if err != nil {
			return err
		}

		object := ManifestObject{
			End:              false,
			Hash:             hex,
			UncompressedSize: stat.Size(),
		}

		err = WriteStructPacket(stream, &object)
		if err != nil {
			return err
		}
	}

	return nil
}
