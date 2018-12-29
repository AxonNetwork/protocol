package nodep2p

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	netp2p "github.com/libp2p/go-libp2p-net"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/packfile"

	"github.com/Conscience/protocol/log"
	. "github.com/Conscience/protocol/swarm/wire"
)

type Server struct {
	node INode
}

func NewServer(node INode) *Server {
	return &Server{node}
}

func (s *Server) HandlePackfileStreamRequest(stream netp2p.Stream) {
	defer stream.Close()

	req := GetPackfileRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}
	log.Debugf("[p2p server] incoming handshake")

	// Ensure the client has access
	{
		addr, err := s.node.AddrFromSignedHash([]byte(req.RepoID), req.Signature)
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
			err := WriteStructPacket(stream, &GetPackfileResponse{ErrUnauthorized: true})
			if err != nil {
				log.Errorf("[p2p server] %v", err)
				return
			}
			return
		}
	}

	// Write the packfile to the stream
	{
		log.Debugf("[p2p server] writing %v objects to packfile stream", len(req.ObjectIDs)/20)

		r := s.node.Repo(req.RepoID)
		if r == nil {
			log.Warnf("[p2p server] cannot find repo %v", req.RepoID)
			err := WriteStructPacket(stream, &GetPackfileResponse{ObjectIDs: []byte{}})
			if err != nil {
				log.Errorf("[p2p server] %v", err)
				return
			}
			return
		}

		availableObjectIDs := [][]byte{}
		for _, id := range UnflattenObjectIDs(req.ObjectIDs) {
			if r.HasObject(id) {
				availableObjectIDs = append(availableObjectIDs, id)
			}
		}

		err = WriteStructPacket(stream, &GetPackfileResponse{ObjectIDs: FlattenObjectIDs(availableObjectIDs)})
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}

		if len(availableObjectIDs) == 0 {
			return
		}

		cached := getCachedPackfile(availableObjectIDs)
		if cached != nil {
			defer cached.Close()

			log.Infoln("[p2p server] using cached packfile")

			_, err = io.Copy(stream, cached)
			if err != nil {
				log.Errorln("[p2p server] error reading cached packfile:", err)
			}
			return
		}

		log.Infoln("[p2p server] caching new packfile")
		cached = createCachedPackfile(availableObjectIDs)
		defer cached.Close()

		availableHashes := make([]gitplumbing.Hash, len(availableObjectIDs))
		for i := range availableObjectIDs {
			copy(availableHashes[i][:], availableObjectIDs[i])
		}

		mw := io.MultiWriter(stream, cached)
		enc := packfile.NewEncoder(mw, r.Storer, false)
		_, err = enc.Encode(availableHashes, 10) // @@TODO: do we need to negotiate the packfile window with the client?
		if err != nil {
			log.Errorln("[p2p server] error encoding packfile:", err)
		}
	}
}

func getCachedPackfile(objectIDs [][]byte) *os.File {
	sorted := SortByteSlices(objectIDs)
	flattened := FlattenObjectIDs(sorted)
	hash := sha256.Sum256(flattened)
	hex := fmt.Sprintf("%0x", hash[:])

	cacheDir := filepath.Join(os.TempDir(), "conscience-packfile-cache")
	err := os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		log.Errorln("[p2p server] getCachedPackfile:", err)
		return nil
	}

	f, err := os.Open(filepath.Join(cacheDir, hex))
	if os.IsNotExist(err) {
		return nil
	}
	return f
}

func createCachedPackfile(objectIDs [][]byte) *os.File {
	sorted := SortByteSlices(objectIDs)
	flattened := FlattenObjectIDs(sorted)
	hash := sha256.Sum256(flattened)
	hex := fmt.Sprintf("%0x", hash[:])

	cacheDir := filepath.Join(os.TempDir(), "conscience-packfile-cache")
	err := os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		log.Errorln("[p2p server] createCachedPackfile:", err)
		return nil
	}

	log.Infoln("[p2p server] caching packfile", filepath.Join(cacheDir, hex))
	f, err := os.Create(filepath.Join(cacheDir, hex))
	if os.IsNotExist(err) {
		return nil
	}
	return f
}

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

	manifest, err := getManifest(r, req.Commit)
	if err != nil {
		log.Warnf("[p2p server] cannot get manifest for repo %v", req.RepoID)
		err := WriteStructPacket(stream, &GetManifestResponse{ErrMissingCommit: true})
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}
		return
	}

	err = WriteStructPacket(stream, &GetManifestResponse{ManifestLen: len(manifest)})
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}

	for i := range manifest {
		object := manifest[i]
		err = WriteStructPacket(stream, &object)
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}
	}

	log.Printf("[p2p server] sent manifest for %v %v", req.RepoID, req.Commit)
}
