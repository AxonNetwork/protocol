package p2pserver

import (
	"context"

	netp2p "github.com/libp2p/go-libp2p-net"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodep2p"
	. "github.com/Conscience/protocol/swarm/wire"
)

func (s *Server) HandleBecomeReplicatorRequest(stream netp2p.Stream) {
	log.Printf("[become replicator] receiving 'become replicator' request")
	defer stream.Close()

	req := BecomeReplicatorRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[become replicator] error: %v", err)
		return
	}
	log.Debugf("[become replicator] repoID: %v", req.RepoID)

	cfg := s.node.GetConfig()
	if cfg.Node.ReplicateEverything {
		err = s.node.SetReplicationPolicy(req.RepoID, true)
		if err != nil {
			log.Errorf("[become replicator] error: %v", err)
			_ = WriteStructPacket(stream, &BecomeReplicatorResponse{Error: err.Error()})
			return
		}

		// Acknowledge that we will now replicate the repo
		err = WriteStructPacket(stream, &BecomeReplicatorResponse{Error: ""})
		if err != nil {
			log.Errorf("[become replicator] error: %v", err)
			return
		}

	} else {
		err = WriteStructPacket(stream, &BecomeReplicatorResponse{Error: "no"})
		if err != nil {
			log.Errorf("[become replicator] error: %v", err)
			return
		}
	}
}

// Handles an incoming request to replicate (pull changes from) a given repository.
func (s *Server) HandleReplicationRequest(stream netp2p.Stream) {
	log.Printf("[replication] receiving replication request")
	defer stream.Close()

	req := ReplicationRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[replication server] error: %v", err)
		return
	}
	log.Debugf("[replication server] repoID: %v", req.RepoID)

	// Ensure that the repo has been whitelisted for replication.
	{
		whitelisted := false
		cfg := s.node.GetConfig()
		for _, repo := range cfg.Node.ReplicateRepos {
			if repo == req.RepoID {
				whitelisted = true
				break
			}
		}

		if !whitelisted {
			err = WriteStructPacket(stream, &Progress{ErrorMsg: "not a whitelisted repo"})
			if err != nil {
				log.Errorf("[replication server] error: %v", err)
			}
			return
		}
	}

	// Perform the fetch
	{
		r := s.node.Repo(req.RepoID)
		if r == nil {
			log.Errorf("[replication server] don't have this repo locally")
			return
		}

		// @@TODO: give context a timeout and make it configurable
		_, err := nodep2p.FetchConscienceRemote(context.TODO(), &nodep2p.FetchOptions{
			Repo: r,
			ProgressCb: func(current, total uint64) error {
				err := WriteStructPacket(stream, &Progress{Current: current, Total: total})
				if err != nil {
					log.Errorf("[replication server] error: %v", err)
					return err
				}
				return nil
			},
		})
		if err != nil {
			log.Errorf("[replication server] error fetching conscience:// remote for repo %v: %v", req.RepoID, err)
			err = WriteStructPacket(stream, &Progress{ErrorMsg: err.Error()})
			if err != nil {
				log.Errorf("[replication server] error: %v", err)
				return
			}
			return
		}
	}

	err = WriteStructPacket(stream, &Progress{Done: true})
	if err != nil {
		log.Errorf("[replication server] error: %v", err)
		return
	}
}
