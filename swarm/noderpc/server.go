package noderpc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"gopkg.in/src-d/go-git.v4"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm"
	"github.com/Conscience/protocol/swarm/nodeeth"
	"github.com/Conscience/protocol/swarm/noderpc/pb"
	"github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
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

	var opts []grpc.ServerOption = []grpc.ServerOption{
		grpc.StreamInterceptor(StreamServerInterceptor()),
		grpc.UnaryInterceptor(UnaryServerInterceptor()),
	}
	s.server = grpc.NewServer(opts...)
	pb.RegisterNodeRPCServer(s.server, s)
	s.server.Serve(lis)
}

func (s *Server) Close() error {
	// This closes the net.Listener as well.
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

func (s *Server) InitRepo(ctx context.Context, req *pb.InitRepoRequest) (*pb.InitRepoResponse, error) {
	if req.RepoID == "" {
		return nil, errors.New("empty repoID")
	}

	// Before anything else, try to claim the repoID in the smart contract
	tx, err := s.node.EnsureRepoIDRegistered(ctx, req.RepoID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if tx != nil {
		log.Printf("[rpc] create repo tx sent: %s", tx.Hash().Hex())
		txResult := <-tx.Await(ctx)
		if txResult.Err != nil {
			return nil, errors.WithStack(txResult.Err)
		}
		log.Printf("[rpc] create repo tx resolved: %s", tx.Hash().Hex())
	}

	// If no path was specified, create the repo in the ReplicationRoot
	path := req.Path
	if path == "" {
		path = filepath.Join(s.node.Config.Node.ReplicationRoot, req.RepoID)
	}

	// Open or create the git repo
	r, err := repo.Open(path)
	if err != nil {
		r, err = repo.Init(path)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// Setup the Conscience plugins, etc.
	err = r.SetupConfig(req.RepoID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Have the node track the local repo
	_, err = s.node.RepoManager.TrackRepo(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &pb.InitRepoResponse{Path: path}, nil
}

func (s *Server) GetObject(req *pb.GetObjectRequest, server pb.NodeRPC_GetObjectServer) error {
	objectReader, err := s.node.GetObjectReader(server.Context(), req.RepoID, req.ObjectID)
	if err != nil {
		return errors.WithStack(err)
	}
	defer objectReader.Close()

	// First, send a special header packet containing the type and length of the object
	{
		headerbuf := &bytes.Buffer{}
		err = wire.WriteStructPacket(headerbuf, &wire.ObjectMetadata{Type: objectReader.Type(), Len: objectReader.Len()})
		if err != nil {
			return errors.WithStack(err)
		}

		err = server.Send(&pb.GetObjectResponsePacket{Data: headerbuf.Bytes()})
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// @@TODO: make this configurable
	const CHUNK_SIZE = 1048576 // 1 MiB
	data := &bytes.Buffer{}

	eof := false
	for !eof {
		_, err = io.CopyN(data, objectReader, CHUNK_SIZE)
		if err == io.EOF {
			eof = true
		} else if err != nil {
			return errors.WithStack(err)
		}

		err = server.Send(&pb.GetObjectResponsePacket{Data: data.Bytes()})
		if err != nil {
			return errors.WithStack(err)
		}

		data.Reset()
	}
	return nil
}

func (s *Server) RegisterRepoID(ctx context.Context, req *pb.RegisterRepoIDRequest) (*pb.RegisterRepoIDResponse, error) {
	tx, err := s.node.EnsureRepoIDRegistered(ctx, req.RepoID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if tx != nil {
		log.Printf("[rpc] create repo tx sent: %s", tx.Hash().Hex())
		txResult := <-tx.Await(ctx)
		if txResult.Err != nil {
			return nil, errors.WithStack(txResult.Err)
		}
		log.Printf("[rpc] create repo tx resolved: %s", tx.Hash().Hex())
	}
	return &pb.RegisterRepoIDResponse{}, nil
}

func (s *Server) TrackLocalRepo(ctx context.Context, req *pb.TrackLocalRepoRequest) (*pb.TrackLocalRepoResponse, error) {
	_, err := s.node.RepoManager.TrackRepo(req.RepoPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &pb.TrackLocalRepoResponse{}, nil
}

func (s *Server) GetLocalRepos(req *pb.GetLocalReposRequest, server pb.NodeRPC_GetLocalReposServer) error {
	return s.node.RepoManager.ForEachRepo(func(r *repo.Repo) error {
		select {
		case <-server.Context().Done():
			return errors.WithStack(server.Context().Err())
		default:
		}

		repoID, err := r.RepoID()
		if err != nil {
			return errors.WithStack(err)
		}
		err = server.Send(&pb.GetLocalReposResponsePacket{RepoID: repoID, Path: r.Path})
		return errors.WithStack(err)
	})
}

func (s *Server) SetReplicationPolicy(ctx context.Context, req *pb.SetReplicationPolicyRequest) (*pb.SetReplicationPolicyResponse, error) {
	err := s.node.SetReplicationPolicy(req.RepoID, req.ShouldReplicate)
	if err != nil {
		return nil, errors.WithStack(err)
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

func (s *Server) GetLocalRefs(ctx context.Context, req *pb.GetLocalRefsRequest) (*pb.GetLocalRefsResponse, error) {
	// @@TODO: stub
	return nil, nil
}

func (s *Server) GetRemoteRefs(ctx context.Context, req *pb.GetRemoteRefsRequest) (*pb.GetRemoteRefsResponse, error) {
	refMap, total, err := s.node.GetRemoteRefs(ctx, req.RepoID, req.PageSize, req.Page)
	if err != nil {
		return nil, err
	}

	refs := []*pb.Ref{}
	for _, ref := range refMap {
		refs = append(refs, &pb.Ref{RefName: ref.RefName, CommitHash: ref.CommitHash})
	}

	return &pb.GetRemoteRefsResponse{Total: total, Refs: refs}, nil
}

func (s *Server) UpdateRef(ctx context.Context, req *pb.UpdateRefRequest) (*pb.UpdateRefResponse, error) {
	tx, err := s.node.UpdateRef(ctx, req.RepoID, req.RefName, req.CommitHash)
	if err != nil {
		return nil, err
	}

	txResult := <-tx.Await(ctx)
	if txResult.Err != nil {
		return nil, txResult.Err
	} else if txResult.Receipt.Status == 0 {
		return nil, errors.New("transaction failed")
	}

	return &pb.UpdateRefResponse{}, nil
}

func (s *Server) GetRepoUsers(ctx context.Context, req *pb.GetRepoUsersRequest) (*pb.GetRepoUsersResponse, error) {
	users, total, err := s.node.GetRepoUsers(ctx, req.RepoID, nodeeth.UserType(req.Type), req.PageSize, req.Page)
	if err != nil {
		return nil, err
	}
	return &pb.GetRepoUsersResponse{Total: total, Users: users}, nil
}

func (s *Server) RequestReplication(ctx context.Context, req *pb.ReplicationRequest) (*pb.ReplicationResponse, error) {
	err := s.node.RequestReplication(ctx, req.RepoID)
	if err != nil {
		return nil, err
	}
	return &pb.ReplicationResponse{}, nil
}

func (s *Server) GetRepoHistory(ctx context.Context, req *pb.GetRepoHistoryRequest) (*pb.GetRepoHistoryResponse, error) {
	var r *repo.Repo

	if len(req.Path) > 0 {
		r = s.node.RepoManager.RepoAtPath(req.Path)
		if r == nil {
			return nil, errors.Errorf("repo '%v'  not found", req.RepoID)
		}

	} else if len(req.RepoID) > 0 {
		r = s.node.RepoManager.Repo(req.RepoID)
		if r == nil {
			return nil, errors.Errorf("repo '%v'  not found", req.RepoID)
		}

	} else {
		return nil, errors.Errorf("must provide either repoID or path")
	}

	cIter, err := r.Log(&git.LogOptions{From: gitplumbing.ZeroHash, Order: git.LogOrderDFS})
	if err != nil {
		return nil, err
	}

	commits := []*pb.Commit{}
	err = cIter.ForEach(func(commit *gitobject.Commit) error {
		if commit == nil {
			log.Warnf("[node] nil commit (repoID: %v)", req.RepoID)
			return nil
		}
		commits = append(commits, &pb.Commit{
			CommitHash: commit.Hash.String(),
			Author:     commit.Author.String(),
			Message:    commit.Message,
			Timestamp:  uint64(commit.Author.When.Unix()),
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &pb.GetRepoHistoryResponse{Commits: commits}, nil
}

func parseGitStatusLine(line string) (*pb.File, error) {
	parts := strings.Split(line, " ")
	file := &pb.File{}

	switch parts[0] {
	case "1":
		mode, err := strconv.ParseUint(parts[3], 8, 32)
		if err != nil {
			return nil, err
		}

		hash, err := hex.DecodeString(parts[6])
		if err != nil {
			return nil, err
		}

		file.Name = parts[8]
		file.Hash = hash
		file.Mode = uint32(mode)
		file.UnstagedStatus = parts[1][:1]
		file.StagedStatus = parts[1][1:]

	case "2":
		// @@TODO: these are renames

	case "?":
		file.Name = parts[1]
		file.UnstagedStatus = "?"
		file.StagedStatus = "?"
	}

	return file, nil
}

func parseGitLSFilesLine(line string) (*pb.File, error) {
	parts := strings.Split(line, " ")
	moarParts := strings.Split(parts[2], "\t")

	mode, err := strconv.ParseUint(moarParts[0], 8, 32)
	if err != nil {
		return nil, err
	}

	hash, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	return &pb.File{
		Name:           moarParts[1],
		Hash:           hash,
		Mode:           uint32(mode),
		Size:           0,
		UnstagedStatus: ".",
		StagedStatus:   ".",
	}, nil
}

// @@TODO: move this into the Node
func (s *Server) GetRepoFiles(ctx context.Context, req *pb.GetRepoFilesRequest) (*pb.GetRepoFilesResponse, error) {
	var r *repo.Repo

	if len(req.Path) > 0 {
		r = s.node.RepoManager.RepoAtPath(req.Path)
		if r == nil {
			return nil, errors.Errorf("repo at path '%v' not found", req.Path)
		}

	} else if len(req.RepoID) > 0 {
		r = s.node.RepoManager.Repo(req.RepoID)
		if r == nil {
			return nil, errors.Errorf("repo '%v' not found", req.RepoID)
		}

	} else {
		return nil, errors.Errorf("must provide either 'path' or 'repoID'")
	}

	files := map[string]*pb.File{}

	// Start by taking the output of `git ls-files --stage`
	err := util.ExecAndScanStdout(ctx, []string{"git", "ls-files", "--stage"}, r.Path, func(line string) error {
		file, err := parseGitLSFilesLine(line)
		if err != nil {
			return err
		}
		files[file.Name] = file
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Then, overlay the output of `git status --porcelain`
	err = util.ExecAndScanStdout(ctx, []string{"git", "status", "--porcelain"}, r.Path, func(line string) error {
		file, err := parseGitStatusLine(line)
		if err != nil {
			return err
		}
		files[file.Name] = file
		return nil
	})
	if err != nil {
		return nil, err
	}

	fileList := make([]*pb.File, len(files))
	i := 0
	for _, file := range files {
		fileList[i] = file
		i++
	}

	return &pb.GetRepoFilesResponse{Files: fileList}, nil
}

func (s *Server) RepoHasObject(ctx context.Context, req *pb.RepoHasObjectRequest) (*pb.RepoHasObjectResponse, error) {
	var r *repo.Repo

	if len(req.Path) > 0 {
		r = s.node.RepoManager.RepoAtPath(req.Path)
		if r == nil {
			return nil, errors.Errorf("repo at path '%v' not found", req.Path)
		}

	} else if len(req.RepoID) > 0 {
		r = s.node.RepoManager.Repo(req.RepoID)
		if r == nil {
			return nil, errors.Errorf("repo '%v' not found", req.RepoID)
		}

	} else {
		return nil, errors.Errorf("must provide either 'path' or 'repoID'")
	}

	return &pb.RepoHasObjectResponse{
		HasObject: r.HasObject(req.ObjectID),
	}, nil
}
