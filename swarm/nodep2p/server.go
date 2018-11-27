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
	req := HandshakeRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}
	log.Debugf("[p2p server] incoming handshake %+v", req)

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
		err := WriteStructPacket(stream, &HandshakeResponse{Authorized: false})
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}
		return
	}

	repo := s.node.Repo(req.RepoID)
	commit, err := repo.HeadHash()
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}

	err = WriteStructPacket(stream, &HandshakeResponse{Authorized: true, Commit: commit})
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}

	repoID := req.RepoID
	go func() {
		defer stream.Close()

		for {
			req := GetPackfileRequest{}
			err := ReadStructPacket(stream, &req)
			if err != nil {
				log.Debugf("[p2p server] stream closed (err: %v)", err)
				return
			}

			log.Debugf("[p2p server] writing %v objects to packfile stream", len(req.ObjectIDs)/20)
			shouldClose := s.writePackfileToStream(repoID, req.ObjectIDs, stream)
			if shouldClose {
				return
			}
		}
	}()
}

func (s *Server) writePackfileToStream(repoID string, objectIDsFlattened []byte, stream netp2p.Stream) (shouldClose bool) {
	r := s.node.Repo(repoID)
	if r == nil {
		log.Warnf("[p2p server] cannot find repo %v", repoID)
		err := WriteStructPacket(stream, &GetPackfileResponse{ObjectIDs: []byte{}})
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return false
		}
		return false
	}

	availableObjectIDs := [][]byte{}
	for _, id := range UnflattenObjectIDs(objectIDsFlattened) {
		if r.HasObject(id) {
			availableObjectIDs = append(availableObjectIDs, id)
		}
	}

	err := WriteStructPacket(stream, &GetPackfileResponse{ObjectIDs: FlattenObjectIDs(availableObjectIDs)})
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return false
	}

	if len(availableObjectIDs) == 0 {
		return false
	}

	pr, pw := io.Pipe()
	go func() {
		var err error
		defer func() { pw.CloseWithError(err) }()

		cached := getCachedPackfile(availableObjectIDs)
		if cached != nil {
			_, err = io.Copy(pw, cached)
			if err != nil {
				log.Errorln("[p2p server] error reading cached packfile:", err)
			}
			return
		}

		cached = createCachedPackfile(availableObjectIDs)
		defer cached.Close()

		availableHashes := make([]gitplumbing.Hash, len(availableObjectIDs))
		for i := range availableObjectIDs {
			var hash gitplumbing.Hash
			copy(hash[:], availableObjectIDs[i])
			availableHashes[i] = hash
		}

		mw := io.MultiWriter(pw, cached)
		enc := packfile.NewEncoder(mw, r.Storer, false)
		_, err = enc.Encode(availableHashes, 10) // @@TODO: do we need to negotiate the packfile window with the client?
		if err != nil {
			log.Errorln("[p2p server] error encoding packfile:", err)
		}
	}()

	data := make([]byte, OBJ_CHUNK_SIZE)
	for {
		n, err := io.ReadFull(pr, data)
		if err == io.EOF {
			// no data was read
			break
		} else if err == io.ErrUnexpectedEOF {
			data = data[:n]
		} else if err != nil {
			log.Errorln("[p2p server] error reading packfile stream:", err)
			return true
		}

		err = WriteStructPacket(stream, &PackfileStreamChunk{Data: data})
		if err != nil {
			log.Errorln("[p2p server] error writing packfile stream:", err)
			return true
		}
	}

	err = WriteStructPacket(stream, &PackfileStreamChunk{End: true})
	if err != nil {
		log.Errorln("[p2p server] error ending packfile stream:", err)
		return true
	}

	return false
}

func getCachedPackfile(objectIDs [][]byte) *os.File {
	return nil
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

func (s *Server) HandleObjectStreamRequest(stream netp2p.Stream) {
	req := HandshakeRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}
	log.Debugf("[p2p server] incoming handshake %+v", req)

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
		err := WriteStructPacket(stream, &HandshakeResponse{Authorized: false})
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}
		return
	}

	repo := s.node.Repo(req.RepoID)
	commit, err := repo.HeadHash()
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}

	err = WriteStructPacket(stream, &HandshakeResponse{Authorized: true, Commit: commit})
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}

	go s.connectLoop(req.RepoID, stream)
}

func (s *Server) connectLoop(repoID string, stream netp2p.Stream) {
	defer stream.Close()

	for {
		req := GetObjectRequest{}
		err := ReadStructPacket(stream, &req)
		if err != nil {
			log.Debugf("[p2p server] stream closed")
			return
		}
		s.writeObjectToStream(repoID, req.ObjectID, stream)
	}
}

func (s *Server) writeObjectToStream(repoID string, objectID []byte, stream netp2p.Stream) {
	r := s.node.Repo(repoID)
	if r == nil {
		log.Warnf("[p2p server] cannot find repo %v", repoID)
		err := WriteStructPacket(stream, &GetObjectResponse{HasObject: false})
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}
		return
	}

	objectStream, err := r.OpenObject(objectID)
	if err != nil {
		log.Debugf("[p2p server] we don't have %v %0x (err: %v)", repoID, objectID, err)

		// tell the peer we don't have the object
		err := WriteStructPacket(stream, &GetObjectResponse{HasObject: false})
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}
		return
	}
	defer objectStream.Close()

	err = WriteStructPacket(stream, &GetObjectResponse{
		Unauthorized: false,
		HasObject:    true,
		ObjectID:     objectID,
		ObjectType:   objectStream.Type(),
		ObjectLen:    objectStream.Len(),
	})
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}

	sent, err := io.Copy(stream, objectStream)
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	} else if uint64(sent) < objectStream.Len() {
		log.Errorf("[p2p server] terminated while sending")
		return
	}

	// log.Infof("[p2p server] successfully sent %v (%v bytes) (%v ms)", hex.EncodeToString(objectID), sent, time.Now().Sub(start).Seconds()*1000)
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

	addr, err := s.node.AddrFromSignedHash([]byte(req.Commit), req.Signature)
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
		err := WriteStructPacket(stream, &GetManifestResponse{Authorized: false})
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}
		return
	}

	// Send our response:
	// 1. peer is not authorized to pull
	//    - GetManifestResponse{Authorized: false}
	//    - <close connection>
	// 2. we don't have the repo/commit:
	//    - GetCommitResponse{HasCommit: false}
	//    - <close connection>
	// 3. we do have the commit:
	//    - GetCommitResponse{Authorized: true, HasCommit: true, ManifestLen: ...}
	//    - [stream of manifest bytes...]
	//    - <close connection>
	//
	r := s.node.Repo(req.RepoID)
	if r == nil {
		log.Warnf("[p2p server] cannot find repo %v", req.RepoID)
		err := WriteStructPacket(stream, &GetManifestResponse{HasCommit: false})
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}
		return
	}

	manifest, err := getManifest(r)
	if err != nil {
		log.Warnf("[p2p server] cannot get manifest for repo %v", req.RepoID)
		err := WriteStructPacket(stream, &GetManifestResponse{HasCommit: false})
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}
		return
	}

	err = WriteStructPacket(stream, &GetManifestResponse{
		Authorized:  true,
		HasCommit:   true,
		ManifestLen: len(manifest),
	})
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
