package nodep2p

import (
	"context"
	"io"
	"time"

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

	flatHead, flatHistory, err := r.GetManifest()
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
		Authorized: true,
		HasCommit:  true,
		HeadLen:    len(flatHead),
		HistoryLen: len(flatHistory),
	})
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}

	sent, err := stream.Write(flatHead)
	if err != nil {
		log.Errorf("[p2p server] %v", err)
	} else if sent < len(flatHead) {
		log.Errorf("[p2p server] terminated while sending head")
	}

	sent, err = stream.Write(flatHistory)
	if err != nil {
		log.Errorf("[p2p server] %v", err)
	} else if sent < len(flatHistory) {
		log.Errorf("[p2p server] terminated while sending history")
	}

	log.Printf("[p2p server] sent manifest for %v %v (%v bytes)", req.RepoID, req.Commit, sent)
}
