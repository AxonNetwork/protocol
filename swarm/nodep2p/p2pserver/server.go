package p2pserver

import (
	"context"
	"time"

	netp2p "github.com/libp2p/go-libp2p-net"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodep2p"
	. "github.com/Conscience/protocol/swarm/wire"
)

type Server struct {
	node nodep2p.INode
}

func NewServer(node nodep2p.INode) *Server {
	return &Server{node}
}

func (s *Server) isAuthorised(repoID string, sig []byte) (bool, error) {
	addr, err := s.node.AddrFromSignedHash([]byte(repoID), sig)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	isAuth, err := s.node.AddressHasPullAccess(ctx, addr, repoID)
	if err != nil {
		return false, err
	}

	if isAuth == false {
		log.Warnf("[p2p server] address 0x%0x does not have pull access", addr.Bytes())
	}

	return isAuth, nil
}

func (s *Server) HandleHandshakeRequest(stream netp2p.Stream) {
	req := HandshakeRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}
	log.Debugf("[p2p server] incoming handshake")
	// Ensure the client has access
	{
		isAuth, err := s.isAuthorised(req.RepoID, req.Signature)
		if err != nil {
			log.Errorf("[p2p server] %v", err)
			return
		}

		if isAuth == false {
			err := WriteStructPacket(stream, &HandshakeResponse{ErrUnauthorized: true})
			if err != nil {
				log.Errorf("[p2p server] %v", err)
				return
			}
			return
		}
	}
	err = WriteStructPacket(stream, &HandshakeResponse{})
	if err != nil {
		log.Errorf("[p2p server] %v", err)
		return
	}
	go func() {

	}()
}
