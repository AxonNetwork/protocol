package swarm

import (
	"context"
	"io"
	"net"

	log "github.com/sirupsen/logrus"
)

func (n *Node) initRPC(network, addr string) error {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Errorf("[rpc stream] %v", err)
			} else {
				log.Printf("[rpc stream] opening new stream")
				go n.rpcStreamHandler(conn)
			}
		}
	}()

	return nil
}

func (n *Node) rpcStreamHandler(stream io.ReadWriteCloser) {
	defer stream.Close()

	msgType, err := readUint64(stream)
	if err != nil {
		panic(err)
	}

	log.Printf("[rpc stream] msgType = %v", msgType)

	switch MessageType(msgType) {
	case MessageType_GetObject:
		log.Printf("[rpc stream] GetObject")
		req := GetObjectRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc stream] %+v", req)

		objectReader, err := n.GetObjectReader(context.Background(), req.RepoID, req.ObjectID)
		// @@TODO: maybe don't assume err == 404
		if err != nil {
			log.Printf("[rpc stream] don't have object: %v", err)
			err = writeStructPacket(stream, &GetObjectResponse{HasObject: false})
			if err != nil {
				panic(err)
			}
			return
		}

		log.Printf("[rpc stream] do have object")
		err = writeStructPacket(stream, &GetObjectResponse{
			HasObject:  true,
			ObjectType: objectReader.Type(),
			ObjectLen:  objectReader.Len(),
		})
		if err != nil {
			panic(err)
		}

		// Write the object
		written, err := io.Copy(stream, objectReader)
		if err != nil {
			panic(err)
		} else if written < objectReader.Len() {
			panic("written < objectLen")
		}

	default:
		log.Errorf("Node.rpcStreamHandler: bad message type from peer (%v)", msgType)
	}
}

// type AddRepoInput struct {
//  RepoPath string
// }

// type AddRepoOutput struct{}

// func (nr *NodeRPC) AddRepo(in *AddRepoInput, out *AddRepoOutput) error {
//  err := nr.node.RepoManager.AddRepo(in.RepoPath)
//  return err
// }

// type ListHelperInput struct {
//  RepoID   string
//  ObjectID []byte
// }

// type ListHelperOutput struct {
//  Stream inet.Stream
// }

// func (nr *NodeRPC) ListHelper(in *ListHelperInput, out *ListHelperOutput) error {
//  stream, err := nr.node.ListHelper(in.RepoID, in.ObjectID)
//  out.Stream = stream
//  return err
// }
