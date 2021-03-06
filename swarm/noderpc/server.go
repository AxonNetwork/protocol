package noderpc

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/libgit2/git2go"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm"
	"github.com/Conscience/protocol/swarm/nodeeth"
	"github.com/Conscience/protocol/swarm/nodeevents"
	"github.com/Conscience/protocol/swarm/nodeexec"
	"github.com/Conscience/protocol/swarm/nodep2p"
	"github.com/Conscience/protocol/swarm/noderpc/pb"
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
	tx, err := s.node.EthereumClient().EnsureUsername(ctx, req.Username)
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
	signature, err := s.node.EthereumClient().SignHash([]byte(req.Username))
	if err != nil {
		return nil, err
	}
	return &pb.SetUsernameResponse{Signature: signature}, nil
}

func (s *Server) GetUsername(ctx context.Context, req *pb.GetUsernameRequest) (*pb.GetUsernameResponse, error) {
	un, err := s.node.EthereumClient().GetUsername(ctx)
	if err != nil {
		return nil, err
	}
	signature, err := s.node.EthereumClient().SignHash([]byte(un))
	if err != nil {
		return nil, err
	}

	return &pb.GetUsernameResponse{Username: un, Signature: signature}, nil
}

func (s *Server) GetEthereumBIP39Seed(ctx context.Context, req *pb.GetEthereumBIP39SeedRequest) (*pb.GetEthereumBIP39SeedResponse, error) {
	return &pb.GetEthereumBIP39SeedResponse{Seed: s.node.Config.Node.EthereumBIP39Seed}, nil
}

func (s *Server) SetEthereumBIP39Seed(ctx context.Context, req *pb.SetEthereumBIP39SeedRequest) (*pb.SetEthereumBIP39SeedResponse, error) {
	err := s.node.Config.Update(func() error {
		s.node.Config.Node.EthereumBIP39Seed = req.Seed
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &pb.SetEthereumBIP39SeedResponse{}, nil
}

func (s *Server) InitRepo(ctx context.Context, req *pb.InitRepoRequest) (*pb.InitRepoResponse, error) {
	if req.RepoID == "" {
		return nil, errors.New("empty repoID")
	}

	// Before anything else, try to claim the repoID in the smart contract
	isRegistered, err := s.node.EthereumClient().IsRepoIDRegistered(ctx, req.RepoID)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if isRegistered {
		return nil, errors.New("repoID already registered")
	}

	tx, err := s.node.EthereumClient().RegisterRepoID(ctx, req.RepoID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	log.Printf("[rpc] create repo tx sent: %s", tx.Hash().Hex())

	txResult := <-tx.Await(ctx)
	if txResult.Err != nil {
		return nil, errors.WithStack(txResult.Err)
	}

	log.Printf("[rpc] create repo tx resolved: %s", tx.Hash().Hex())

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

	_, err = r.RepoID()
	if err != nil {
		err = r.SetupConfig(req.RepoID)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// Have the node track the local repo
	_, err = s.node.RepoManager().TrackRepo(path, true)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// if local HEAD exists, push to contract
	_, err = r.Head()
	if err == nil {
		_, err = s.node.P2PHost().Push(ctx, &nodep2p.PushOptions{
			Repo:       r,
			BranchName: "master", // @@TODO: don't hard code this @@branches
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

func (s *Server) ImportRepo(ctx context.Context, req *pb.ImportRepoRequest) (*pb.ImportRepoResponse, error) {
	if req.RepoRoot == "" {
		return nil, errors.New("missing RepoRoot param")
	}

	r, err := repo.Open(req.RepoRoot)
	if err != nil {
		return nil, err
	}

	var isUninitialized bool

	_, err = r.RepoID()
	if errors.Cause(err) == repo.ErrNoRepoID {
		isUninitialized = true
	} else if err != nil {
		return nil, err
	}

	if !isUninitialized && req.RepoID != "" {
		return nil, errors.New("repo already initialized, cannot specify a new RepoID") // @@TODO: maybe relax this restriction
	} else if isUninitialized && req.RepoID == "" {
		return nil, errors.New("must specify RepoID if repo is not initialized")
	}

	if isUninitialized {
		// Before anything else, try to claim the repoID in the smart contract
		isRegistered, err := s.node.EthereumClient().IsRepoIDRegistered(ctx, req.RepoID)
		if err != nil {
			return nil, errors.WithStack(err)
		} else if isRegistered {
			return nil, errors.New("repoID already registered")
		}

		tx, err := s.node.EthereumClient().RegisterRepoID(ctx, req.RepoID)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		log.Infof("[rpc] create repo tx sent: %s", tx.Hash().Hex())

		txResult := <-tx.Await(ctx)
		if txResult.Err != nil {
			return nil, errors.WithStack(txResult.Err)
		}

		log.Infof("[rpc] create repo tx resolved: %s", tx.Hash().Hex())

		err = r.SetupConfig(req.RepoID)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// Have the node track the local repo
	_, err = s.node.RepoManager().TrackRepo(req.RepoRoot, true)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &pb.ImportRepoResponse{}, nil
}

func (s *Server) CheckpointRepo(ctx context.Context, req *pb.CheckpointRepoRequest) (*pb.CheckpointRepoResponse, error) {
	r := s.node.RepoManager().RepoAtPath(req.Path)
	if r == nil {
		return nil, errors.WithStack(repo.Err404)
	}

	_, err := r.CommitCurrentWorkdir(&repo.CommitOptions{
		Pathspecs: []string{""},
		Message:   req.Message,
	})
	if err != nil {
		log.Errorln("[checkpoint]  - error:", err)
		return nil, errors.WithStack(err)
	}

	_, err = s.node.P2PHost().Push(ctx, &nodep2p.PushOptions{
		Repo:       r,
		BranchName: "master", // @@TODO: don't hard code this @@branches
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

	// @@TODO: don't hardcode origin/master @@branches
	_, err := s.node.P2PHost().Pull(context.TODO(), &nodep2p.PullOptions{
		Repo:       r,
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
	replDir := req.Path
	if len(replDir) == 0 {
		replDir = s.node.Config.Node.ReplicationRoot
	}
	repoRoot := filepath.Join(replDir, req.RepoID)

	r, err := s.node.P2PHost().Clone(context.TODO(), &nodep2p.CloneOptions{
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
	ctx := server.Context()

	r, err := s.node.RepoManager().RepoAtPathOrID(req.Path, req.RepoID)
	if errors.Cause(err) == repo.Err404 {
		// no-op
	} else if err != nil {
		return errors.WithStack(err)
	}

	ch, manifest, err := s.node.P2PHost().FetchFromCommit(ctx, req.RepoID, r, *util.OidFromBytes(req.Commit), nodep2p.CheckoutType(req.CheckoutType), nil)
	if err != nil {
		return err
	}

	uncompressedSize := manifest.GitObjects.UncompressedSize() + manifest.ChunkObjects.UncompressedSize()
	totalChunks := len(manifest.ChunkObjects)

	err = server.Send(&pb.FetchFromCommitResponse{
		Payload: &pb.FetchFromCommitResponse_Header_{&pb.FetchFromCommitResponse_Header{
			UncompressedSize: uncompressedSize,
			TotalChunks:      int64(totalChunks),
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
	var chunkManifest []nodep2p.ManifestObject
	for i := range req.Chunks {
		chunkManifest = append(chunkManifest, nodep2p.ManifestObject{Hash: req.Chunks[i]})
	}

	ch := s.node.P2PHost().FetchChunks(context.TODO(), req.RepoID, chunkManifest)

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
	isRegistered, err := s.node.EthereumClient().IsRepoIDRegistered(ctx, req.RepoID)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if isRegistered {
		return nil, errors.New("repoID already registered")
	}

	tx, err := s.node.EthereumClient().RegisterRepoID(ctx, req.RepoID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	log.Printf("[rpc] create repo tx sent: %s", tx.Hash().Hex())
	txResult := <-tx.Await(ctx)
	if txResult.Err != nil {
		return nil, errors.WithStack(txResult.Err)
	}
	log.Printf("[rpc] create repo tx resolved: %s", tx.Hash().Hex())

	return &pb.RegisterRepoIDResponse{}, nil
}

func (s *Server) IsRepoIDRegistered(ctx context.Context, req *pb.IsRepoIDRegisteredRequest) (*pb.IsRepoIDRegisteredResponse, error) {
	isRegistered, err := s.node.EthereumClient().IsRepoIDRegistered(ctx, req.RepoID)
	if err != nil {
		return nil, err
	}
	return &pb.IsRepoIDRegisteredResponse{IsRegistered: isRegistered}, nil
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
	err := s.node.P2PHost().SetReplicationPolicy(req.RepoID, req.MaxBytes)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &pb.SetReplicationPolicyResponse{}, nil
}

func (s *Server) AnnounceRepoContent(ctx context.Context, req *pb.AnnounceRepoContentRequest) (*pb.AnnounceRepoContentResponse, error) {
	err := s.node.P2PHost().AnnounceRepo(ctx, req.RepoID)
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

	var refs []*pb.Ref
	err = r.ForEachLocalRef(func(ref *git.Reference) error {
		refs = append(refs, &pb.Ref{
			RefName:    ref.Name(),
			CommitHash: ref.Target().String(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &pb.GetLocalRefsResponse{Refs: refs}, nil
}

func (s *Server) GetRemoteRefs(ctx context.Context, req *pb.GetRemoteRefsRequest) (*pb.GetRemoteRefsResponse, error) {
	refMap, total, err := s.node.EthereumClient().GetRemoteRefs(ctx, req.RepoID, req.PageSize, req.Page)
	if err != nil {
		log.Errorln("[rpc server] error fetching remote refs:", err)
		return nil, err
	}

	var refs []*pb.Ref
	for _, ref := range refMap {
		refs = append(refs, &pb.Ref{RefName: ref.RefName, CommitHash: ref.CommitHash})
	}

	return &pb.GetRemoteRefsResponse{Total: total, Refs: refs}, nil
}

func (s *Server) IsBehindRemote(ctx context.Context, req *pb.IsBehindRemoteRequest) (*pb.IsBehindRemoteResponse, error) {
	r, err := s.node.RepoManager().RepoAtPathOrID(req.Path, req.RepoID)
	if err != nil {
		return nil, err
	}

	// @@TODO: don't hard code this @@branches
	remote, err := s.node.EthereumClient().GetRemoteRef(ctx, req.RepoID, "refs/heads/master")
	if err != nil {
		return nil, err
	}

	if len(remote) == 0 {
		return nil, nil
	}

	return &pb.IsBehindRemoteResponse{RepoID: req.RepoID, IsBehindRemote: !r.HasObject(remote[:])}, nil
}

func (s *Server) PushRepo(req *pb.PushRepoRequest, server pb.NodeRPC_PushRepoServer) error {
	r := s.node.RepoManager().RepoAtPath(req.RepoRoot)
	if r == nil {
		return repo.ErrNotTracked
	}

	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	var innerErr error
	_, err := s.node.P2PHost().Push(ctx, &nodep2p.PushOptions{
		Repo:       r,
		BranchName: req.BranchName,
		Force:      req.Force,
		ProgressCb: func(percent int) {
			innerErr = server.Send(&pb.ProgressPacket{Current: uint64(percent), Total: 100})
			if innerErr != nil {
				cancel()
				return
			}
		},
	})
	if innerErr != nil {
		return innerErr
	} else if err != nil {
		return err
	}

	err = server.Send(&pb.ProgressPacket{Current: 100, Total: 100, Done: true})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SetRepoPublic(ctx context.Context, req *pb.SetRepoPublicRequest) (*pb.SetRepoPublicResponse, error) {
	tx, err := s.node.EthereumClient().SetRepoPublic(ctx, req.RepoID, req.IsPublic)
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
	isPublic, err := s.node.EthereumClient().IsRepoPublic(ctx, req.RepoID)
	if err != nil {
		return nil, err
	}
	return &pb.IsRepoPublicResponse{RepoID: req.RepoID, IsPublic: isPublic}, nil
}

func (s *Server) GetRepoUsers(ctx context.Context, req *pb.GetRepoUsersRequest) (*pb.GetRepoUsersResponse, error) {
	users, total, err := s.node.EthereumClient().GetRepoUsers(ctx, req.RepoID, nodeeth.UserType(req.Type), req.PageSize, req.Page)
	if err != nil {
		return nil, err
	}
	return &pb.GetRepoUsersResponse{Total: total, Users: users}, nil
}

func (s *Server) RequestReplication(req *pb.ReplicationRequest, server pb.NodeRPC_RequestReplicationServer) error {
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	ch, err := s.node.P2PHost().RequestReplication(ctx, req.RepoID)
	if err != nil {
		return err
	}

	for progress := range ch {
		select {
		case <-server.Context().Done():
			return errors.WithStack(server.Context().Err())
		default:
		}

		if progress.Error != nil {
			return errors.WithStack(progress.Error)
		}
		err := server.Send(&pb.ProgressPacket{Current: uint64(progress.Current), Total: 100})
		if err != nil {
			return errors.WithStack(err)
		}
	}

	err = server.Send(&pb.ProgressPacket{Current: 100, Total: 100, Done: true})
	if err != nil {
		return errors.WithStack(err)
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
		const branchName = "master" // @@TODO: add this field to the RPC request @@branches

		branch, err := r.LookupBranch(branchName, git.BranchLocal)
		if git.IsErrorCode(err, git.ErrNotFound) {
			// empty repo with no timeline
			return &pb.GetRepoHistoryResponse{Commits: []*pb.Commit{}, IsEnd: true}, nil
		} else if err != nil {
			return nil, err
		}
		fromCommit = branch.Target()
	}

	commit, err := r.LookupCommit(fromCommit)
	if err != nil {
		return nil, err
	}

	var commits []*pb.Commit
	for {
		if req.OnlyHashes {
			commits = append(commits, &pb.Commit{
				CommitHash: commit.Id().String(),
			})

		} else {
			files, err := r.FilesChangedByCommit(ctx, commit.Id())
			if err != nil {
				return nil, err
			}

			filenames := make([]string, len(files))
			for i := range files {
				filenames[i] = files[i].Filename
			}

			author := commit.Author()

			commits = append(commits, &pb.Commit{
				CommitHash: commit.Id().String(),
				Author:     author.Name + " <" + author.Email + ">", // @@TODO: break this up into .Name and .Email fields
				Message:    commit.Message(),
				Timestamp:  uint64(author.When.Unix()),
				Files:      filenames,
			})
		}

		if len(commits) >= int(req.PageSize) {
			break
		} else if commit.ParentCount() == 0 {
			break
		}
		commit = commit.Parent(0)
	}

	isEnd := commit.ParentCount() == 0

	return &pb.GetRepoHistoryResponse{Commits: commits, IsEnd: isEnd}, nil
}

func (s *Server) GetUpdatedRefEvents(ctx context.Context, req *pb.GetUpdatedRefEventsRequest) (*pb.GetUpdatedRefEventsResponse, error) {
	repoIDs := []string{req.RepoID}
	var end *uint64
	if req.EndBlock > 0 {
		end = &req.EndBlock
	}

	evts, err := s.node.EthereumClient().GetUpdatedRefEvents(ctx, repoIDs, req.StartBlock, end)
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
	signature, err := s.node.EthereumClient().SignHash(req.Message)
	if err != nil {
		return nil, err
	}
	return &pb.SignMessageResponse{Signature: signature}, nil
}

func (s *Server) EthAddress(ctx context.Context, req *pb.EthAddressRequest) (*pb.EthAddressResponse, error) {
	return &pb.EthAddressResponse{Address: s.node.EthereumClient().Address().String()}, nil
}

func (s *Server) GetUserPermissions(ctx context.Context, req *pb.GetUserPermissionsRequest) (*pb.GetUserPermissionsResponse, error) {
	perms, err := s.node.EthereumClient().GetUserPermissions(ctx, req.RepoID, req.Username)
	if err != nil {
		return nil, err
	}
	return &pb.GetUserPermissionsResponse{Puller: perms.Puller, Pusher: perms.Pusher, Admin: perms.Admin}, nil
}

func (s *Server) SetUserPermissions(ctx context.Context, req *pb.SetUserPermissionsRequest) (*pb.SetUserPermissionsResponse, error) {
	tx, err := s.node.EthereumClient().SetUserPermissions(ctx, req.RepoID, req.Username, nodeeth.UserPermissions{Puller: req.Puller, Pusher: req.Pusher, Admin: req.Admin})
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
	} else if diffReader == nil {
		err = server.Send(&pb.GetDiffResponse{End: true})
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	}

	if diffReader == nil {
		err = server.Send(&pb.GetDiffResponse{End: true})
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
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
	r, err := s.node.RepoManager().RepoAtPathOrID(req.RepoRoot, req.RepoID)
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
	var eventTypes nodeevents.EventType
	for _, t := range req.EventTypes {
		eventTypes |= nodeevents.EventType(t)
	}

	// settings := &nodeevents.WatcherSettings{EventTypes: eventTypes}

	// if eventTypes&nodeevents.EventType_UpdatedRef != 0 {
	// 	if req.UpdatedRefEventParams == nil {
	// 		return errors.New("to watch UpdatedRef events, you must provide UpdatedRefEventParams")
	// 	}

	// 	settings.UpdatedRefEventParams.FromBlock = req.UpdatedRefEventParams.FromBlock
	// 	settings.UpdatedRefEventParams.RepoIDs = req.UpdatedRefEventParams.RepoIDs
	// }

	watcher := s.node.EventBus().Watch(&nodeevents.WatcherSettings{EventTypes: eventTypes})
	defer s.node.EventBus().Unwatch(watcher)

	for {
		var evt nodeevents.MaybeEvent
		var open bool

		select {
		case <-server.Context().Done():
			return errors.WithStack(server.Context().Err())
		case evt, open = <-watcher.Events():
			if !open {
				return errors.New("event watcher closed early")
			}
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
					RepoID:      evt.PulledRepoEvent.RepoID,
					RepoRoot:    evt.PulledRepoEvent.RepoRoot,
					UpdatedRefs: evt.PulledRepoEvent.UpdatedRefs,
				}},
			})

		case evt.PushedRepoEvent != nil:
			err = server.Send(&pb.WatchResponse{
				Payload: &pb.WatchResponse_PushedRepoEvent_{&pb.WatchResponse_PushedRepoEvent{
					RepoID:     evt.PushedRepoEvent.RepoID,
					RepoRoot:   evt.PushedRepoEvent.RepoRoot,
					BranchName: evt.PushedRepoEvent.BranchName,
					Commit:     evt.PushedRepoEvent.Commit,
				}},
			})

			// case evt.UpdatedRefEvent != nil:
			// 	err = server.Send(&pb.WatchResponse{
			// 		Payload: &pb.WatchResponse_UpdatedRefEvent_{&pb.WatchResponse_UpdatedRefEvent{
			// 			Commit:      evt.UpdatedRefEvent.Commit,
			// 			RepoID:      evt.UpdatedRefEvent.RepoID,
			// 			TxHash:      evt.UpdatedRefEvent.TxHash,
			// 			Time:        evt.UpdatedRefEvent.Time,
			// 			BlockNumber: evt.UpdatedRefEvent.BlockNumber,
			// 		}},
			// 	})

		}

		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (s *Server) CreateCommit(server pb.NodeRPC_CreateCommitServer) error {
	// @@TODO: per-repo mutex

	pkt, err := server.Recv()
	if err != nil {
		return errors.WithStack(err)
	}

	pktHeader := pkt.GetHeader()
	if pktHeader == nil {
		return errors.New("[noderpc] CreateCommit protocol error: did not receive a header packet")
	}

	r := s.node.RepoManager().Repo(pktHeader.RepoID)
	if r == nil {
		return errors.Wrapf(repo.Err404, "[noderpc] CreateCommit bad request:")
	}

	if len(pktHeader.ParentCommitHash) != 20 {
		return errors.New("[noderpc] CreateCommit bad request: ParentCommitHash must be 20 bytes")
	}

	//
	// Create a blank index.  If a parent commit was specified, copy it into the index as our
	// initial index state.
	//
	idx, err := git.NewIndex()
	if err != nil {
		return errors.WithStack(err)
	}

	var parentCommit *git.Commit
	if parentCommitHash := util.OidFromBytes(pktHeader.ParentCommitHash); !parentCommitHash.IsZero() {
		parentCommit, err = r.LookupCommit(parentCommitHash)
		if err != nil {
			return errors.WithStack(err)
		}

		parentCommitTree, err := parentCommit.Tree()
		if err != nil {
			return errors.WithStack(err)
		}

		err = idx.ReadTree(parentCommitTree)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	branch, err := r.LookupBranch(pktHeader.RefName, git.BranchLocal)
	if git.IsErrorCode(err, git.ErrNotFound) == false {
		if parentCommit == nil {
			return errors.New("[noderpc] CreateCommit bad request: that branch already exists.  You must specify its current HEAD as the ParentCommitHash if you want to add a new commit to it.")
		}

		branchHeadObj, err := branch.Peel(git.ObjectCommit)
		if err != nil {
			return errors.WithStack(err)
		}

		branchHeadCommit, err := branchHeadObj.AsCommit()
		if err != nil {
			return errors.WithStack(err)
		}

		if branchHeadCommit.Id().Equal(parentCommit.Id()) == false {
			return errors.New("[noderpc] CreateCommit bad request: the parent commit you specified is not the current HEAD of the branch you specified")
		}
	}

	//
	// Stream the updates into the new index and the ODB
	//
	odb, err := r.Odb()
	if err != nil {
		return errors.WithStack(err)
	}

	var (
		receivingUpsertData bool
		upsertHeader        *pb.CreateCommitRequest_FileOperation_UpsertHeader
		writeStream         *git.OdbWriteStream
	)
	defer func() {
		if writeStream != nil {
			writeStream.Close()
		}
	}()
	for {
		pkt, err = server.Recv()
		if err != nil {
			return errors.WithStack(err)
		}

		// If we receive a 'Done' packet, break out of the read loop
		if pkt.GetDone() != nil {
			break
		}

		// Otherwise, we read a stream of FileOperation packets that build the commit
		payload := pkt.GetFileOperation()
		if payload == nil {
			return errors.New("[noderpc] CreateCommit protocol error: unexpected packet")
		}

		if _upsertHeader := payload.GetUpsertHeader(); _upsertHeader != nil {
			upsertHeader = _upsertHeader

			if receivingUpsertData {
				return errors.New("[noderpc] CreateCommit protocol error: unexpected packet")
			} else if upsertHeader.Filename == "" {
				return errors.New("[noderpc] CreateCommit bad request: missing filename")
			} else if upsertHeader.UncompressedSize == 0 {
				return errors.New("[noderpc] CreateCommit bad request: uncompressedSize must be > 0")
			} else if upsertHeader.Ctime == 0 {
				return errors.New("[noderpc] CreateCommit bad request: ctime must be > 0")
			} else if upsertHeader.Mtime == 0 {
				return errors.New("[noderpc] CreateCommit bad request: mtime must be > 0")
			}
			if upsertHeader.Ctime > upsertHeader.Mtime {
				log.Warnln("[noderpc] CreateCommit: ctime > mtime")
			}

			receivingUpsertData = true

			writeStream, err = odb.NewWriteStream(int64(upsertHeader.UncompressedSize), git.ObjectBlob)
			if err != nil {
				return errors.WithStack(err)
			}

		} else if upsertData := payload.GetUpsertData(); upsertData != nil {

			if upsertData.End == false {
				// This is a 'data' packet
				n, err := writeStream.Write(upsertData.Data)
				if err != nil {
					return errors.WithStack(err)
				} else if n < len(upsertData.Data) {
					return errors.New("[noderpc] CreateCommit i/o error: did not finish writing")
				}

			} else if upsertData.End == true {
				// This is an 'end of data' packet
				receivingUpsertData = false

				err = writeStream.Close()
				if err != nil {
					return errors.WithStack(err)
				}

				oid := writeStream.Id
				writeStream = nil

				entry, err := idx.EntryByPath(upsertHeader.Filename, int(git.IndexStageNormal))
				if err != nil && git.IsErrorCode(err, git.ErrNotFound) == false {
					return errors.WithStack(err)

				} else if git.IsErrorCode(err, git.ErrNotFound) {
					// Adding a new entry
					idx.Add(&git.IndexEntry{
						Ctime: git.IndexTime{
							Seconds: int32(upsertHeader.Ctime),
							// Nanoseconds: uint32(time.Now().UnixNano()),
						},
						Mtime: git.IndexTime{
							Seconds: int32(upsertHeader.Mtime),
						},
						Mode: git.FilemodeBlob,
						Uid:  uint32(os.Getuid()),
						Gid:  uint32(os.Getgid()),
						Size: uint32(upsertHeader.UncompressedSize),
						Id:   &oid,
						Path: upsertHeader.Filename,
					})

				} else {
					// Updating an existing entry
					entry.Id = &oid
					entry.Size = uint32(upsertHeader.UncompressedSize)
					entry.Ctime.Seconds = int32(upsertHeader.Ctime)
					entry.Mtime.Seconds = int32(upsertHeader.Mtime)
				}
			}

		} else if delete := payload.GetDelete(); delete != nil {
			if delete.Filename == "" {
				return errors.New("[noderpc] CreateCommit bad request: missing filename")
			}

			err = idx.RemoveByPath(delete.Filename)
			if err != nil {
				return errors.WithStack(err)
			}

		} else {
			return errors.New("[noderpc] CreateCommit protocol error: received empty packet")
		}
	}

	log.Debugln("[noderpc] CreateCommit: finished update packets")

	//
	// Write the new tree object to disk
	//
	treeOid, err := idx.WriteTreeTo(r.Repository)
	if err != nil {
		return errors.WithStack(err)
	}
	log.Debugln("[noderpc] CreateCommit: wrote tree to disk", treeOid)

	tree, err := r.LookupTree(treeOid)
	if err != nil {
		return errors.WithStack(err)
	}

	//
	// Create a commit based on the new tree
	//
	var (
		now       = time.Now()
		message   = pktHeader.CommitMessage
		author    = &git.Signature{Name: pktHeader.AuthorName, Email: pktHeader.AuthorEmail, When: now}
		committer = &git.Signature{Name: pktHeader.AuthorName, Email: pktHeader.AuthorEmail, When: now}
	)

	var parentCommits []*git.Commit
	if parentCommit != nil {
		parentCommits = append(parentCommits, parentCommit)
	}

	newCommitHash, err := r.CreateCommit("refs/heads/"+pktHeader.RefName, author, committer, message, tree, parentCommits...)
	if err != nil {
		return errors.WithStack(err)
	}
	log.Debugln("[noderpc] CreateCommit: created commit", newCommitHash)

	//
	// Send an Ethereum transaction updating the ref to the new commit
	//
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second) // @@TODO: make this configurable
	_, err = s.node.P2PHost().Push(ctx, &nodep2p.PushOptions{
		Repo:       r,
		BranchName: pktHeader.RefName,
		ProgressCb: func(percent int) {
			// @@TODO: stream progress over RPC
		},
	})
	if err != nil {
		return errors.WithStack(err)
	}
	log.Debugln("[noderpc] CreateCommit: pushed to network")

	err = server.SendAndClose(&pb.CreateCommitResponse{Success: true, CommitHash: newCommitHash[:]})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (s *Server) RunPipeline(ctx context.Context, req *pb.RunPipelineRequest) (*pb.RunPipelineResponse, error) {
	inputRepo := s.node.RepoManager().Repo(req.InputRepoID)
	if inputRepo == nil {
		panic("fuck") // @@TODO: try to fetch?
	}

	inputObject, err := inputRepo.OpenObject(req.InputObjectID)
	if err != nil {
		panic(err) // @@TODO: try to fetch?
	}

	var inputStages []nodeexec.InputStage

	for _, stage := range req.Stages {
		stageRepo := s.node.RepoManager().Repo(stage.CodeRepoID)
		if stageRepo == nil {
			panic("fuck") // @@TODO: try to fetch?
		}

		commitID := repo.CommitID{Hash: util.OidFromBytes(stage.CommitHash)}

		files, err := stageRepo.ListFiles(context.Background(), commitID)
		if err != nil {
			return nil, err
		}

		var stageFiles []nodeexec.File
		for _, file := range files {
			contentsReader, err := stageRepo.OpenFileAtCommit(file.Filename, commitID)
			if err != nil {
				return nil, err
			}

			stageFiles = append(stageFiles, nodeexec.File{
				Filename: file.Filename,
				Size:     int64(file.Size),
				Contents: contentsReader,
			})
		}

		inputStages = append(inputStages, nodeexec.InputStage{
			Platform:      stage.Platform,
			Files:         stageFiles,
			EntryFilename: stage.EntryFilename,
			EntryArgs:     stage.EntryArgs,
		})
	}

	pipelineIn, pipelineOut, err := nodeexec.StartPipeline(inputStages)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	log.Warnln("done starting pipeline!")

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		buf := make([]byte, 1024)
		for {
			log.Warnln("[read] reading...")
			n, err := io.ReadFull(pipelineOut, buf)
			if err == io.EOF {
				break
			} else if err == io.ErrUnexpectedEOF {

			} else if err != nil {
				panic(err)
			}
			log.Warnln("[read]", n, "bytes")

			log.Warnln("[out]", string(buf[:n]))
		}
	}()

	// data := []byte("Us and them\nAnd after all we're only ordinary men\nMe, and you\nGod only knows it's not what we would choose to do\nForward he cried from the rear\nAnd the front rank died\nAnd the General sat, as the lines on the map\nMoved from side to side\nBlack and Blue\nAnd who knows which is which and who is who\nUp and down\nAnd in the end it's only round and round and round\nHaven't you heard it's a battle of words\nThe poster bearer cried\nListen son, said the man with the gun\nThere's room for you inside\nDown and out\nIt can't be helped but there's a lot of it about\nWith, without\nAnd who'll deny that's what the fighting's all about\nGet out of the way, it's a busy day\nAnd I've got things on my mind\nFor want of the price of tea and a slice\nThe old man died")
	log.Warnln("[write] writing...")
	num, err := io.Copy(pipelineIn, inputObject)
	if err != nil {
		return nil, err
	}
	log.Warnln("[write] wrote", num, "bytes")
	pipelineIn.Close()

	wg.Wait()

	return &pb.RunPipelineResponse{}, nil
}
