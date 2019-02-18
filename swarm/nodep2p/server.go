package nodep2p

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/libgit2/git2go"
	netp2p "github.com/libp2p/go-libp2p-net"

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

		// cached := getCachedPackfile(availableObjectIDs)
		// if cached != nil {
		// 	defer cached.Close()

		// 	log.Infoln("[p2p server] using cached packfile")

		// 	_, err = io.Copy(stream, cached)
		// 	if err != nil {
		// 		log.Errorln("[p2p server] error reading cached packfile:", err)
		// 	}
		// 	return
		// }

		// log.Infoln("[p2p server] caching new packfile")
		// cached = createCachedPackfile(availableObjectIDs)
		// defer cached.Close()

		availableHashes := make([]git.Oid, len(availableObjectIDs))
		for i := range availableObjectIDs {
			copy(availableHashes[i][:], availableObjectIDs[i])
		}

		packbuilder, err := r.NewPackbuilder()
		if err != nil {
			log.Errorln("[p2p server] error instantiating packbuilder:", err)
			return
		}

		for i := range availableHashes {
			err = packbuilder.Insert(&availableHashes[i], "")
			if err != nil {
				log.Errorln("[p2p server] error adding object to packbuilder:", err)
				return
			}
		}

		// This multiwriter will write the packfile to both an on-disk cache as well as the p2p stream.
		// mw := io.MultiWriter(stream, cached)
		err = packbuilder.Write(stream)
		if err != nil {
			log.Errorln("[p2p server] error writing packfile to stream:", err)
			return
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
		log.Warnf("[p2p server] address 0x%0x does not have pull access to repo %v", addr.Bytes(), req.RepoID)
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
		log.Warnf("[p2p server] cannot get manifest for repo %v", req.RepoID, err)
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
