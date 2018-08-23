package noderpc

import (
	"encoding/hex"
	"fmt"
	"io"
	"net"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"../../util"
	. "../wire"
)

// @@TODO: make all Client methods take a context
type Client struct {
	network, addr string
}

func NewClient(network, addr string) (*Client, error) {
	client := &Client{
		network: network,
		addr:    addr,
	}
	return client, nil
}

func (c *Client) writeMessageType(conn net.Conn, typ MessageType) error {
	return WriteUint64(conn, uint64(typ))
}

func (c *Client) SetUsername(username string) error {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = c.writeMessageType(conn, MessageType_SetUsername)
	if err != nil {
		return err
	}

	// Write the request packet
	err = WriteStructPacket(conn, &SetUsernameRequest{Username: username})
	if err != nil {
		return err
	}

	// Read the response packet (i.e., the header for the subsequent object stream)
	resp := SetUsernameResponse{}
	err = ReadStructPacket(conn, &resp)
	if err != nil {
		return err
	} else if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

func (c *Client) GetObject(repoID string, objectID []byte) (*util.ObjectReader, error) {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return nil, err
	}
	// It is the caller's responsibility to `.Close()` the conn.

	err = c.writeMessageType(conn, MessageType_GetObject)
	if err != nil {
		return nil, err
	}

	// Write the request packet
	err = WriteStructPacket(conn, &GetObjectRequest{RepoID: repoID, ObjectID: objectID})
	if err != nil {
		return nil, err
	}

	// Read the response packet (i.e., the header for the subsequent object stream)
	resp := GetObjectResponse{}
	err = ReadStructPacket(conn, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Unauthorized {
		return nil, errors.Wrapf(ErrUnauthorized, "%v:%v", repoID, hex.EncodeToString(objectID))
	}

	if !resp.HasObject {
		return nil, errors.Wrapf(ErrObjectNotFound, "%v:%v", repoID, hex.EncodeToString(objectID))
	}

	reader := &util.ObjectReader{
		Reader:     &io.LimitedReader{conn, resp.ObjectLen},
		Closer:     conn,
		ObjectType: resp.ObjectType,
		ObjectLen:  resp.ObjectLen,
	}

	return reader, nil
}

func (c *Client) RegisterRepoID(repoID string) error {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = c.writeMessageType(conn, MessageType_RegisterRepoID)
	if err != nil {
		return err
	}

	// Write the request packet
	err = WriteStructPacket(conn, &RegisterRepoIDRequest{RepoID: repoID})
	if err != nil {
		return err
	}

	log.Printf("Create Repo TX Sent")

	// Read the response packet (i.e., the header for the subsequent object stream)
	resp := RegisterRepoIDResponse{}
	err = ReadStructPacket(conn, &resp)
	if err != nil {
		return err
	} else if !resp.OK {
		return errors.New("repo could not be added")
	}

	return nil
}

func (c *Client) AddRepo(repoPath string) error {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = c.writeMessageType(conn, MessageType_AddRepo)
	if err != nil {
		return err
	}

	// Write the request packet
	err = WriteStructPacket(conn, &AddRepoRequest{RepoPath: repoPath})
	if err != nil {
		return err
	}

	// Read the response packet
	resp := AddRepoResponse{}
	err = ReadStructPacket(conn, &resp)
	if err != nil {
		return err
	} else if !resp.OK {
		return errors.New("repo could not be added")
	}
	return nil
}

func (c *Client) GetRepos() ([]Repo, error) {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	err = c.writeMessageType(conn, MessageType_GetRepos)
	if err != nil {
		return nil, err
	}

	// Read the response packet
	resp := GetReposResponse{}
	err = ReadStructPacket(conn, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Repos, nil
}

func (c *Client) SetReplicationPolicy(repoID string, shouldReplicate bool) error {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = c.writeMessageType(conn, MessageType_SetReplicationPolicy)
	if err != nil {
		return err
	}

	err = WriteStructPacket(conn, &SetReplicationPolicyRequest{RepoID: repoID, ShouldReplicate: shouldReplicate})
	if err != nil {
		return err
	}

	resp := SetReplicationPolicyResponse{}
	err = ReadStructPacket(conn, &resp)
	if err != nil {
		return err
	} else if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

func (c *Client) AnnounceRepoContent(repoID string) error {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = c.writeMessageType(conn, MessageType_AnnounceRepoContent)
	if err != nil {
		return err
	}

	// Write the request packet
	err = WriteStructPacket(conn, &AnnounceRepoContentRequest{RepoID: repoID})
	if err != nil {
		return err
	}

	// Read the response packet (i.e., the header for the subsequent object stream)
	resp := AnnounceRepoContentResponse{}
	err = ReadStructPacket(conn, &resp)
	if err != nil {
		return err
	} else if !resp.OK {
		return fmt.Errorf("repo could not be added")
	}

	return nil
}

const (
	REF_PAGE_SIZE = 10 // @@TODO: make configurable
)

func (c *Client) GetAllRefs(repoID string) (map[string]Ref, error) {
	var page int64
	var numRefs int64
	var err error

	refMap := make(map[string]Ref)

	for {
		var refs map[string]Ref
		refs, numRefs, err = c.GetRefs(repoID, page)
		if err != nil {
			return nil, err
		}

		for _, ref := range refs {
			refMap[ref.Name] = ref
		}

		if int64(page*REF_PAGE_SIZE) >= numRefs {
			break
		}

		page++
	}

	return refMap, nil
}

func (c *Client) GetRefs(repoID string, page int64) (map[string]Ref, int64, error) {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return nil, 0, err
	}
	defer conn.Close()

	err = c.writeMessageType(conn, MessageType_GetRefs)
	if err != nil {
		return nil, 0, err
	}

	// Write the request packet
	err = WriteStructPacket(conn, &GetRefsRequest{RepoID: repoID, Page: page})
	if err != nil {
		return nil, 0, err
	}

	// Read the response packet (i.e., the header for the subsequent object stream)
	resp := GetRefsResponse{}
	err = ReadStructPacket(conn, &resp)
	if err != nil {
		return nil, 0, err
	}

	refs := map[string]Ref{}
	for _, ref := range resp.Refs {
		refs[ref.Name] = ref
	}

	return refs, resp.NumRefs, nil
}

func (c *Client) UpdateRef(repoID string, refName string, commitHash string) error {
	if len(commitHash) != 40 {
		return errors.New("commit hash is not 40 hex characters")
	}

	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = c.writeMessageType(conn, MessageType_UpdateRef)
	if err != nil {
		return err
	}

	// Write the request packet
	err = WriteStructPacket(conn, &UpdateRefRequest{
		RepoID:  repoID,
		RefName: refName,
		Commit:  commitHash,
	})
	if err != nil {
		return err
	}

	log.Printf("Update Ref TX Sent")

	// Read the response packet
	resp := UpdateRefResponse{}
	err = ReadStructPacket(conn, &resp)
	if err != nil {
		return err
	} else if !resp.OK {
		return errors.New("UpdateRefResponse.OK is false")
	}
	return nil
}

func (c *Client) RequestReplication(repoID string) error {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = c.writeMessageType(conn, MessageType_Replicate)
	if err != nil {
		return err
	}

	// Write the request packet
	err = WriteStructPacket(conn, &ReplicationRequest{RepoID: repoID})
	if err != nil {
		return err
	}

	// Read the response packet (i.e., the header for the subsequent object stream)
	resp := ReplicationResponse{}
	err = ReadStructPacket(conn, &resp)
	if err != nil {
		return err
	}

	if resp.Error == "" {
		log.Printf("[rpc stream] RequestReplication: ok")
		return nil
	} else {
		log.Errorf("[rpc stream] RequestReplication: error = %v", resp.Error)
		return errors.New(resp.Error)
	}
}
