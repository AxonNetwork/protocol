package noderpc

import (
	"bytes"
	"context"
	"io"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"../../util"
	"../wire"
	"./pb"
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

func (c *Client) SetUsername(ctx context.Context, username string) error {
	_, err := c.client.SetUsername(ctx, &pb.SetUsernameRequest{Username: username})
	return errors.WithStack(err)
}

func (c *Client) GetObject(ctx context.Context, repoID string, objectID []byte) (*util.ObjectReader, error) {
	getObjectClient, err := c.client.GetObject(ctx, &pb.GetObjectRequest{RepoID: repoID, ObjectID: objectID})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// First, read the special header packet containing a wire.ObjectMetadata{} struct
	var meta wire.ObjectMetadata
	{
		packet, err := getObjectClient.Recv()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		headerbuf := bytes.NewBuffer(packet.Data)
		err = wire.ReadStructPacket(headerbuf, &meta)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// Second, receive protobuf packets and pipe their blob payloads into an io.Reader so that
	// consumers can interact with them as a regular byte stream.
	r, w := io.Pipe()
	go func() {
		var err error
		defer func() {
			if err != nil && err != io.EOF {
				w.CloseWithError(err)
			} else {
				w.Close()
			}
		}()

		var packet *pb.GetObjectResponsePacket
		for {
			packet, err = getObjectClient.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				err = errors.WithStack(err)
				return
			}

			_, err = w.Write(packet.Data)
			if err != nil {
				err = errors.WithStack(err)
				return
			}
		}
	}()

	return &util.ObjectReader{
		Reader:     r,
		Closer:     r,
		ObjectLen:  meta.Len,
		ObjectType: meta.Type,
	}, nil
}

func (c *Client) RegisterRepoID(ctx context.Context, repoID string) error {
	_, err := c.client.RegisterRepoID(ctx, &pb.RegisterRepoIDRequest{RepoID: repoID})
	return errors.WithStack(err)
}

func (c *Client) TrackLocalRepo(ctx context.Context, repoPath string) error {
	_, err := c.client.TrackLocalRepo(ctx, &pb.TrackLocalRepoRequest{RepoPath: repoPath})
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

func (c *Client) GetRefs(ctx context.Context, repoID string, page uint64) (map[string]wire.Ref, uint64, error) {
	resp, err := c.client.GetRefs(ctx, &pb.GetRefsRequest{RepoID: repoID, Page: page})
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}

	refMap := make(map[string]wire.Ref)
	for _, ref := range resp.Ref {
		refMap[ref.RefName] = wire.Ref{RefName: ref.RefName, CommitHash: ref.CommitHash}
	}
	return refMap, resp.NumRefs, nil
}

const (
	REF_PAGE_SIZE = 10 // @@TODO: make configurable
)

func (c *Client) GetAllRefs(ctx context.Context, repoID string) (map[string]wire.Ref, error) {
	var page uint64
	var numRefs uint64
	var err error

	refMap := make(map[string]wire.Ref)

	for {
		var refs map[string]wire.Ref
		refs, numRefs, err = c.GetRefs(ctx, repoID, page)
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

func (c *Client) RequestReplication(ctx context.Context, repoID string) error {
	_, err := c.client.RequestReplication(ctx, &pb.ReplicationRequest{RepoID: repoID})
	return errors.WithStack(err)
}

func (c *Client) Close() error {
	err := c.conn.Close()
	return errors.WithStack(err)
}
