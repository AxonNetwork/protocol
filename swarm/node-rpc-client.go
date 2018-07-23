package swarm

import (
	"encoding/hex"
	"io"
	"net"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
	log.Printf("RPCClient.GetObject(%v, %v)", repoID, hex.EncodeToString(objectID))
	err = writeUint64(conn, uint64(RPCMessageType_GetObject))
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

	log.Printf("[rpc stream] response %+v", resp)

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
