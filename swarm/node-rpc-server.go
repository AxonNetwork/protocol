package swarm

import (
	"context"
	"io"
	"net"
	"time"

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

				log.Errorf("[rpc] %v", err)
			} else {
				log.Printf("[rpc] opening new stream")
				go n.rpcStreamHandler(conn)
			}

		}
	}()

	return nil
}

func (n *Node) rpcStreamHandler(stream io.ReadWriteCloser) {
	defer stream.Close()

	logErr := func(err error) {
		if err != nil {
			log.Errorf("[rpc] %v", err)
		}
	}

	msgType, err := readUint64(stream)
	if err != nil {
		panic(err)
	}

	switch MessageType(msgType) {

	case MessageType_SetUsername:
		req := SetUsernameRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)

		tx, err := n.Eth.EnsureUsername(ctx, req.Username)
		if err != nil {
			writeStructPacket(stream, &SetUsernameResponse{Error: err.Error()})
			return
		}

		if tx != nil {
			resp := <-tx.Await(ctx)
			if resp.Err != nil {
				writeStructPacket(stream, &SetUsernameResponse{Error: resp.Err.Error()})
				return
			} else if resp.Receipt.Status != 1 {
				writeStructPacket(stream, &SetUsernameResponse{Error: "transaction failed"})
				return
			}
		}

		err = writeStructPacket(stream, &SetUsernameResponse{Error: ""})

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
		log.Printf("[rpc] CreateRepo")
		req := CreateRepoRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc] create repo: %s", req.RepoID)

		tx, err := n.Eth.EnsureRepo(context.Background(), req.RepoID)
		if err != nil {
			panic(err)
		}

		if tx != nil {
			log.Printf("[rpc] create repo tx sent: %s", tx.Hash().Hex())
			txResult := <-tx.Await(context.Background())
			if txResult.Err != nil {
				panic(err)
			}
			log.Printf("[rpc] create repo tx resolved: %s", tx.Hash().Hex())
		}

		err = writeStructPacket(stream, &CreateRepoResponse{OK: true})
		if err != nil {
			panic(err)
		}

	case MessageType_AddRepo:
		log.Printf("[rpc] AddRepo")
		req := AddRepoRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc] add repo: %s", req.RepoPath)

		_, err = n.RepoManager.AddRepo(req.RepoPath)
		if err != nil {
			panic(err)
		}

		err = writeStructPacket(stream, &AddRepoResponse{OK: true})
		if err != nil {
			panic(err)
		}

	case MessageType_SetReplicationPolicy:
		log.Printf("[rpc] SetReplicationPolicy")
		req := SetReplicationPolicyRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			logErr(err)
			return
		}

		log.Printf("[rpc] SetReplicationPolicy(%s, %v)", req.RepoID, req.ShouldReplicate)

		err = n.SetReplicationPolicy(req.RepoID, req.ShouldReplicate)
		if err != nil {
			err = writeStructPacket(stream, &SetReplicationPolicyResponse{Error: err.Error()})
			logErr(err)
			return
		}

		err = writeStructPacket(stream, &SetReplicationPolicyResponse{Error: ""})
		logErr(err)

	case MessageType_AnnounceRepoContent:
		log.Printf("[rpc] AnnounceRepoContent")
		req := AnnounceRepoContentRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc] announce repo content: %s", req.RepoID)

		err = n.AnnounceRepoContent(context.Background(), req.RepoID)
		if err != nil {
			panic(err)
		}

		err = writeStructPacket(stream, &AnnounceRepoContentResponse{OK: true})
		if err != nil {
			panic(err)
		}

	case MessageType_GetRefs:
		log.Printf("[rpc] GetRefs")
		req := GetRefsRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc] get refs from %s", req.RepoID)

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
		log.Printf("[rpc] UpdateRef")
		req := UpdateRefRequest{}
		err := readStructPacket(stream, &req)
		if err != nil {
			panic(err)
		}

		log.Printf("[rpc] add ref to %s: %s %s", req.RepoID, req.RefName, req.Commit)
		tx, err := n.Eth.UpdateRef(context.Background(), req.RepoID, req.RefName, req.Commit)
		if err != nil {
			panic(err)
		}
		log.Printf("[rpc] update ref tx sent: %s", tx.Hash().Hex())
		txResult := <-tx.Await(context.Background())
		if txResult.Err != nil {
			panic(err)
		}
		log.Printf("[rpc] update ref tx resolved: %s", tx.Hash().Hex())

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
		errStr := ""
		if err != nil {
			log.Errorf("[rpc] MessageType_Pull error: %v", err)
			errStr = err.Error()
		}

		err = writeStructPacket(stream, &PullResponse{Error: errStr})
		if err != nil {
			panic(err)
		}

	default:
		log.Errorf("Node.rpcStreamHandler: bad message type from peer (%v)", msgType)
	}
}
