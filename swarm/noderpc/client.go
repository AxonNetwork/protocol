package noderpc

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodeeth"
	"github.com/Conscience/protocol/swarm/noderpc/pb"
	"github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
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

func (c *Client) FetchFromCommit(ctx context.Context, repoID string, path string, commit string) (io.Reader, error) {
	fetchFromCommitClient, err := c.client.FetchFromCommit(ctx, &pb.FetchFromCommitRequest{
		RepoID: repoID,
		Path:   path,
		Commit: commit,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	r, w := io.Pipe()
	go func() {
		var err error
		defer func() {
			if err != nil && err != io.EOF {
				log.Println("CLOSING WITH ERROR")
				w.CloseWithError(err)
			} else {
				log.Println("CLOSING")
				w.Close()
			}
		}()

		files := map[gitplumbing.Hash]*bytes.Buffer{}
		for {
			packet, err := fetchFromCommitClient.Recv()
			if err == io.EOF {
				return
			} else if err != nil {
				err = errors.WithStack(err)
				return
			}
			hashB := [20]byte{}
			copy(hashB[:], packet.ObjHash[:])
			hash := gitplumbing.Hash(hashB)

			files[hash].Write(packet.Data)
			if uint64(files[hash].Len()) == packet.ObjLen {
				wire.WriteStructPacket(w, &wire.ObjectHeader{
					Hash: hash,
					Type: gitplumbing.ObjectType(packet.ObjType),
					Len:  packet.ObjLen,
				})
				n, err := io.Copy(w, files[hash])
				if err != nil {
					err = errors.WithStack(err)
					return
				} else if uint64(n) != packet.ObjLen {
					err = fmt.Errorf("RPC Client: Could not write entire packet")
					return
				}
			}
		}
	}()
	return io.Reader(r), nil
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

func (c *Client) GetLocalRefs(ctx context.Context, repoID string, path string) (map[string]wire.Ref, string, error) {
	resp, err := c.client.GetLocalRefs(ctx, &pb.GetLocalRefsRequest{RepoID: repoID, Path: path})
	if err != nil {
		return nil, "", err
	}

	refs := make(map[string]wire.Ref, len(resp.Refs))
	for _, ref := range resp.Refs {
		refs[ref.RefName] = wire.Ref{RefName: ref.RefName, CommitHash: ref.CommitHash}
	}
	return refs, resp.Path, nil
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

func (c *Client) RequestReplication(ctx context.Context, repoID string) error {
	_, err := c.client.RequestReplication(ctx, &pb.ReplicationRequest{RepoID: repoID})
	return errors.WithStack(err)
}

func (c *Client) RepoHasObject(ctx context.Context, repoID string, objectID []byte) (bool, error) {
	resp, err := c.client.RepoHasObject(ctx, &pb.RepoHasObjectRequest{RepoID: repoID, ObjectID: objectID})
	if err != nil {
		return false, err
	}
	return resp.HasObject, nil
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
