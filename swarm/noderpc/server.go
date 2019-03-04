package noderpc

import (
	"fmt"
	"io"
	"net"
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/libgit2/git2go"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm"
	"github.com/Conscience/protocol/swarm/nodeeth"
	"github.com/Conscience/protocol/swarm/nodep2p"
	"github.com/Conscience/protocol/swarm/noderpc/pb"
	"github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

type Server struct {
	node   *swarm.Node
	server *grpc.Server
}

// @@TODO: make configurable
const OBJ_CHUNK_SIZE = 512 * 1024 // 512kb

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

	// Ping other nodes in the swarm to ask them to replicate this repo
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
	if errors.Cause(err) == repo.Err404 {
		r, err = repo.Init(&repo.InitOptions{
			RepoID:    req.RepoID,
			RepoRoot:  path,
			UserName:  req.Name,
			UserEmail: req.Email,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
	} else if err != nil {
		return nil, errors.WithStack(err)
	}
	defer r.Free()

	// Have the node track the local repo
	_, err = s.node.RepoManager().TrackRepo(path, true)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// if local HEAD exists, push to contract
	_, err = r.Head()
	if err == nil {
		err = nodep2p.Push(ctx, &nodep2p.PushOptions{
			Node:       s.node,
			Repo:       r,
			BranchName: "master", // @@TODO: don't hard code this
			ProgressCb: func(percent int) {
				// @@TODO: stream progress over RPC
			},
		})
		if err != nil {
			return nil, err
		}
	}

	return &pb.InitRepoResponse{Path: path}, nil
}

func (s *Server) CheckpointRepo(ctx context.Context, req *pb.CheckpointRepoRequest) (*pb.CheckpointRepoResponse, error) {
	r := s.node.RepoManager().RepoAtPath(req.Path)
	if r == nil {
		return nil, errors.WithStack(repo.Err404)
	}

	_, err := r.CommitCurrentWorkdir(&repo.CommitOptions{
		Pathspecs: []string{"."},
		Message:   req.Message,
	})
	if err != nil {
		log.Errorln("[checkpoint]  - error:", err)
		return nil, errors.WithStack(err)
	}

	err = nodep2p.Push(ctx, &nodep2p.PushOptions{
		Node:       s.node,
		Repo:       r,
		BranchName: "master", // @@TODO: don't hard code this
		ProgressCb: func(percent int) {
			// @@TODO: stream progress over RPC
		},
	})
	if err != nil {
		log.Errorln("[checkpoint]  - error:", err)
		return nil, errors.WithStack(err)
	}

	return &pb.CheckpointRepoResponse{Ok: true}, nil
}

func (s *Server) PullRepo(req *pb.PullRepoRequest, server pb.NodeRPC_PullRepoServer) error {
	r := s.node.RepoManager().RepoAtPath(req.Path)
	if r == nil {
		return errors.Errorf("repo at path '%v' not found", req.Path)
	}

	// @@TODO: don't hardcode origin/master
	err := nodep2p.Pull(context.TODO(), &nodep2p.PullOptions{
		RemoteName: "origin",
		BranchName: "master",
		ProgressCb: func(done, total uint64) error {

			return server.Send(&pb.PullRepoResponsePacket{
				ToFetch: int64(done),
				Fetched: int64(total),
			})

		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) CloneRepo(req *pb.CloneRepoRequest, server pb.NodeRPC_CloneRepoServer) error {
	repoRoot := req.Path
	if len(repoRoot) == 0 {
		repoRoot = s.node.Config.Node.ReplicationRoot
	}

	r, err := nodep2p.Clone(context.TODO(), &nodep2p.CloneOptions{
		RepoID:    req.RepoID,
		RepoRoot:  repoRoot,
		Bare:      false, // @@TODO
		UserName:  req.Name,
		UserEmail: req.Email,
		ProgressCb: func(done, total uint64) error {

			return server.Send(&pb.CloneRepoResponsePacket{
				Payload: &pb.CloneRepoResponsePacket_Progress_{&pb.CloneRepoResponsePacket_Progress{
					Fetched: int64(done),
					ToFetch: int64(total),
				}},
			})

		},
	})
	if err != nil {
		return errors.WithStack(err)
	}

	err = server.Send(&pb.CloneRepoResponsePacket{
		Payload: &pb.CloneRepoResponsePacket_Success_{&pb.CloneRepoResponsePacket_Success{
			Path: r.Path(),
		}},
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s *Server) FetchFromCommit(req *pb.FetchFromCommitRequest, server pb.NodeRPC_FetchFromCommitServer) error {
	// @@TODO: give context a timeout and make it configurable
	ch, uncompressedSize, totalChunks := s.node.FetchFromCommit(context.TODO(), req.RepoID, req.Path, *util.OidFromBytes(req.Commit), wire.CheckoutType(req.CheckoutType))

	err := server.Send(&pb.FetchFromCommitResponse{
		Payload: &pb.FetchFromCommitResponse_Header_{&pb.FetchFromCommitResponse_Header{
			UncompressedSize: uncompressedSize,
			TotalChunks:      totalChunks,
		}},
	})
	if err != nil {
		return errors.WithStack(err)
	}

	for pkt := range ch {
		select {
		case <-server.Context().Done():
			return errors.WithStack(server.Context().Err())
		default:
		}

		switch {
		case pkt.Error != nil:
			return errors.WithStack(pkt.Error)

		case pkt.PackfileHeader != nil:
			err = server.Send(&pb.FetchFromCommitResponse{
				Payload: &pb.FetchFromCommitResponse_PackfileHeader_{&pb.FetchFromCommitResponse_PackfileHeader{
					PackfileID:       pkt.PackfileHeader.PackfileID,
					UncompressedSize: pkt.PackfileHeader.UncompressedSize,
				}},
			})

		case pkt.PackfileData != nil:
			err = server.Send(&pb.FetchFromCommitResponse{
				Payload: &pb.FetchFromCommitResponse_PackfileData_{&pb.FetchFromCommitResponse_PackfileData{
					PackfileID: pkt.PackfileData.PackfileID,
					Data:       pkt.PackfileData.Data,
					End:        pkt.PackfileData.End,
				}},
			})

		case pkt.Chunk != nil:
			err = server.Send(&pb.FetchFromCommitResponse{
				Payload: &pb.FetchFromCommitResponse_Chunk_{&pb.FetchFromCommitResponse_Chunk{
					ObjectID: pkt.Chunk.ObjectID,
					Data:     pkt.Chunk.Data,
					End:      pkt.Chunk.End,
				}},
			})

		}

		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (s *Server) FetchChunks(req *pb.FetchChunksRequest, server pb.NodeRPC_FetchChunksServer) error {
	ch := s.node.FetchChunks(context.TODO(), req.RepoID, req.Path, req.Chunks)

	for pkt := range ch {
		if pkt.Error != nil {
			return errors.WithStack(pkt.Error)
		}

		err := server.Send(&pb.FetchChunksResponse{
			ObjectID: pkt.Chunk.ObjectID,
			Data:     pkt.Chunk.Data,
			End:      pkt.Chunk.End,
		})
		if err != nil {
			return errors.WithStack(err)
		}
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
	_, err := s.node.RepoManager().TrackRepo(req.RepoPath, req.ForceReload)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &pb.TrackLocalRepoResponse{}, nil
}

func (s *Server) GetLocalRepos(req *pb.GetLocalReposRequest, server pb.NodeRPC_GetLocalReposServer) error {
	return s.node.RepoManager().ForEachRepo(func(r *repo.Repo) error {
		select {
		case <-server.Context().Done():
			return errors.WithStack(server.Context().Err())
		default:
		}

		repoID, err := r.RepoID()
		if err != nil {
			return errors.WithStack(err)
		}
		err = server.Send(&pb.GetLocalReposResponsePacket{RepoID: repoID, Path: r.Path()})
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
	r, err := s.node.RepoManager().RepoAtPathOrID(req.Path, req.RepoID)
	if err != nil {
		return nil, err
	}

	rIter, err := r.NewReferenceIterator()
	if err != nil {
		return nil, err
	}
	defer rIter.Free()

	refs := []*pb.Ref{}
	for {
		ref, err := rIter.Next()
		if git.IsErrorCode(err, git.ErrIterOver) {
			break
		} else if err != nil {
			return nil, err
		}

		ref, err = ref.Resolve()
		if err != nil {
			return nil, err
		}

		refs = append(refs, &pb.Ref{
			RefName:    ref.Name(),
			CommitHash: ref.Target().String(),
		})
	}

	return &pb.GetLocalRefsResponse{Refs: refs}, nil
}

func (s *Server) GetRemoteRefs(ctx context.Context, req *pb.GetRemoteRefsRequest) (*pb.GetRemoteRefsResponse, error) {
	refMap, total, err := s.node.GetRemoteRefs(ctx, req.RepoID, req.PageSize, req.Page)
	if err != nil {
		log.Errorln("[rpc server] error fetching remote refs:", err)
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

func (s *Server) SetRepoPublic(ctx context.Context, req *pb.SetRepoPublicRequest) (*pb.SetRepoPublicResponse, error) {
	tx, err := s.node.SetRepoPublic(ctx, req.RepoID, req.IsPublic)
	if err != nil {
		return nil, err
	}
	txResult := <-tx.Await(ctx)
	if txResult.Err != nil {
		return nil, txResult.Err
	} else if txResult.Receipt.Status == 0 {
		return nil, errors.New("transaction failed")
	}

	return &pb.SetRepoPublicResponse{RepoID: req.RepoID, IsPublic: req.IsPublic}, nil
}

func (s *Server) IsRepoPublic(ctx context.Context, req *pb.IsRepoPublicRequest) (*pb.IsRepoPublicResponse, error) {
	isPublic, err := s.node.IsRepoPublic(ctx, req.RepoID)
	if err != nil {
		return nil, err
	}
	return &pb.IsRepoPublicResponse{RepoID: req.RepoID, IsPublic: isPublic}, nil
}

func (s *Server) GetRepoUsers(ctx context.Context, req *pb.GetRepoUsersRequest) (*pb.GetRepoUsersResponse, error) {
	users, total, err := s.node.GetRepoUsers(ctx, req.RepoID, nodeeth.UserType(req.Type), req.PageSize, req.Page)
	if err != nil {
		return nil, err
	}
	return &pb.GetRepoUsersResponse{Total: total, Users: users}, nil
}

func (s *Server) RequestReplication(req *pb.ReplicationRequest, server pb.NodeRPC_RequestReplicationServer) error {
	ch := s.node.RequestReplication(context.TODO(), req.RepoID)
	for progress := range ch {
		select {
		case <-server.Context().Done():
			return errors.WithStack(server.Context().Err())
		default:
		}

		if progress.Error != nil {
			return errors.WithStack(progress.Error)
		}
		err := server.Send(&pb.ReplicationResponsePacket{Percent: int32(progress.Percent)})
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (s *Server) GetRepoHistory(ctx context.Context, req *pb.GetRepoHistoryRequest) (*pb.GetRepoHistoryResponse, error) {
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	r, err := s.node.RepoManager().RepoAtPathOrID(req.Path, req.RepoID)
	if err != nil {
		return nil, err
	}

	var fromCommit *git.Oid
	if len(req.FromCommitHash) > 0 || len(req.FromCommitRef) > 0 {
		oid, err := r.ResolveCommitHash(repo.CommitID{Hash: util.OidFromBytes(req.FromCommitHash), Ref: req.FromCommitRef})
		if err != nil {
			return nil, err
		}
		fromCommit = &oid

	} else {
		const branchName = "master" // @@TODO: add this field to the RPC request

		branch, err := r.LookupBranch(branchName, git.BranchLocal)
		if err != nil {
			return nil, err
		}
		fromCommit = branch.Target()
	}
	if err != nil {
		return nil, err
	}

	walker, err := r.Walk()
	if err != nil {
		return nil, err
	}

	walker.Sorting(git.SortTopological | git.SortTime /*| git.SortReverse*/)

	// When we're at the end of the range (at the beginning of the history), and we specify a `~N`
	// revspec that exceeds the number of commits available, libgit2 returns an error.  We deal with
	// this by simply looping with smaller and smaller values of N until we figure out how many
	// commits we have left.  @@TODO: this is probably inefficient (although maybe not, given that
	// revwalkers are heavily cached).
	numCommits := req.PageSize
	for {
		err = walker.PushRange(fmt.Sprintf("%s~%d..%s", fromCommit.String(), numCommits, fromCommit.String()))
		if err == nil {
			break
		}
		numCommits--
	}

	var innerErr error
	var commits []*pb.Commit
	err = walker.Iterate(func(commit *git.Commit) bool {
		commitHash := commit.Id().String()

		if req.OnlyHashes {
			commits = append(commits, &pb.Commit{
				CommitHash: commitHash,
			})

		} else {
			files, err := r.FilesChangedByCommit(ctx, commit.Id())
			if err != nil {
				innerErr = err
				return false
			}

			filenames := make([]string, len(files))
			for i := range files {
				filenames[i] = files[i].Filename
			}

			author := commit.Author()

			commits = append(commits, &pb.Commit{
				CommitHash: commitHash,
				Author:     author.Name + " <" + author.Email + ">", // @@TODO: break this up into .Name and .Email fields
				Message:    commit.Message(),
				Timestamp:  uint64(author.When.Unix()),
				Files:      filenames,
			})
		}

		if len(commits) >= int(req.PageSize) {
			return false
		}
		return true
	})

	isEnd := numCommits < req.PageSize

	return &pb.GetRepoHistoryResponse{Commits: commits, IsEnd: isEnd}, nil
}

func (s *Server) GetUpdatedRefEvents(ctx context.Context, req *pb.GetUpdatedRefEventsRequest) (*pb.GetUpdatedRefEventsResponse, error) {
	repoIDs := []string{req.RepoID}
	var end *uint64
	if req.EndBlock > 0 {
		end = &req.EndBlock
	}

	evts, err := s.node.GetUpdatedRefEvents(ctx, repoIDs, req.StartBlock, end)
	if err != nil {
		return nil, err
	}

	evtResp := make([]*pb.UpdatedRefEvent, len(evts))
	for i, evt := range evts {
		evtResp[i] = &pb.UpdatedRefEvent{
			Commit:      evt.Commit,
			RepoID:      req.RepoID,
			TxHash:      evt.TxHash,
			Time:        evt.Time,
			BlockNumber: evt.BlockNumber,
		}
	}

	return &pb.GetUpdatedRefEventsResponse{Events: evtResp}, nil
}

func (s *Server) GetRepoFiles(ctx context.Context, req *pb.GetRepoFilesRequest) (*pb.GetRepoFilesResponse, error) {
	r, err := s.node.RepoManager().RepoAtPathOrID(req.RepoRoot, req.RepoID)
	if err != nil {
		return nil, err
	}

	fileList, err := r.ListFiles(ctx, repo.CommitID{Hash: util.OidFromBytes(req.CommitHash), Ref: req.CommitRef})
	if err != nil {
		return nil, err
	}

	files := make([]*pb.File, len(fileList))
	for i := range fileList {
		files[i] = &pb.File{
			Name: fileList[i].Filename,
			Hash: fileList[i].Hash[:],
			// Mode:           uint32(fileList[i].Mode),
			Size:           fileList[i].Size,
			Modified:       fileList[i].Modified,
			UnstagedStatus: string(fileList[i].Status.Unstaged),
			StagedStatus:   string(fileList[i].Status.Staged),
			IsChunked:      fileList[i].IsChunked,
		}
	}

	return &pb.GetRepoFilesResponse{Files: files}, nil
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

func (s *Server) GetObject(req *pb.GetObjectRequest, server pb.NodeRPC_GetObjectServer) error {
	r, err := s.node.RepoManager().RepoAtPathOrID(req.RepoRoot, req.RepoID)
	if err != nil {
		return err
	}

	var objectReader repo.ObjectReader

	if len(req.ObjectID) > 0 {
		objectReader, err = r.OpenObject(req.ObjectID)
		if err != nil {
			return err
		}

	} else {
		if len(req.CommitHash) != 20 && req.CommitRef == "" {
			return errors.New("need commitHash or commitRef")
		} else if len(req.Filename) == 0 {
			return errors.New("need filename")
		}

		if req.CommitRef == "working" {
			objectReader, err = r.OpenFileInWorktree(req.Filename)
			if err != nil {
				return err
			}

		} else {
			objectReader, err = r.OpenFileAtCommit(req.Filename, repo.CommitID{Hash: util.OidFromBytes(req.CommitHash), Ref: req.CommitRef})
			if err != nil {
				return err
			}
		}
	}
	defer objectReader.Close()

	err = server.Send(&pb.GetObjectResponse{
		Payload: &pb.GetObjectResponse_Header_{&pb.GetObjectResponse_Header{
			UncompressedSize: objectReader.Len(),
		}},
	})
	if err != nil {
		return err
	}

	totalBytes := req.MaxSize
	if totalBytes > objectReader.Len() {
		totalBytes = objectReader.Len()
	}

	var sent uint64
	for sent < totalBytes {
		bufSize := uint64(OBJ_CHUNK_SIZE)
		if sent+bufSize > totalBytes {
			bufSize = totalBytes - sent
		}

		data := make([]byte, bufSize)
		n, err := io.ReadFull(objectReader, data)
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			data = data[:n]
		} else if err != nil {
			return err
		}

		sent += uint64(n)

		err = server.Send(&pb.GetObjectResponse{
			Payload: &pb.GetObjectResponse_Data_{&pb.GetObjectResponse_Data{
				Data: data,
			}},
		})
		if err != nil {
			return err
		}

		if sent == totalBytes {
			break
		}
	}

	err = server.Send(&pb.GetObjectResponse{
		Payload: &pb.GetObjectResponse_Data_{&pb.GetObjectResponse_Data{
			End: true,
		}},
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) GetDiff(req *pb.GetDiffRequest, server pb.NodeRPC_GetDiffServer) error {
	r, err := s.node.RepoManager().RepoAtPathOrID(req.RepoRoot, req.RepoID)
	if err != nil {
		return err
	}

	if len(req.CommitHash) != 20 && req.CommitRef == "" {
		return errors.New("need commitHash or commitRef")
	}

	diffReader, err := r.GetDiff(context.TODO(), repo.CommitID{Hash: util.OidFromBytes(req.CommitHash), Ref: req.CommitRef})
	if err != nil {
		return err
	}

	for {
		data := make([]byte, OBJ_CHUNK_SIZE)
		n, err := io.ReadFull(diffReader, data)
		if err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			data = data[:n]
		} else if err != nil {
			return err
		}

		err = server.Send(&pb.GetDiffResponse{Data: data})
		if err != nil {
			return errors.WithStack(err)
		}
	}

	err = server.Send(&pb.GetDiffResponse{End: true})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (s *Server) SetFileChunking(ctx context.Context, req *pb.SetFileChunkingRequest) (*pb.SetFileChunkingResponse, error) {
	r, err := s.node.RepoAtPathOrID(req.RepoRoot, req.RepoID)
	if err != nil {
		return nil, err
	}

	err = r.SetFileChunking(req.Filename, req.Enabled)
	if err != nil {
		return nil, err
	}

	return &pb.SetFileChunkingResponse{}, nil
}

func (s *Server) Watch(req *pb.WatchRequest, server pb.NodeRPC_WatchServer) error {
	eventTypes := make([]swarm.EventType, 0)
	for _, t := range req.EventTypes {
		eventTypes = append(eventTypes, swarm.EventType(t))
	}

	settings := &swarm.WatcherSettings{
		EventTypes:      eventTypes,
		UpdatedRefStart: req.UpdatedRefStart,
	}

	ctx := context.Background()
	watcher := s.node.Watch(ctx, settings)

	for evt := range watcher.EventCh {
		select {
		case <-server.Context().Done():
			return errors.WithStack(server.Context().Err())
		default:
		}

		var err error

		switch {
		case evt.Error != nil:
			return errors.WithStack(evt.Error)

		case evt.AddedRepoEvent != nil:
			err = server.Send(&pb.WatchResponse{
				Payload: &pb.WatchResponse_AddedRepoEvent_{&pb.WatchResponse_AddedRepoEvent{
					RepoID:   evt.AddedRepoEvent.RepoID,
					RepoRoot: evt.AddedRepoEvent.RepoRoot,
				}},
			})

		case evt.PulledRepoEvent != nil:
			err = server.Send(&pb.WatchResponse{
				Payload: &pb.WatchResponse_PulledRepoEvent_{&pb.WatchResponse_PulledRepoEvent{
					RepoID:   evt.PulledRepoEvent.RepoID,
					RepoRoot: evt.PulledRepoEvent.RepoRoot,
					NewHEAD:  evt.PulledRepoEvent.NewHEAD,
				}},
			})

		case evt.UpdatedRefEvent != nil:
			err = server.Send(&pb.WatchResponse{
				Payload: &pb.WatchResponse_UpdatedRefEvent_{&pb.WatchResponse_UpdatedRefEvent{
					Commit:      evt.UpdatedRefEvent.Commit,
					RepoID:      evt.UpdatedRefEvent.RepoID,
					TxHash:      evt.UpdatedRefEvent.TxHash,
					Time:        evt.UpdatedRefEvent.Time,
					BlockNumber: evt.UpdatedRefEvent.BlockNumber,
				}},
			})

		}

		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
