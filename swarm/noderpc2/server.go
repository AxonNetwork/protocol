package noderpc

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	swarm ".."
	"../../repo"
	"../wire"
	"./pb"
)

type Server struct {
	node   *swarm.Node
	server *grpc.Server
}

func NewServer(node *swarm.Node) *Server {
	return &Server{node: node}
}

func (s *Server) Start() {
	lis, err := net.Listen(s.node.Config.Node.RPCListenNetwork, s.node.Config.Node.RPCListenHost)
	if err != nil {
		panic(fmt.Sprintf("failed to listen: %v\n", err))
	}

	var opts []grpc.ServerOption
	s.server = grpc.NewServer(opts...)
	pb.RegisterNodeRPCServer(s.server, s)
	s.server.Serve(lis)
}

func (s *Server) Close() error {
	s.server.GracefulStop()
	return nil
}

func (s *Server) SetUsername(ctx context.Context, req *pb.SetUsernameRequest) (*pb.SetUsernameResponse, error) {
	tx, err := s.node.EnsureUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	if tx != nil {
		resp := <-tx.Await(ctx)
		if resp.Err != nil {
			return nil, resp.Err
		} else if resp.Receipt.Status != 1 {
			return nil, errors.New("transaction failed")
		}
	}
	return &pb.SetUsernameResponse{}, nil
}

func (s *Server) GetObject(req *pb.GetObjectRequest, server pb.NodeRPC_GetObjectServer) error {
	objectReader, err := s.node.GetObjectReader(server.Context(), req.RepoID, req.ObjectID)
	if err != nil {
		return err
	}
	defer objectReader.Close()

	// First, send a special header packet containing the type and length of the object
	{
		headerbuf := &bytes.Buffer{}
		err = wire.WriteStructPacket(headerbuf, wire.ObjectMetadata{Type: objectReader.Type(), Len: objectReader.Len()})
		if err != nil {
			return err
		}
		err = server.Send(&pb.GetObjectResponsePacket{Data: headerbuf.Bytes()})
		if err != nil {
			return err
		}
	}

	const CHUNK_SIZE = 2 ^ 20 // 1 MiB
	data := bytes.NewBuffer(make([]byte, CHUNK_SIZE))

	for {
		_, err = io.CopyN(data, objectReader, CHUNK_SIZE)
		if err != nil {
			return err
		}

		err = server.Send(&pb.GetObjectResponsePacket{Data: data.Bytes()})
		if err != nil {
			return err
		}

		data.Reset()
	}
	return nil
}

func (s *Server) RegisterRepoID(ctx context.Context, req *pb.RegisterRepoIDRequest) (*pb.RegisterRepoIDResponse, error) {
	tx, err := s.node.EnsureRepoIDRegistered(ctx, req.RepoID)
	if err != nil {
		return nil, err
	}

	if tx != nil {
		log.Printf("[rpc] create repo tx sent: %s", tx.Hash().Hex())
		txResult := <-tx.Await(ctx)
		if txResult.Err != nil {
			return nil, txResult.Err
		}
		log.Printf("[rpc] create repo tx resolved: %s", tx.Hash().Hex())
	}
	return &pb.RegisterRepoIDResponse{}, nil
}

func (s *Server) TrackLocalRepo(ctx context.Context, req *pb.TrackLocalRepoRequest) (*pb.TrackLocalRepoResponse, error) {
	_, err := s.node.RepoManager.TrackRepo(req.RepoPath)
	if err != nil {
		return nil, err
	}
	return &pb.TrackLocalRepoResponse{}, nil
}

func (s *Server) GetLocalRepos(req *pb.GetLocalReposRequest, server pb.NodeRPC_GetLocalReposServer) error {
	return s.node.RepoManager.ForEachRepo(func(r *repo.Repo) error {
		select {
		case <-server.Context().Done():
			return errors.New("context timed out")
		default:
		}

		repoID, err := r.RepoID()
		if err != nil {
			return err
		}
		return server.Send(&pb.GetLocalReposResponsePacket{RepoID: repoID, Path: r.Path})
	})
}

func (s *Server) SetReplicationPolicy(ctx context.Context, req *pb.SetReplicationPolicyRequest) (*pb.SetReplicationPolicyResponse, error) {
	err := s.node.SetReplicationPolicy(req.RepoID, req.ShouldReplicate)
	if err != nil {
		return nil, err
	}
	return &pb.SetReplicationPolicyResponse{}, nil
}

func (s *Server) AnnounceRepoContent(ctx context.Context, req *pb.AnnounceRepoContentRequest) (*pb.AnnounceRepoContentResponse, error) {
	err := s.node.AnnounceRepoContent(ctx, req.RepoID)
	if err != nil {
		return nil, err
	}
	return &pb.AnnounceRepoContentResponse{}, nil
}

func (s *Server) GetRefs(ctx context.Context, req *pb.GetRefsRequest) (*pb.GetRefsResponse, error) {
	numRefs, err := s.node.GetNumRefs(ctx, req.RepoID)
	if err != nil {
		return nil, err
	}

	refMap, err := s.node.GetRefs(ctx, req.RepoID, int64(req.Page))
	if err != nil {
		return nil, err
	}

	refs := []*pb.Ref{}
	for _, ref := range refMap {
		refs = append(refs, &pb.Ref{RefName: ref.RefName, CommitHash: ref.CommitHash})
	}

	return &pb.GetRefsResponse{NumRefs: numRefs, Ref: refs}, nil
}

func (s *Server) UpdateRef(ctx context.Context, req *pb.UpdateRefRequest) (*pb.UpdateRefResponse, error) {
	tx, err := s.node.UpdateRef(ctx, req.RepoID, req.RefName, req.CommitHash)
	if err != nil {
		return nil, err
	}
	log.Printf("[rpc] update ref tx sent: %s", tx.Hash().Hex())
	txResult := <-tx.Await(ctx)
	if txResult.Err != nil {
		return nil, txResult.Err
	}
	log.Printf("[rpc] update ref tx resolved: %s", tx.Hash().Hex())
	return &pb.UpdateRefResponse{}, nil
}

func (s *Server) RequestReplication(ctx context.Context, req *pb.ReplicationRequest) (*pb.ReplicationResponse, error) {
	err := s.node.RequestReplication(context.Background(), req.RepoID)
	if err != nil {
		return nil, err
	}
	return &pb.ReplicationResponse{}, nil
}
