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

	n.rpc = listener

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
			switch err {
			case ErrUnauthorized:
				err = writeStructPacket(stream, &GetObjectResponse{Unauthorized: true})
			case ErrObjectNotFound:
				err = writeStructPacket(stream, &GetObjectResponse{HasObject: false})
			default:
				err = writeStructPacket(stream, &GetObjectResponse{HasObject: false})
			}
			if err != nil {
				panic(err)
			}
			return
		}

		err = writeStructPacket(stream, &GetObjectResponse{
			Unauthorized: false,
			HasObject:    true,
			ObjectType:   objectReader.Type(),
			ObjectLen:    objectReader.Len(),
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

	case MessageType_CreateRepo:
		log.Printf("[rpc stream] CreateRepo")
		req := CreateRepoRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc stream] create repo: %s", req.RepoID)

		tx, err := n.Eth.CreateRepository(context.Background(), req.RepoID)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc stream] create repo tx sent: %s", tx.Hash().Hex())
		ch := n.Eth.WatchTX(context.Background(), tx)
		txResult := <-ch
		if txResult.Err != nil {
			panic(err)
		}
		log.Printf("[rpc stream] create repo tx resolved: %s", tx.Hash().Hex())

		err = writeStructPacket(stream, &CreateRepoResponse{OK: true})
		if err != nil {
			panic(err)
		}

	case MessageType_AddRepo:
		log.Printf("[rpc stream] AddRepo")
		req := AddRepoRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc stream] add repo: %s", req.RepoPath)

		_, err = n.RepoManager.AddRepo(req.RepoPath)
		if err != nil {
			panic(err)
		}

		err = writeStructPacket(stream, &AddRepoResponse{OK: true})
		if err != nil {
			panic(err)
		}

	case MessageType_AnnounceRepoContent:
		log.Printf("[rpc stream] AnnounceRepoContent")
		req := AnnounceRepoContentRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc stream] announce repo content: %s", req.RepoID)

		err = n.AnnounceRepoContent(context.Background(), req.RepoID)
		if err != nil {
			panic(err)
		}

		err = writeStructPacket(stream, &AnnounceRepoContentResponse{OK: true})
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

		numRefs, err := n.Eth.GetNumRefs(context.Background(), req.RepoID)
		if err != nil {
			panic(err)
		}

		refMap, err := n.Eth.GetRefs(context.Background(), req.RepoID, req.Page)
		if err != nil {
			panic(err)
		}

		refs := make([]Ref, len(refMap))
		i := 0
		for _, ref := range refMap {
			refs[i] = ref
			i++
		}

		err = writeStructPacket(stream, &GetRefsResponse{Refs: refs, NumRefs: numRefs})
		if err != nil {
			panic(err)
		}

	case MessageType_UpdateRef:
		log.Printf("[rpc stream] UpdateRef")
		req := UpdateRefRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc stream] add ref to %s: %s %s", req.RepoID, req.RefName, req.Commit)
		tx, err := n.Eth.UpdateRef(context.Background(), req.RepoID, req.RefName, req.Commit)
		if err != nil {
			panic(err)
		}
		log.Printf("[rpc stream] update ref tx sent: %s", tx.Hash().Hex())
		ch := n.Eth.WatchTX(context.Background(), tx)
		txResult := <-ch
		if txResult.Err != nil {
			panic(err)
		}
		log.Printf("[rpc stream] update ref tx resolved; %s", tx.Hash().Hex())

		err = writeStructPacket(stream, &UpdateRefResponse{OK: true})
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
