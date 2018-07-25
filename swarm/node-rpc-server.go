package swarm

import (
	"context"
	"encoding/json"
	"io"
	"net"

	log "github.com/sirupsen/logrus"
)

func (n *Node) initRPC(network, addr string) error {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}

	n.rpcListener = listener

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// this is the awkward way that Go requires us to detect that listener.Close() has been called
				select {
				case <-n.chShutdown:
					return
				default:
				}

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

	switch MessageType(msgType) {

	case MessageType_GetObject:
		req := GetObjectRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		objectReader, err := n.GetObjectReader(context.Background(), req.RepoID, req.ObjectID)
		// @@TODO: maybe don't assume err == 404
		if err != nil {
			err = writeStructPacket(stream, &GetObjectResponse{HasObject: false})
			if err != nil {
				panic(err)
			}
			return
		}

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

	case MessageType_AddRepo:
		log.Printf("[rpc stream] AddRepo")
		req := AddRepoRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc stream] add repo: %s", req.RepoPath)

		success := true
		err = n.RepoManager.AddRepo(req.RepoPath)
		if err != nil {
			log.Printf("[rpc stream] couldn't add repo")
			success = false
		}

		err = writeStructPacket(stream, &AddRepoResponse{
			Success: success,
		})
		if err != nil {
			panic(err)
		}

	case MessageType_GetRefs:
		log.Printf("[rpc stream] GetRefs")
		req := GetRefsRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc stream] get refs from %s", req.RepoID)
		refs, err := n.GetRefs(context.Background(), req.RepoID)

		if err != nil {
			log.Printf("[rpc stream] couldn't find refs")
			refs = map[string]string{}
		}
		refsBin, err := json.Marshal(refs)
		if err != nil {
			panic(err)
		}
		err = writeStructPacket(stream, &GetRefsResponse{
			Refs: refsBin,
		})
		if err != nil {
			panic(err)
		}

	case MessageType_AddRef:
		log.Printf("[rpc stream] AddRef")
		req := AddRefRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc stream] add ref to %s: %s %s", req.RepoID, req.Target, req.Name)
		refs, err := n.AddRef(context.Background(), req.RepoID, req.Target, req.Name)

		if err != nil {
			log.Printf("[rpc stream] couldn't add ref")
			refs = map[string]string{}
		}
		refsBin, err := json.Marshal(refs)
		if err != nil {
			panic(err)
		}
		err = writeStructPacket(stream, &AddRefResponse{
			Refs: refsBin,
		})
		if err != nil {
			panic(err)
		}

	case MessageType_Pull:
		req := PullRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		err = n.requestReplication(context.Background(), req.RepoID)
		if err != nil {
			panic(err)
		}

		err = writeStructPacket(stream, &PullResponse{OK: true})
		if err != nil {
			panic(err)
		}

	default:
		log.Errorf("Node.rpcStreamHandler: bad message type from peer (%v)", msgType)
	}
}
