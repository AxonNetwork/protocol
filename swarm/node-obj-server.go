package swarm

import (
	"encoding/hex"
	"io"

	log "github.com/sirupsen/logrus"

	netp2p "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
)

// Handles incoming requests for objects.
func (n *Node) objectStreamHandler(stream netp2p.Stream) {
	defer stream.Close()

	// Read the request packet
	req := GetObjectRequest{}
	err := readStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}

	log.Printf("[stream] peer requested %v %v", req.RepoID, hex.EncodeToString(req.ObjectID))

	//
	// Send our response:
	// 1. we don't have the object:
	//    - GetObjectResponse{HasObject: false}
	//    - <close connection>
	// 2. we do have the object:
	//    - GetObjectResponse{HasObject: true, ObjectType: ..., ObjectLen: ...}
	//    - [stream of object bytes...]
	//    - <close connection>
	//
	objectStream, err := n.RepoManager.OpenObject(req.RepoID, req.ObjectID)
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
		HasObject:  true,
		ObjectType: objectStream.Type(),
		ObjectLen:  objectStream.Len(),
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
