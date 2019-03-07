package p2pserver

import (
	"bufio"
	"bytes"
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

	manifestStream, err := getManifestStream(r, req.Commit, CheckoutType(req.CheckoutType))
	if err != nil {
		log.Errorf("[manifest server] %+v", errors.WithStack(err))
		return
	}

	_, err = io.Copy(stream, manifestStream)
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

func getManifestStream(r *repo.Repo, commitHash git.Oid, checkoutType CheckoutType) (io.Reader, error) {
	stack := []git.Oid{commitHash}

	odb, err := r.Odb()
	if err != nil {
		return nil, err
	}

	rPipe, wPipe := io.Pipe()

	m := &ManifestWriter{
		repo:           r,
		odb:            odb,
		checkoutType:   checkoutType,
		checkoutCommit: commitHash,
		seenObj:        make(map[git.Oid]bool),
		seenChunks:     make(map[string]bool),
		writeStream:    wPipe,
	}

	go func() {
		var err error
		defer func() {
			odb.Free()
			wPipe.CloseWithError(err)
		}()

		for len(stack) > 0 {
			if m.seenObj[stack[0]] {
				stack = stack[1:]
				continue
			}

			func() {
				commit, err := m.repo.LookupCommit(&stack[0])
				if err != nil {
					return
				}
				defer commit.Free()

				parentCount := commit.ParentCount()
				parentHashes := []git.Oid{}
				for i := uint(0); i < parentCount; i++ {
					hash := commit.ParentId(i)
					if !m.seenObj[*hash] {
						parentHashes = append(parentHashes, *hash)
					}
				}

				stack = append(stack[1:], parentHashes...)

				err = m.addCommit(commit)
				if err != nil {
					return
				}
			}()
		}
	}()

	return rPipe, nil
}

type ManifestWriter struct {
	repo           *repo.Repo
	odb            *git.Odb
	checkoutType   CheckoutType
	checkoutCommit git.Oid
	seenObj        map[git.Oid]bool
	seenChunks     map[string]bool
	writeStream    io.Writer
}

func (m *ManifestWriter) addCommit(commit *git.Commit) error {
	tree, err := commit.Tree()
	if err != nil {
		return err
	}
	defer tree.Free()

	var innerErr error
	err = tree.Walk(func(name string, entry *git.TreeEntry) int {
		_, objType, err := m.odb.ReadHeader(entry.Id)
		if err != nil {
			innerErr = err
			return -1
		}

		switch objType {
		case git.ObjectTree, git.ObjectBlob:
			innerErr = m.writeGitOid(*entry.Id)
			if innerErr != nil {
				return -1
			}
		default:
			log.Printf("found weird object: %v (%v)\n", entry.Id.String(), objType.String())
			return 0
		}

		// Only write large file chunk hashes to the stream if:
		// 1. this is a full checkout, or...
		// 2. this is a working checkout and we're on the first (i.e. checked out) commit
		// @@TODO: can we assume that the requested commit is the one that will be checked out?
		if objType == git.ObjectBlob &&
			(m.checkoutType == CheckoutTypeFull || (m.checkoutType == CheckoutTypeWorking && m.checkoutCommit == *commit.Id())) {

			isChunked, err := m.repo.FileIsChunked(name, commit.Id())
			if err != nil {
				innerErr = err
				return -1
			}

			if isChunked {
				innerErr = m.writeChunkIDsForBlob(*entry.Id)
				if innerErr != nil {
					return -1
				}
			}
		}

		return 0
	})
	if innerErr != nil {
		return innerErr
	} else if err != nil {
		return err
	}

	err = m.writeGitOid(*commit.Id())
	if err != nil {
		return err
	}

	err = m.writeGitOid(*commit.TreeId())
	if err != nil {
		return err
	}
	return nil
}

func (m *ManifestWriter) writeGitOid(oid git.Oid) error {
	if m.seenObj[oid] == true {
		return nil
	}
	m.seenObj[oid] = true

	size, _, err := m.odb.ReadHeader(&oid)
	if err != nil {
		return err
	}

	return WriteStructPacket(m.writeStream, &ManifestObject{
		Hash:             oid[:],
		UncompressedSize: int64(size),
	})
}

func (m *ManifestWriter) writeChunkIDsForBlob(oid git.Oid) error {
	odbObject, err := m.odb.Read(&oid)
	if err != nil {
		return err
	}
	// It's necessary to manually .Free this object because it must stay live until we're done with
	// its .Data slice.
	defer odbObject.Free()

	br := bufio.NewReader(bytes.NewReader(odbObject.Data()))

	repoRoot := m.repo.Path()
	for {
		line, _, err := br.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if m.seenChunks[string(line)] {
			continue
		}
		m.seenChunks[string(line)] = true

		chunkID, err := hex.DecodeString(string(line))
		if err != nil {
			return err
		}

		p := filepath.Join(repoRoot, ".git", repo.CONSCIENCE_DATA_SUBDIR, string(line))
		stat, err := os.Stat(p)
		if err != nil {
			return err
		}

		err = WriteStructPacket(m.writeStream, &ManifestObject{
			Hash:             chunkID,
			UncompressedSize: stat.Size(),
		})
		if err != nil {
			return err
		}
	}

	return nil
}
