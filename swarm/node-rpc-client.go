package swarm

import (
	"encoding/hex"
	"io"
	"net"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
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

// func (c *RPCClient) Close() error {
//  return conn.Close()
// }

// func (c *RPCClient) sendMessageType(msgType RPCMessageType) error {
//  log.Printf("RPCClient.sendMessageType(%v)", msgType)
//  return writeUint64(conn, uint64(msgType))
// }

type objectReader struct {
	conn net.Conn
	io.Reader
}

func (or objectReader) Close() error {
	return or.conn.Close()
}

func (c *RPCClient) GetObject(repoID string, objectID []byte) (io.ReadCloser, gitplumbing.ObjectType, int64, error) {
	conn, err := net.Dial(c.network, c.addr)
	if err != nil {
		return nil, 0, 0, err
	}

	// Write the message type
	log.Printf("RPCClient.GetObject(%v, %v)", repoID, hex.EncodeToString(objectID))
	err = writeUint64(conn, uint64(RPCMessageType_GetObject))
	if err != nil {
		return nil, 0, 0, err
	}

	// Write the request packet
	err = writeStructPacket(conn, &GetObjectRequest{RepoID: repoID, ObjectID: objectID})
	if err != nil {
		return nil, 0, 0, err
	}

	// Read the response packet (i.e., the header for the subsequent object stream)
	resp := GetObjectResponse{}
	err = readStructPacket(conn, &resp)
	if err != nil {
		return nil, 0, 0, err
	}

	log.Printf("[rpc stream] response %+v", resp)

	if !resp.HasObject {
		return nil, 0, 0, errors.Wrapf(ErrObjectNotFound, "%v:%v", repoID, hex.EncodeToString(objectID))
	}

	reader := &objectReader{
		conn:   conn,
		Reader: &io.LimitedReader{conn, resp.ObjectLen},
	}

	return reader, resp.ObjectType, resp.ObjectLen, nil
}
