package strategy

import (
	"context"
	"io"
	"time"

	netp2p "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"

	"github.com/Conscience/protocol/log"
	. "github.com/Conscience/protocol/swarm/wire"
)

type NaiveServer struct {
	node INode
}

func NewNaiveServer(node INode) *NaiveServer {
	return &NaiveServer{node}
}

// Handles incoming requests for objects.
func (ns *NaiveServer) HandleObjectRequest(stream netp2p.Stream) {
	defer stream.Close()

	// Read the request packet
	req := GetObjectRequestSigned{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
		return
	}

	log.Debugf("[p2p object server] peer requested %v %0x", req.RepoID, req.ObjectID)

	addr, err := ns.node.AddrFromSignedHash(req.ObjectID, req.Signature)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
		return
	}

	// @@TODO: give context a timeout and make it configurable
	ctx := context.Background()

	hasAccess, err := ns.node.AddressHasPullAccess(ctx, addr, req.RepoID)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
		return
	}

	//
	// Send our response:
	// 1. peer is not authorized to pull
	//    - GetObjectResponse{Unauthorized: true}
	//    - <close connection>
	// 2. we don't have the object:
	//    - GetObjectResponse{HasObject: false}
	//    - <close connection>
	// 3. we do have the object:
	//    - GetObjectResponse{HasObject: true, ObjectType: ..., ObjectLen: ...}
	//    - [stream of object bytes...]
	//    - <close connection>
	//
	r := ns.node.Repo(req.RepoID)
	if r == nil {
		log.Warnf("[p2p object server] cannot find repo %v", req.RepoID)
		err := WriteStructPacket(stream, &GetObjectResponse{HasObject: false})
		if err != nil {
			log.Errorf("[p2p object server] %v", err)
			return
		}
		return
	}

	if hasAccess == false {
		log.Warnf("[p2p object server] address 0x%0x does not have pull access", addr.Bytes())
		err := WriteStructPacket(stream, &GetObjectResponse{Unauthorized: true})
		if err != nil {
			log.Errorf("[p2p object server] %v", err)
			return
		}
		return
	}

	objectStream, err := r.OpenObject(req.ObjectID)
	if err != nil {
		log.Debugf("[p2p object server] we don't have %v %0x (err: %v)", req.RepoID, req.ObjectID, err)

		// tell the peer we don't have the object and then close the connection
		err := WriteStructPacket(stream, &GetObjectResponse{HasObject: false})
		if err != nil {
			log.Errorf("[p2p object server] %v", err)
			return
		}
		return
	}
	defer objectStream.Close()

	err = WriteStructPacket(stream, &GetObjectResponse{
		Unauthorized: false,
		HasObject:    true,
		ObjectType:   objectStream.Type(),
		ObjectLen:    objectStream.Len(),
	})
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
		return
	}

	sent, err := io.Copy(stream, objectStream)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
	} else if uint64(sent) < objectStream.Len() {
		log.Errorf("[p2p object server] terminated while sending")
	}

	log.Printf("[p2p object server] sent %v %0x (%v bytes)", req.RepoID, req.ObjectID, sent)
}

// Handles incoming requests for commit manifests
func (ns *NaiveServer) HandleManifestRequest(stream netp2p.Stream) {
	defer stream.Close()

	// Read the request packet
	req := GetManifestRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
		return
	}

	addr, err := ns.node.AddrFromSignedHash([]byte(req.Commit), req.Signature)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hasAccess, err := ns.node.AddressHasPullAccess(ctx, addr, req.RepoID)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
		return
	}

	if hasAccess == false {
		log.Warnf("[p2p object server] address 0x%0x does not have pull access", addr.Bytes())
		err := WriteStructPacket(stream, &GetManifestResponse{Authorized: false})
		if err != nil {
			log.Errorf("[p2p object server] %v", err)
			return
		}
		return
	}

	// Send our response:
	// 1. peer is not authorized to pull
	//    - GetManifestResponse{Authorized: false}
	//    - <close connection>
	// 2. we don't have the repo/commit:
	//    - GetCommitResponse{HasCommit: false}
	//    - <close connection>
	// 3. we do have the commit:
	//    - GetCommitResponse{Authorized: true, HasCommit: true, ManifestLen: ...}
	//    - [stream of manifest bytes...]
	//    - <close connection>
	//
	r := ns.node.Repo(req.RepoID)
	if r == nil {
		log.Warnf("[p2p object server] cannot find repo %v", req.RepoID)
		err := WriteStructPacket(stream, &GetManifestResponse{HasCommit: false})
		if err != nil {
			log.Errorf("[p2p object server] %v", err)
			return
		}
		return
	}

	flatHead, flatHistory, err := r.GetManifest()
	if err != nil {
		log.Warnf("[p2p object server] cannot get manifest for repo %v", req.RepoID)
		err := WriteStructPacket(stream, &GetManifestResponse{HasCommit: false})
		if err != nil {
			log.Errorf("[p2p object server] %v", err)
			return
		}
		return
	}

	err = WriteStructPacket(stream, &GetManifestResponse{
		Authorized: true,
		HasCommit:  true,
		HeadLen:    len(flatHead),
		HistoryLen: len(flatHistory),
	})
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
		return
	}

	sent, err := stream.Write(flatHead)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
	} else if sent < len(flatHead) {
		log.Errorf("[p2p object server] terminated while sending head")
	}

	sent, err = stream.Write(flatHistory)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
	} else if sent < len(flatHead) {
		log.Errorf("[p2p object server] terminated while sending history")
	}

	log.Printf("[p2p object server] sent manifest for %v %v (%v bytes)", req.RepoID, req.Commit, sent)
}
