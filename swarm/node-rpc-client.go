package swarm

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/pkg/errors"
)

type RPCClient struct {
	conn          net.Conn
	network, addr string
}

func NewRPCClient(network, addr string) (*RPCClient, error) {
	client := &RPCClient{
		network: network,
		addr:    addr,
	}
	return client, nil
}

func (c *RPCClient) GetObject(repoID string, objectID []byte) (ObjectReader, error) {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return nil, err
	}

	// Write the message type
	err = writeUint64(conn, uint64(MessageType_GetObject))
	if err != nil {
		return nil, err
	}

	// Write the request packet
	err = writeStructPacket(conn, &GetObjectRequest{RepoID: repoID, ObjectID: objectID})
	if err != nil {
		return nil, err
	}

	// Read the response packet (i.e., the header for the subsequent object stream)
	resp := GetObjectResponse{}
	err = readStructPacket(conn, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.HasObject {
		return nil, errors.Wrapf(ErrObjectNotFound, "%v:%v", repoID, hex.EncodeToString(objectID))
	}

	reader := &objectReader{
		Reader:     &io.LimitedReader{conn, resp.ObjectLen},
		Closer:     conn,
		objectType: resp.ObjectType,
		objectLen:  resp.ObjectLen,
	}

	return reader, nil
}

func (c *RPCClient) AddRepo(repoPath string) error {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return err
	}

	// Write the message type
	err = writeUint64(conn, uint64(MessageType_AddRepo))
	if err != nil {
		return err
	}

	// Write the request packet
	err = writeStructPacket(conn, &AddRepoRequest{RepoPath: repoPath})
	if err != nil {
		return err
	}

	// Read the response packet (i.e., the header for the subsequent object stream)
	resp := AddRepoResponse{}
	err = readStructPacket(conn, &resp)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("Repo could not be added")
	}

	log.Printf("[rpc stream] response %+v", resp)

	return nil
}

func (c *RPCClient) GetRefs(repoID string) (map[string]string, error) {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return nil, err
	}

	// Write the message type
	err = writeUint64(conn, uint64(MessageType_GetRefs))
	if err != nil {
		return nil, err
	}

	// Write the request packet
	err = writeStructPacket(conn, &GetRefsRequest{
		RepoID: repoID,
	})
	if err != nil {
		return nil, err
	}

	// Read the response packet (i.e., the header for the subsequent object stream)
	resp := GetRefsResponse{}
	err = readStructPacket(conn, &resp)
	if err != nil {
		return nil, err
	}

	log.Printf("[rpc stream] response %+v", resp)
	refs := map[string]string{}
	err = json.Unmarshal(resp.Refs, &refs)
	if err != nil {
		return nil, err
	}

	return refs, nil
}

func (c *RPCClient) AddRef(repoID string, target string, name string) (map[string]string, error) {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return nil, err
	}

	// Write the message type
	err = writeUint64(conn, uint64(MessageType_AddRef))
	if err != nil {
		return nil, err
	}

	// Write the request packet
	err = writeStructPacket(conn, &AddRefRequest{
		RepoID: repoID,
		Target: target,
		Name:   name,
	})
	if err != nil {
		return nil, err
	}

	// Read the response packet (i.e., the header for the subsequent object stream)
	resp := AddRefResponse{}
	err = readStructPacket(conn, &resp)
	if err != nil {
		return nil, err
	}

	log.Printf("[rpc stream] response %+v", resp)
	refs := map[string]string{}
	err = json.Unmarshal(resp.Refs, &refs)
	if err != nil {
		return nil, err
	}

	return refs, nil
}
