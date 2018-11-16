package swarm

import (
	"context"
	"io"
	"time"

	netp2p "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"

	"github.com/Conscience/protocol/log"
	. "github.com/Conscience/protocol/swarm/wire"
)

func (n *Node) handleHandshakeRequest(stream netp2p.Stream) {
	req := HandshakeRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[p2p swarm server] %v", err)
		return
	}

	addr, err := n.eth.AddrFromSignedHash([]byte(req.RepoID), req.Signature)
	if err != nil {
		log.Errorf("[p2p swarm server] %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hasAccess, err := n.eth.AddressHasPullAccess(ctx, addr, req.RepoID)
	if err != nil {
		log.Errorf("[p2p swarm server] %v", err)
		return
	}

	if hasAccess == false {
		log.Warnf("[p2p swarm server] address 0x%0x does not have pull access", addr.Bytes())
		err := WriteStructPacket(stream, &HandshakeResponse{Authorized: false})
		if err != nil {
			log.Errorf("[p2p swarm server] %v", err)
			return
		}
		return
	}
	repo := n.RepoManager.Repo(req.RepoID)
	commit, err := repo.HeadHash()
	if err != nil {
		log.Errorf("[p2p swarm server] %v", err)
		return
	}
	err = WriteStructPacket(stream, &HandshakeResponse{Authorized: true, Commit: commit})
	if err != nil {
		log.Errorf("[p2p swarm server] %v", err)
		return
	}
	go n.connectLoop(req.RepoID, stream)
}

func (n *Node) connectLoop(repoID string, stream netp2p.Stream) {
	defer stream.Close()
	for {
		req := GetObjectRequest{}
		err := ReadStructPacket(stream, &req)
		if err != nil {
			log.Debugf("[p2p object server] Stream closed")
			return
		}
		go n.writeObjectToStream(repoID, req.ObjectID, stream)
	}
}

func (n *Node) writeObjectToStream(repoID string, objectID []byte, stream netp2p.Stream) {
	r := n.RepoManager.Repo(repoID)
	if r == nil {
		log.Warnf("[p2p object server] cannot find repo %v", repoID)
		err := WriteStructPacket(stream, &GetObjectResponse{HasObject: false})
		if err != nil {
			log.Errorf("[p2p object server] %v", err)
			return
		}
		return
	}

	objectStream, err := r.OpenObject(objectID)
	if err != nil {
		log.Debugf("[p2p object server] we don't have %v %0x (err: %v)", repoID, objectID, err)

		// tell the peer we don't have the object
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
		ObjectID:     objectID,
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
		return
	} else if uint64(sent) < objectStream.Len() {
		log.Errorf("[p2p object server] terminated while sending")
		return
	}
}

// Handles incoming requests for commit manifests
func (n *Node) handleManifestRequest(stream netp2p.Stream) {
	defer stream.Close()

	// Read the request packet
	req := GetManifestRequest{}
	err := ReadStructPacket(stream, &req)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
		return
	}

	addr, err := n.eth.AddrFromSignedHash([]byte(req.Commit), req.Signature)
	if err != nil {
		log.Errorf("[p2p object server] %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hasAccess, err := n.eth.AddressHasPullAccess(ctx, addr, req.RepoID)
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
	r := n.RepoManager.Repo(req.RepoID)
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
	} else if sent < len(flatHistory) {
		log.Errorf("[p2p object server] terminated while sending history")
	}

	log.Printf("[p2p object server] sent manifest for %v %v (%v bytes)", req.RepoID, req.Commit, sent)
}
