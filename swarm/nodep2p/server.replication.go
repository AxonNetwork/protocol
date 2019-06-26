package nodep2p

import (
	"context"
	"time"

	netp2p "github.com/libp2p/go-libp2p-net"

	"github.com/Conscience/protocol/log"
	. "github.com/Conscience/protocol/swarm/wire"
)

// Handles an incoming request to replicate (pull changes from) a given repository.
func (s *Server) HandleReplicationRequest(stream netp2p.Stream) {
	log.Printf("[replication server] receiving replication request")
	defer stream.Close()

	var req ReplicationRequest
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[replication server] error: %v", err)
		return
	}

	// @@TODO: make context timeout configurable
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := s.node.GetConfig()

	err = Replicate(ctx, req.RepoID, s.node, cfg.Node.ReplicationPolicies[req.RepoID], func(current, total uint64) error {
		return WriteStructPacket(stream, &Progress{Current: current, Total: total})
	})
	if err != nil {
		log.Errorf("[replication server] error: %v", err)
		err = WriteStructPacket(stream, &Progress{ErrorMsg: err.Error()})
		if err != nil {
			log.Errorf("[replication server] error: %v", err)
		}
		return
	}

	err = WriteStructPacket(stream, &Progress{Done: true})
	if err != nil {
		log.Errorf("[replication server] error: %v", err)
		return
	}
}
