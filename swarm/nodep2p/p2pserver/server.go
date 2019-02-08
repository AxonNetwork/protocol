package p2pserver

import (
	"context"
	"time"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodep2p"
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
