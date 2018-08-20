package noderpc

import (
	"context"
	"io"
	"net"
	"time"

	"../../repo"

	log "github.com/sirupsen/logrus"

	swarm ".."
	. "../wire"
)

type Server struct {
	listener   net.Listener
	node       *swarm.Node
	chShutdown chan struct{}
}

func NewServer(node *swarm.Node) *Server {
	return &Server{node: node, chShutdown: make(chan struct{})}
}

func (s *Server) Start() {
	listener, err := net.Listen(s.node.Config.Node.RPCListenNetwork, s.node.Config.Node.RPCListenHost)
	if err != nil {
		panic(err)
	}

	s.listener = listener

	for {
		conn, err := listener.Accept()
		if err != nil {
			// this is the awkward way that Go requires us to detect that listener.Close() has been called
			select {
			case <-s.chShutdown:
				return
			default:
			}

			log.Errorf("[rpc] %v", err)
		} else {
			log.Printf("[rpc] opening new stream")
			go s.rpcStreamHandler(conn)
		}

	}
}

func (s *Server) Close() error {
	close(s.chShutdown)
	return s.listener.Close()
}

func (s *Server) rpcStreamHandler(stream io.ReadWriteCloser) {
	defer stream.Close()

	logErr := func(err error) {
		if err != nil {
			log.Errorf("[rpc] %v", err)
		}
	}

	msgType, err := ReadUint64(stream)
	if err != nil {
		panic(err)
	}

	switch MessageType(msgType) {

	case MessageType_SetUsername:
		req := SetUsernameRequest{}
		err := ReadStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)

		tx, err := s.node.EnsureUsername(ctx, req.Username)
		if err != nil {
			WriteStructPacket(stream, &SetUsernameResponse{Error: err.Error()})
			return
		}

		if tx != nil {
			resp := <-tx.Await(ctx)
			if resp.Err != nil {
				WriteStructPacket(stream, &SetUsernameResponse{Error: resp.Err.Error()})
				return
			} else if resp.Receipt.Status != 1 {
				WriteStructPacket(stream, &SetUsernameResponse{Error: "transaction failed"})
				return
			}
		}

		err = WriteStructPacket(stream, &SetUsernameResponse{Error: ""})

	case MessageType_GetObject:
		req := GetObjectRequest{}
		err := ReadStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		objectReader, err := s.node.GetObjectReader(context.Background(), req.RepoID, req.ObjectID)
		// @@TODO: maybe don't assume err == 404
		if err != nil {
			switch err {
			case ErrUnauthorized:
				err = WriteStructPacket(stream, &GetObjectResponse{Unauthorized: true})
			case ErrObjectNotFound:
				err = WriteStructPacket(stream, &GetObjectResponse{HasObject: false})
			default:
				err = WriteStructPacket(stream, &GetObjectResponse{HasObject: false})
			}
			if err != nil {
				panic(err)
			}
			return
		}

		err = WriteStructPacket(stream, &GetObjectResponse{
			Unauthorized: false,
			HasObject:    true,
			ObjectType:   objectReader.Type(),
			ObjectLen:    objectReader.Len(),
		})
		if err != nil {
			panic(err)
		}

		// Write the object
		written, err := io.Copy(stream, objectReader)
		if err != nil {
			panic(err)
		} else if written < objectReader.Len() {
			panic("written < objectLen")
		}

	case MessageType_RegisterRepoID:
		log.Printf("[rpc] RegisterRepoID")
		req := RegisterRepoIDRequest{}
		err := ReadStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc] create repo: %s", req.RepoID)

		tx, err := s.node.EnsureRepoIDRegistered(context.Background(), req.RepoID)
		if err != nil {
			panic(err)
		}

		if tx != nil {
			log.Printf("[rpc] create repo tx sent: %s", tx.Hash().Hex())
			txResult := <-tx.Await(context.Background())
			if txResult.Err != nil {
				panic(err)
			}
			log.Printf("[rpc] create repo tx resolved: %s", tx.Hash().Hex())
		}

		err = WriteStructPacket(stream, &RegisterRepoIDResponse{OK: true})
		if err != nil {
			panic(err)
		}

	case MessageType_AddRepo:
		log.Printf("[rpc] AddRepo")
		req := AddRepoRequest{}
		err := ReadStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc] add repo: %s", req.RepoPath)

		_, err = s.node.RepoManager.TrackRepo(req.RepoPath)
		if err != nil {
			panic(err)
		}

		err = WriteStructPacket(stream, &AddRepoResponse{OK: true})
		if err != nil {
			panic(err)
		}

	case MessageType_GetRepos:
		log.Printf("[rpc] GetRepos")

		repos := make([]Repo, 0)

		err := s.node.RepoManager.ForEachRepo(func(r *repo.Repo) error {
			repoID, err := r.RepoID()
			if err != nil {
				return err
			}
			repos = append(repos, Repo{RepoID: repoID, Path: r.Path})
			return nil
		})
		if err != nil {
			panic(err)
		}

		err = WriteStructPacket(stream, &GetReposResponse{Repos: repos})
		if err != nil {
			panic(err)
		}

	case MessageType_SetReplicationPolicy:
		log.Printf("[rpc] SetReplicationPolicy")
		req := SetReplicationPolicyRequest{}
		err := ReadStructPacket(stream, &req)
		if err != nil {
			logErr(err)
			return
		}

		log.Printf("[rpc] SetReplicationPolicy(%s, %v)", req.RepoID, req.ShouldReplicate)

		err = s.node.SetReplicationPolicy(req.RepoID, req.ShouldReplicate)
		if err != nil {
			err = WriteStructPacket(stream, &SetReplicationPolicyResponse{Error: err.Error()})
			logErr(err)
			return
		}

		err = WriteStructPacket(stream, &SetReplicationPolicyResponse{Error: ""})
		logErr(err)

	case MessageType_AnnounceRepoContent:
		log.Printf("[rpc] AnnounceRepoContent")
		req := AnnounceRepoContentRequest{}
		err := ReadStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc] announce repo content: %s", req.RepoID)

		err = s.node.AnnounceRepoContent(context.Background(), req.RepoID)
		if err != nil {
			panic(err)
		}

		err = WriteStructPacket(stream, &AnnounceRepoContentResponse{OK: true})
		if err != nil {
			panic(err)
		}

	case MessageType_GetRefs:
		log.Printf("[rpc] GetRefs")
		req := GetRefsRequest{}
		err := ReadStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc] get refs from %s", req.RepoID)

		numRefs, err := s.node.GetNumRefs(context.Background(), req.RepoID)
		if err != nil {
			panic(err)
		}

		refMap, err := s.node.GetRefs(context.Background(), req.RepoID, req.Page)
		if err != nil {
			log.Errorf("[rpc] GetRefs: %v", err)
			refMap = map[string]Ref{}
		}

		refs := make([]Ref, len(refMap))
		i := 0
		for _, ref := range refMap {
			refs[i] = ref
			i++
		}

		err = WriteStructPacket(stream, &GetRefsResponse{Refs: refs, NumRefs: numRefs})
		if err != nil {
			panic(err)
		}

	case MessageType_UpdateRef:
		log.Printf("[rpc] UpdateRef")
		req := UpdateRefRequest{}
		err := ReadStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc] add ref to %s: %s %s", req.RepoID, req.RefName, req.Commit)
		tx, err := s.node.UpdateRef(context.Background(), req.RepoID, req.RefName, req.Commit)
		if err != nil {
			panic(err)
		}
		log.Printf("[rpc] update ref tx sent: %s", tx.Hash().Hex())
		txResult := <-tx.Await(context.Background())
		if txResult.Err != nil {
			panic(err)
		}
		log.Printf("[rpc] update ref tx resolved: %s", tx.Hash().Hex())

		err = WriteStructPacket(stream, &UpdateRefResponse{OK: true})
		if err != nil {
			panic(err)
		}

	case MessageType_Replicate:
		req := ReplicationRequest{}
		err := ReadStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		err = s.node.RequestReplication(context.Background(), req.RepoID)
		errStr := ""
		if err != nil {
			log.Errorf("[rpc] MessageType_Replicate error: %v", err)
			errStr = err.Error()
		}

		err = WriteStructPacket(stream, &ReplicationResponse{Error: errStr})
		if err != nil {
			panic(err)
		}

	default:
		log.Errorf("Node.rpcStreamHandler: bad message type from peer (%v)", msgType)
	}
}
