package swarm

import (
	"context"
	"encoding/hex"
	"io"

	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"

	netp2p "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
)

// Handles incoming requests for objects.
func (n *Node) objectStreamHandler(stream netp2p.Stream) {
	defer stream.Close()
	log.Printf("handling stream")

	// Read the request packet
	req := GetObjectRequestSigned{}
	err := readStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}

	hash := crypto.Keccak256(req.ObjectID)
	pubkey, err := crypto.SigToPub(hash, req.Signature)
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}
	addr := crypto.PubkeyToAddress(*pubkey)
	ctx := context.Background()

	hasAccess, err := n.Eth.AddressHasPullAccess(ctx, addr, req.RepoID)
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}

	log.Printf("[stream] peer requested %v %v", req.RepoID, hex.EncodeToString(req.ObjectID))
	log.Printf("[stream] address 0x%s has pull access %t", hex.EncodeToString(addr.Bytes()), hasAccess)

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
		err := writeStructPacket(stream, &GetObjectResponse{HasObject: false})
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}
		return
	}

	if hasAccess == false {
		err := writeStructPacket(stream, &GetObjectResponse{Unauthorized: true})
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}
		return
	}

	objectStream, err := r.OpenObject(req.ObjectID)
	if err != nil {
		log.Printf("[stream] we don't have %v %v (err: %v)", req.RepoID, hex.EncodeToString(req.ObjectID), err)

		// tell the peer we don't have the object and then close the connection
		err := writeStructPacket(stream, &GetObjectResponse{HasObject: false})
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}
		return
	}
	defer objectStream.Close()

	err = writeStructPacket(stream, &GetObjectResponse{
		Unauthorized: false,
		HasObject:    true,
		ObjectType:   objectStream.Type(),
		ObjectLen:    objectStream.Len(),
	})
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}

	sent, err := io.Copy(stream, objectStream)
	if err != nil {
		log.Errorf("[stream] %v", err)
	} else if sent < objectStream.Len() {
		log.Errorf("[stream] terminated while sending")
	}

	log.Printf("[stream] sent %v %v (%v bytes)", req.RepoID, hex.EncodeToString(req.ObjectID), sent)
}
