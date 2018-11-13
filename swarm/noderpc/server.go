package noderpc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"gopkg.in/src-d/go-git.v4"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
	gitobject "gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/Conscience/protocol/log"
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
	signature, err := s.node.SignHash([]byte(req.Username))
	if err != nil {
		return nil, err
	}
	return &pb.SetUsernameResponse{Signature: signature}, nil
}

func (s *Server) GetUsername(ctx context.Context, req *pb.GetUsernameRequest) (*pb.GetUsernameResponse, error) {
	un, err := s.node.GetUsername(ctx)
	if err != nil {
		return nil, err
	}
	signature, err := s.node.SignHash([]byte(un))
	if err != nil {
		return nil, err
	}

	return &pb.GetUsernameResponse{Username: un, Signature: signature}, nil
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

	err = s.node.RequestBecomeReplicator(ctx, req.RepoID)
	if err != nil {
		return nil, errors.WithStack(err)
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
	err = r.AddUserToConfig(req.Name, req.Email)
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

func (s *Server) CheckpointRepo(ctx context.Context, req *pb.CheckpointRepoRequest) (*pb.CheckpointRepoResponse, error) {
	log.Debugln("[checkpoint] req.Path =", req.Path)
	log.Debugln("[checkpoint] git add .")
	err := util.ExecAndScanStdout(ctx, []string{"git", "add", "."}, req.Path, func(line string) error {
		log.Debugln("[checkpoint]  -", line)
		return nil
	})
	if err != nil {
		log.Errorln("[checkpoint]  - error:", err)
		return nil, errors.WithStack(err)
	}

	log.Debugln("[checkpoint] git commit -m " + req.Message)
	err = util.ExecAndScanStdout(ctx, []string{"git", "commit", "-m", req.Message}, req.Path, func(line string) error {
		log.Debugln("[checkpoint]  -", line)
		return nil
	})
	if err != nil {
		log.Errorln("[checkpoint]  - error:", err)
		return nil, errors.WithStack(err)
	}

	log.Debugln("[checkpoint] git push origin master")
	err = util.ExecAndScanStdout(ctx, []string{"git", "push", "origin", "master"}, req.Path, func(line string) error {
		log.Debugln("[checkpoint]  -", line)
		return nil
	})
	if err != nil {
		log.Errorln("[checkpoint]  - error:", err)
		return nil, errors.WithStack(err)
	}

	return &pb.CheckpointRepoResponse{Ok: true}, nil
}

func (s *Server) PullRepo(ctx context.Context, req *pb.PullRepoRequest) (*pb.PullRepoResponse, error) {
	didStash := false
	err := util.ExecAndScanStdout(ctx, []string{"git", "stash"}, req.Path, func(line string) error {
		if line != "No local changes to save" {
			didStash = true
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = util.ExecAndScanStdout(ctx, []string{"git", "pull", "origin", "master"}, req.Path, func(line string) error {
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if didStash {
		// @@TODO: handle merge conflict on stash pop
		err = util.ExecAndScanStdout(ctx, []string{"git", "stash", "apply"}, req.Path, func(line string) error {
			return nil
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// @@TODO: git stash drop
	}
	return &pb.PullRepoResponse{Ok: true}, nil
}

func (s *Server) CloneRepo(ctx context.Context, req *pb.CloneRepoRequest) (*pb.CloneRepoResponse, error) {
	location := req.Path
	if len(location) == 0 {
		location = s.node.Config.Node.ReplicationRoot
	}
	remote := fmt.Sprintf("conscience://%s", req.RepoID)
	err := util.ExecAndScanStdout(ctx, []string{"git", "clone", remote}, location, func(line string) error {
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	name := req.RepoID
	if strings.Contains(name, "/") {
		name = strings.Split(name, "/")[1]
	}

	repoPath := filepath.Join(location, name)
	r, err := repo.Open(repoPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = r.AddUserToConfig(req.Name, req.Email)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &pb.CloneRepoResponse{Path: repoPath}, nil
}

func (s *Server) FetchFromCommit(ctx context.Context, req *pb.FetchFromCommitRequest) (*pb.FetchFromCommitResponse, error) {
	err := s.node.FetchFromCommit(ctx, req.RepoID, req.Path, req.Commit)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &pb.FetchFromCommitResponse{Ok: true}, nil

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
	refs, repoPath, err := s.node.GetLocalRefs(ctx, req.RepoID, req.Path)
	if err != nil {
		return nil, err
	}

	refsList := make([]*pb.Ref, len(refs))
	i := 0
	for _, ref := range refs {
		refsList[i] = &pb.Ref{RefName: ref.RefName, CommitHash: ref.CommitHash}
		i++
	}
	return &pb.GetLocalRefsResponse{Path: repoPath, Refs: refsList}, nil
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

func (s *Server) IsBehindRemote(ctx context.Context, req *pb.IsBehindRemoteRequest) (*pb.IsBehindRemoteResponse, error) {
	isBehindRemote, err := s.node.IsBehindRemote(ctx, req.RepoID, req.Path)
	if err != nil {
		return nil, err
	}

	return &pb.IsBehindRemoteResponse{RepoID: req.RepoID, IsBehindRemote: isBehindRemote}, nil
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

	logs, err := s.node.GetRefLogs(ctx, req.RepoID)
	if err != nil {
		return nil, err
	}

	commits := []*pb.Commit{}
	err = cIter.ForEach(func(commit *gitobject.Commit) error {
		if commit == nil {
			log.Warnf("[node] nil commit (repoID: %v)", req.RepoID)
			return nil
		}
		commitHash := commit.Hash.String()
		files, err := getFilesForCommit(ctx, r.Path, commitHash)
		if err != nil {
			return err
		}
		verified := logs[commitHash]
		commits = append(commits, &pb.Commit{
			CommitHash: commitHash,
			Author:     commit.Author.String(),
			Message:    commit.Message,
			Timestamp:  uint64(commit.Author.When.Unix()),
			Files:      files,
			Verified:   verified,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &pb.GetRepoHistoryResponse{Commits: commits}, nil
}

func getFilesForCommit(ctx context.Context, path string, commitHash string) ([]string, error) {
	// Start by taking the output of `git ls-files --stage`
	files := make([]string, 0)
	err := util.ExecAndScanStdout(ctx, []string{"git", "show", "--name-only", "--pretty=format:\"\"", commitHash}, path, func(line string) error {
		if len(line) > 0 {
			files = append(files, line)
		}
		return nil
	})
	return files, err
}

func parseGitStatusLine(line string) (*pb.File, error) {
	parts := strings.Split(line, " ")
	file := &pb.File{}

	switch parts[0] {
	case "u":
		mode, err := strconv.ParseUint(parts[3], 8, 32)
		if err != nil {
			return nil, err
		}

		hash, err := hex.DecodeString(parts[7])
		if err != nil {
			return nil, err
		}

		file.Name = parts[10]
		file.Hash = hash
		file.Mode = uint32(mode)
		file.UnstagedStatus = parts[1][:1]
		file.StagedStatus = parts[1][1:]

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
	moarParts := strings.Split(line, "\t")
	parts := strings.Split(moarParts[0], " ")

	mode, err := strconv.ParseUint(parts[0], 8, 32)
	if err != nil {
		return nil, err
	}

	hash, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	name := moarParts[1]
	if name[0:1] == "\"" {
		name = fmt.Sprintf(name[1 : len(name)-2])
	}

	return &pb.File{
		Name:           name,
		Hash:           hash,
		Mode:           uint32(mode),
		Size:           0,
		UnstagedStatus: ".",
		StagedStatus:   ".",
	}, nil
}

func getStats(path string) (os.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	stat, err := f.Stat()
	return stat, err
}

func getMergeConflicts(ctx context.Context, path string) ([]string, []string, error) {
	unresolved := make([]string, 0)
	err := util.ExecAndScanStdout(ctx, []string{"git", "diff", "--name-only", "--diff-filter=U"}, path, func(line string) error {
		unresolved = append(unresolved, line)
		return nil
	})
	if err != nil {
		return []string{}, []string{}, err
	}

	mergeConflicts := make([]string, 0)
	for i := range unresolved {
		exists, err := util.GrepExists(filepath.Join(path, unresolved[i]), "<<<<<")
		if err != nil {
			return []string{}, []string{}, err
		}
		if exists {
			mergeConflicts = append(mergeConflicts, unresolved[i])
		}

	}
	return unresolved, mergeConflicts, err
}

func contains(arr []string, str string) bool {
	for i := range arr {
		if arr[i] == str {
			return true
		}
	}
	return false
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
			log.Errorln("GetRepoFiles (git ls-files):", err)
			return nil // continue
		}
		files[file.Name] = file
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Then, overlay the output of `git status --porcelain`
	err = util.ExecAndScanStdout(ctx, []string{"git", "status", "--porcelain=2"}, r.Path, func(line string) error {
		file, err := parseGitStatusLine(line)
		if err != nil {
			log.Errorln("GetRepoFiles (git status --porcelain=2):", err)
			return nil // continue
		}
		files[file.Name] = file
		return nil
	})
	if err != nil {
		return nil, err
	}

	unresolved, mergeConflicts, err := getMergeConflicts(ctx, r.Path)
	if err != nil {
		return nil, err
	}

	fileList := []*pb.File{}
	for _, file := range files {
		stat, err := getStats(filepath.Join(r.Path, file.Name))
		if err != nil {
			log.Errorln("GetRepoFiles (getStats):", err)
			continue
		}
		file.Modified = uint32(stat.ModTime().Unix())
		file.Size = uint64(stat.Size())
		file.MergeConflict = contains(mergeConflicts, file.Name)
		file.MergeUnresolved = contains(unresolved, file.Name)
		fileList = append(fileList, file)
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

func (s *Server) SignMessage(ctx context.Context, req *pb.SignMessageRequest) (*pb.SignMessageResponse, error) {
	signature, err := s.node.SignHash(req.Message)
	if err != nil {
		return nil, err
	}
	return &pb.SignMessageResponse{Signature: signature}, nil
}

func (s *Server) EthAddress(ctx context.Context, req *pb.EthAddressRequest) (*pb.EthAddressResponse, error) {
	addr := s.node.EthAddress()
	return &pb.EthAddressResponse{Address: addr.String()}, nil
}

func (s *Server) SetUserPermissions(ctx context.Context, req *pb.SetUserPermissionsRequest) (*pb.SetUserPermissionsResponse, error) {
	tx, err := s.node.SetUserPermissions(ctx, req.RepoID, req.Username, nodeeth.UserPermissions{Puller: req.Puller, Pusher: req.Pusher, Admin: req.Admin})
	if err != nil {
		return nil, err
	}

	txResult := <-tx.Await(ctx)
	if txResult.Err != nil {
		return nil, txResult.Err
	} else if txResult.Receipt.Status == 0 {
		return nil, errors.New("transaction failed")
	}
	return &pb.SetUserPermissionsResponse{}, nil
}
