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

	var repoID, objectIDStr string
	var objectID []byte
	var err error

	//
	// read the repo ID
	//
	{
		repoIDLen, err := readUint64(stream)
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}

		repoIDBytes := make([]byte, repoIDLen)
		_, err = io.ReadFull(stream, repoIDBytes)
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}

		repoID = string(repoIDBytes)
	}

	//
	// read the object ID
	//
	{
		objectIDLen, err := readUint64(stream)
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}

		objectID = make([]byte, objectIDLen)
		_, err = io.ReadFull(stream, objectID)
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}

		objectIDStr = hex.EncodeToString(objectID)
	}

	log.Printf("[stream] peer requested %v %v", repoID, objectIDStr)

	//
	// send our response:
	// 1. we don't have the object:
	//    - 0x0
	//    - <close connection>
	// 2. we do have the object:
	//    - 0x1
	//    - [object type byte, only matters for Git objects]
	//    - [object length]
	//    - [object bytes...]
	//    - <close connection>
	//
	object, exists := n.RepoManager.Object(repoID, objectID)
	if !exists {
		log.Printf("[stream] we don't have %v %v", repoID, objectIDStr)

		// tell the peer we don't have the object and then close the connection
		_, err := stream.Write([]byte{0x0})
		if err != nil {
			log.Errorf("[stream] %v", err)
			return
		}
		return
	}

	_, err = stream.Write([]byte{0x1, byte(object.Type)})
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}

	objectStream, _, objectLen, err := n.RepoManager.OpenObject(repoID, objectID)
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}
	defer objectStream.Close()

	err = writeUint64(stream, uint64(objectLen))
	if err != nil {
		log.Errorf("[stream] %v", err)
		return
	}

	sent, err := io.Copy(stream, objectStream)
	if err != nil {
		log.Errorf("[stream] %v", err)
	}

	if sent < objectLen {
		log.Errorf("[stream] terminated while sending")
	}

	log.Printf("[stream] sent %v %v (%v bytes)", repoID, objectIDStr, sent)
}
