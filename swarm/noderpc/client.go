package noderpc

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/Conscience/protocol/swarm/nodeeth"
	"github.com/Conscience/protocol/swarm/noderpc/pb"
	"github.com/Conscience/protocol/swarm/wire"
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

func (c *Client) SetUsername(ctx context.Context, username string) error {
	_, err := c.client.SetUsername(ctx, &pb.SetUsernameRequest{Username: username})
	return errors.WithStack(err)
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

type MaybeFetchFromCommitPacket struct {
	PackfileHeader *pb.FetchFromCommitResponse_PackfileHeader
	PackfileData   *pb.FetchFromCommitResponse_PackfileData
	Error          error
}

func (c *Client) FetchFromCommit(ctx context.Context, repoID string, path string, commit gitplumbing.Hash) (chan MaybeFetchFromCommitPacket, int64, error) {
	fetchFromCommitClient, err := c.client.FetchFromCommit(ctx, &pb.FetchFromCommitRequest{
		RepoID: repoID,
		Path:   path,
		Commit: commit[:],
	})
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}

	pkt, err := fetchFromCommitClient.Recv()
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}

	header := pkt.GetHeader()
	if header == nil {
		return nil, 0, errors.New("[rpc client] FetchFromCommit: first response packet was not a Header")
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

			ch <- MaybeFetchFromCommitPacket{Error: errors.New("[rpc client] expected PackfileData or PackfileHeader packet, got Header packet")}
			return
		}
	}()

	return ch, header.UncompressedSize, nil
}

func (c *Client) RegisterRepoID(ctx context.Context, repoID string) error {
	_, err := c.client.RegisterRepoID(ctx, &pb.RegisterRepoIDRequest{RepoID: repoID})
	return errors.WithStack(err)
}

func (c *Client) TrackLocalRepo(ctx context.Context, repoPath string, forceReload bool) error {
	_, err := c.client.TrackLocalRepo(ctx, &pb.TrackLocalRepoRequest{RepoPath: repoPath, ForceReload: forceReload})
	return errors.WithStack(err)
}

type MaybeLocalRepo struct {
	wire.LocalRepo
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
				ch <- MaybeLocalRepo{LocalRepo: wire.LocalRepo{RepoID: item.RepoID, Path: item.Path}}
			}
		}
	}()
	return ch, nil
}

func (c *Client) SetReplicationPolicy(ctx context.Context, repoID string, shouldReplicate bool) error {
	_, err := c.client.SetReplicationPolicy(ctx, &pb.SetReplicationPolicyRequest{RepoID: repoID, ShouldReplicate: shouldReplicate})
	return errors.WithStack(err)
}

func (c *Client) AnnounceRepoContent(ctx context.Context, repoID string) error {
	_, err := c.client.AnnounceRepoContent(ctx, &pb.AnnounceRepoContentRequest{RepoID: repoID})
	return errors.WithStack(err)
}

func (c *Client) GetRemoteRefs(ctx context.Context, repoID string, pageSize uint64, page uint64) (map[string]wire.Ref, uint64, error) {
	resp, err := c.client.GetRemoteRefs(ctx, &pb.GetRemoteRefsRequest{RepoID: repoID, PageSize: pageSize, Page: page})
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}

	refMap := make(map[string]wire.Ref)
	for _, ref := range resp.Refs {
		refMap[ref.RefName] = wire.Ref{RefName: ref.RefName, CommitHash: ref.CommitHash}
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

func (c *Client) GetAllRemoteRefs(ctx context.Context, repoID string) (map[string]wire.Ref, error) {
	var page uint64
	var numRefs uint64
	var err error

	refMap := make(map[string]wire.Ref)

	for {
		var refs map[string]wire.Ref
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

func (c *Client) UpdateRef(ctx context.Context, repoID string, refName string, commitHash string) error {
	_, err := c.client.UpdateRef(ctx, &pb.UpdateRefRequest{RepoID: repoID, RefName: refName, CommitHash: commitHash})
	return errors.WithStack(err)
}

func (c *Client) GetRepoUsers(ctx context.Context, repoID string, userType nodeeth.UserType, pageSize uint64, page uint64) ([]string, uint64, error) {
	resp, err := c.client.GetRepoUsers(ctx, &pb.GetRepoUsersRequest{RepoID: repoID, PageSize: pageSize, Page: page})
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}
	return resp.Users, resp.Total, nil
}

type MaybeReplProgress struct {
	Percent int32
	Error   error
}

func (c *Client) RequestReplication(ctx context.Context, repoID string) chan MaybeReplProgress {
	ch := make(chan MaybeReplProgress)
	requestReplicationClient, err := c.client.RequestReplication(ctx, &pb.ReplicationRequest{RepoID: repoID})
	if err != nil {
		go func() {
			defer close(ch)
			ch <- MaybeReplProgress{Error: err}
		}()
		return ch
	}
	go func() {
		defer close(ch)
		for {
			progress, err := requestReplicationClient.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				ch <- MaybeReplProgress{Error: err}
				return
			}
			ch <- MaybeReplProgress{Percent: progress.Percent}
		}
	}()
	return ch
}

func (c *Client) SignMessage(ctx context.Context, message []byte) ([]byte, error) {
	resp, err := c.client.SignMessage(ctx, &pb.SignMessageRequest{Message: message})
	if err != nil {
		return nil, err
	}
	return resp.Signature, nil
}

func (c *Client) SetUserPermissions(ctx context.Context, repoID string, username string, perms nodeeth.UserPermissions) error {
	_, err := c.client.SetUserPermissions(ctx, &pb.SetUserPermissionsRequest{RepoID: repoID, Username: username, Puller: perms.Puller, Pusher: perms.Pusher, Admin: perms.Admin})
	return errors.WithStack(err)
}
