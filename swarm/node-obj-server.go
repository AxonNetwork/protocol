package swarm

import (
	"context"
	"io"

	netp2p "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"

	"github.com/Conscience/protocol/log"
	. "github.com/Conscience/protocol/swarm/wire"
)

// Handles incoming requests for objects.
func (n *Node) handleObjectRequest(stream netp2p.Stream) {
	defer stream.Close()

	// Read the request packet
	req := GetObjectRequestSigned{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
		return
	}

	log.Debugf("[p2p object server] peer requested %v %0x", req.RepoID, req.ObjectID)

	addr, err := n.eth.AddrFromSignedHash(req.ObjectID, req.Signature)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
		return
	}

	ctx := context.Background()

	hasAccess, err := n.eth.AddressHasPullAccess(ctx, addr, req.RepoID)
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
	r := n.RepoManager.Repo(req.RepoID)
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
