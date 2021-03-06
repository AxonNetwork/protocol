package noderpc

import (
	"context"
	"io"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/libgit2/git2go"

	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/nodeeth"
	"github.com/Conscience/protocol/swarm/nodep2p"
	"github.com/Conscience/protocol/swarm/noderpc/pb"
)

type Client struct {
	client pb.NodeRPCClient
	conn   *grpc.ClientConn
	host   string
}

func NewClient(host string) (*Client, error) {
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &Client{
		client: pb.NewNodeRPCClient(conn),
		conn:   conn,
		host:   host,
	}, nil
}

func (c *Client) Close() error {
	err := c.conn.Close()
	return errors.WithStack(err)
}

func (c *Client) GetUsername(ctx context.Context) (string, error) {
	resp, err := c.client.GetUsername(ctx, &pb.GetUsernameRequest{})
	if err != nil {
		return "", errors.WithStack(err)
	}
	return resp.Username, nil
}

func (c *Client) SetUsername(ctx context.Context, username string) error {
	_, err := c.client.SetUsername(ctx, &pb.SetUsernameRequest{Username: username})
	return errors.WithStack(err)
}

func (c *Client) EthAddress(ctx context.Context) (string, error) {
	resp, err := c.client.EthAddress(ctx, &pb.EthAddressRequest{})
	if err != nil {
		return "", errors.WithStack(err)
	}
	return resp.Address, nil
}

func (c *Client) InitRepo(ctx context.Context, repoID string, path string, name string, email string) error {
	_, err := c.client.InitRepo(ctx, &pb.InitRepoRequest{
		RepoID: repoID,
		Path:   path,
		Name:   name,
		Email:  email,
	})
	return errors.WithStack(err)
}

func (c *Client) ImportRepo(ctx context.Context, repoRoot string, repoID string) error {
	_, err := c.client.ImportRepo(ctx, &pb.ImportRepoRequest{
		RepoRoot: repoRoot,
		RepoID:   repoID,
	})
	return errors.WithStack(err)
}

type MaybeFetchFromCommitPacket struct {
	PackfileHeader *pb.FetchFromCommitResponse_PackfileHeader
	PackfileData   *pb.FetchFromCommitResponse_PackfileData
	Chunk          *pb.FetchFromCommitResponse_Chunk
	Error          error
}

func (c *Client) FetchFromCommit(ctx context.Context, repoID string, path string, commit git.Oid, checkoutType nodep2p.CheckoutType) (chan MaybeFetchFromCommitPacket, int64, int64, error) {
	fetchFromCommitClient, err := c.client.FetchFromCommit(ctx, &pb.FetchFromCommitRequest{
		RepoID:       repoID,
		Path:         path,
		Commit:       commit[:],
		CheckoutType: uint64(checkoutType),
	})
	if err != nil {
		return nil, 0, 0, errors.WithStack(err)
	}

	pkt, err := fetchFromCommitClient.Recv()
	if err != nil {
		return nil, 0, 0, errors.WithStack(err)
	}

	header := pkt.GetHeader()
	if header == nil {
		return nil, 0, 0, errors.New("[rpc client] FetchFromCommit: first response packet was not a Header")
	}

	ch := make(chan MaybeFetchFromCommitPacket)
	go func() {
		defer close(ch)
		for {
			x, err := fetchFromCommitClient.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				ch <- MaybeFetchFromCommitPacket{Error: errors.WithStack(err)}
				return
			}

			dataPkt := x.GetPackfileData()
			if dataPkt != nil {
				ch <- MaybeFetchFromCommitPacket{PackfileData: dataPkt}
				continue
			}

			headerPkt := x.GetPackfileHeader()
			if headerPkt != nil {
				ch <- MaybeFetchFromCommitPacket{PackfileHeader: headerPkt}
				continue
			}

			chunkPkt := x.GetChunk()
			if chunkPkt != nil {
				ch <- MaybeFetchFromCommitPacket{Chunk: chunkPkt}
				continue
			}

			ch <- MaybeFetchFromCommitPacket{Error: errors.New("[rpc client] expected PackfileData or PackfileHeader packet, got Header packet")}
			return
		}
	}()

	return ch, header.UncompressedSize, header.TotalChunks, nil
}

type MaybeFetchChunksResponse struct {
	Chunk *pb.FetchChunksResponse
	Error error
}

func (c *Client) FetchChunks(ctx context.Context, repoID string, path string, chunks [][]byte) (<-chan MaybeFetchChunksResponse, error) {
	fetchChunksClient, err := c.client.FetchChunks(ctx, &pb.FetchChunksRequest{
		RepoID: repoID,
		Path:   path,
		Chunks: chunks,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ch := make(chan MaybeFetchChunksResponse)
	go func() {
		defer close(ch)
		for {
			pkt, err := fetchChunksClient.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				ch <- MaybeFetchChunksResponse{Error: errors.WithStack(err)}
				return
			}

			ch <- MaybeFetchChunksResponse{Chunk: pkt}
		}
	}()

	return ch, nil
}

func (c *Client) RegisterRepoID(ctx context.Context, repoID string) error {
	_, err := c.client.RegisterRepoID(ctx, &pb.RegisterRepoIDRequest{RepoID: repoID})
	return errors.WithStack(err)
}

func (c *Client) IsRepoIDRegistered(ctx context.Context, repoID string) (bool, error) {
	resp, err := c.client.IsRepoIDRegistered(ctx, &pb.IsRepoIDRegisteredRequest{RepoID: repoID})
	if err != nil {
		return false, errors.WithStack(err)
	}
	return resp.IsRegistered, nil
}

func (c *Client) TrackLocalRepo(ctx context.Context, repoPath string, forceReload bool) error {
	_, err := c.client.TrackLocalRepo(ctx, &pb.TrackLocalRepoRequest{RepoPath: repoPath, ForceReload: forceReload})
	return errors.WithStack(translateError(err))
}

type MaybeLocalRepo struct {
	nodep2p.LocalRepo
	Error error
}

func (c *Client) GetLocalRepos(ctx context.Context) (chan MaybeLocalRepo, error) {
	cl, err := c.client.GetLocalRepos(ctx, &pb.GetLocalReposRequest{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ch := make(chan MaybeLocalRepo)
	go func() {
		defer close(ch)
		for {
			item, err := cl.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				ch <- MaybeLocalRepo{Error: errors.WithStack(err)}
			} else {
				ch <- MaybeLocalRepo{LocalRepo: nodep2p.LocalRepo{RepoID: item.RepoID, Path: item.Path}}
			}
		}
	}()
	return ch, nil
}

func (c *Client) SetReplicationPolicy(ctx context.Context, repoID string, maxBytes int64) error {
	_, err := c.client.SetReplicationPolicy(ctx, &pb.SetReplicationPolicyRequest{RepoID: repoID, MaxBytes: maxBytes})
	return errors.WithStack(err)
}

func (c *Client) AnnounceRepoContent(ctx context.Context, repoID string) error {
	_, err := c.client.AnnounceRepoContent(ctx, &pb.AnnounceRepoContentRequest{RepoID: repoID})
	return errors.WithStack(err)
}

func (c *Client) GetRemoteRefs(ctx context.Context, repoID string, pageSize uint64, page uint64) (map[string]repo.Ref, uint64, error) {
	resp, err := c.client.GetRemoteRefs(ctx, &pb.GetRemoteRefsRequest{RepoID: repoID, PageSize: pageSize, Page: page})
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}

	refMap := make(map[string]repo.Ref)
	for _, ref := range resp.Refs {
		refMap[ref.RefName] = repo.Ref{RefName: ref.RefName, CommitHash: ref.CommitHash}
	}
	return refMap, resp.Total, nil
}

func (c *Client) IsBehindRemote(ctx context.Context, repoID string, path string) (string, bool, error) {
	resp, err := c.client.IsBehindRemote(ctx, &pb.IsBehindRemoteRequest{RepoID: repoID, Path: path})
	if err != nil {
		return "", false, errors.WithStack(err)
	}
	return resp.RepoID, resp.IsBehindRemote, nil
}

const (
	REF_PAGE_SIZE = 10 // @@TODO: make configurable
)

func (c *Client) GetAllRemoteRefs(ctx context.Context, repoID string) (map[string]repo.Ref, error) {
	var page uint64
	var numRefs uint64
	var err error

	refMap := make(map[string]repo.Ref)

	for {
		var refs map[string]repo.Ref
		refs, numRefs, err = c.GetRemoteRefs(ctx, repoID, REF_PAGE_SIZE, page)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		for _, ref := range refs {
			refMap[ref.RefName] = ref
		}

		if page*REF_PAGE_SIZE >= numRefs {
			break
		}

		page++
	}

	return refMap, nil
}

func (c *Client) GetRepoUsers(ctx context.Context, repoID string, userType nodeeth.UserType, pageSize uint64, page uint64) ([]string, uint64, error) {
	resp, err := c.client.GetRepoUsers(ctx, &pb.GetRepoUsersRequest{RepoID: repoID, PageSize: pageSize, Page: page})
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}
	return resp.Users, resp.Total, nil
}

func (c *Client) PushRepo(ctx context.Context, repoRoot string, branchName string, force bool) (<-chan nodep2p.Progress, error) {
	pushRepoClient, err := c.client.PushRepo(ctx, &pb.PushRepoRequest{RepoRoot: repoRoot, BranchName: branchName, Force: force})
	if err != nil {
		return nil, translateError(err)
	}

	ch := make(chan nodep2p.Progress)

	go func() {
		defer close(ch)
		for {
			pkt, err := pushRepoClient.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				ch <- nodep2p.Progress{Error: translateError(err)}
				return
			}

			ch <- nodep2p.Progress{Current: pkt.Current, Total: pkt.Total, Done: pkt.Done}
		}
	}()

	return ch, nil
}

func (c *Client) RequestReplication(ctx context.Context, repoID string) (<-chan nodep2p.Progress, error) {
	requestReplicationClient, err := c.client.RequestReplication(ctx, &pb.ReplicationRequest{RepoID: repoID})
	if err != nil {
		return nil, err
	}

	ch := make(chan nodep2p.Progress)

	go func() {
		defer close(ch)
		for {
			pkt, err := requestReplicationClient.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				ch <- nodep2p.Progress{Error: err}
				return
			}

			ch <- nodep2p.Progress{Current: pkt.Current, Total: pkt.Total, Done: pkt.Done}
		}
	}()

	return ch, nil
}

func (c *Client) SignMessage(ctx context.Context, message []byte) ([]byte, error) {
	resp, err := c.client.SignMessage(ctx, &pb.SignMessageRequest{Message: message})
	if err != nil {
		return nil, err
	}
	return resp.Signature, nil
}

func (c *Client) GetUserPermissions(ctx context.Context, repoID string, username string) (nodeeth.UserPermissions, error) {
	perms, err := c.client.GetUserPermissions(ctx, &pb.GetUserPermissionsRequest{RepoID: repoID, Username: username})
	if err != nil {
		return nodeeth.UserPermissions{}, errors.WithStack(err)
	}
	return nodeeth.UserPermissions{Puller: perms.Puller, Pusher: perms.Pusher, Admin: perms.Admin}, nil
}

func (c *Client) SetUserPermissions(ctx context.Context, repoID string, username string, perms nodeeth.UserPermissions) error {
	_, err := c.client.SetUserPermissions(ctx, &pb.SetUserPermissionsRequest{RepoID: repoID, Username: username, Puller: perms.Puller, Pusher: perms.Pusher, Admin: perms.Admin})
	return errors.WithStack(err)
}

// @@TODO: refactor using proper gRPC errors
func translateError(err error) error {
	if err == nil {
		return nil
	}

	// @@TODO: refactor using proper gRPC errors
	if strings.Contains(err.Error(), "repo ID not registered") {
		return nodep2p.ErrRepoIDNotRegistered
	} else if strings.Contains(err.Error(), "error looking up axon.repoid") {
		return repo.ErrNoRepoID
	} else if strings.Contains(err.Error(), "not tracking the given repo") {
		return repo.ErrNotTracked
	} else if strings.Contains(err.Error(), "no replicators available") {
		return nodep2p.ErrNoReplicatorsAvailable
	} else if strings.Contains(err.Error(), "every replicator failed to replicate repo") {
		return nodep2p.ErrAllReplicatorsFailed
	} else {
		return err
	}
}

func (c *Client) SetFileChunking(ctx context.Context, repoID string, repoRoot string, filename string, enabled bool) error {
	_, err := c.client.SetFileChunking(ctx, &pb.SetFileChunkingRequest{RepoID: repoID, RepoRoot: repoRoot, Filename: filename, Enabled: enabled})
	return err
}

type MaybeEvent struct {
	AddedRepoEvent  *pb.WatchResponse_AddedRepoEvent
	PulledRepoEvent *pb.WatchResponse_PulledRepoEvent
	UpdatedRefEvent *pb.WatchResponse_UpdatedRefEvent
	Error           error
}

// func (c *Client) Watch(ctx context.Context, settings *swarm.WatcherSettings) (<-chan MaybeEvent, error) {
// 	eventTypes := swarm.EventTypeBitfieldToArray(settings.EventTypes)

// 	watchClient, err := c.client.Watch(ctx, &pb.WatchRequest{
// 		EventTypes:      eventTypes,
// 		UpdatedRefStart: settings.UpdatedRefStart,
// 	})
// 	if err != nil {
// 		return nil, errors.WithStack(err)
// 	}

// 	ch := make(chan MaybeEvent)
// 	go func() {
// 		defer close(ch)
// 		for {
// 			evt, err := watchClient.Recv()
// 			if err == io.EOF {
// 				return
// 			} else if err != nil {
// 				ch <- MaybeEvent{Error: errors.WithStack(err)}
// 				return
// 			}

// 			addedEvt := evt.GetAddedRepoEvent()
// 			if addedEvt != nil {
// 				ch <- MaybeEvent{AddedRepoEvent: addedEvt}
// 				continue
// 			}

// 			pulledEvt := evt.GetPulledRepoEvent()
// 			if pulledEvt != nil {
// 				ch <- MaybeEvent{PulledRepoEvent: pulledEvt}
// 				continue
// 			}

// 			refEvt := evt.GetUpdatedRefEvent()
// 			if refEvt != nil {
// 				ch <- MaybeEvent{UpdatedRefEvent: refEvt}
// 				continue
// 			}

// 			ch <- MaybeEvent{Error: errors.New("[rpc client] unexpected event")}
// 			return
// 		}
// 	}()

// 	return ch, nil
// }
