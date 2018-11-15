package swarm

import (
	"context"

	"github.com/Conscience/protocol/util"
	peer "github.com/libp2p/go-libp2p-peer"
)

type PeerConnection struct {
	peerID peer.ID
	repoID string
	stream netp2p.Stream
}

func NewPeerConnection(peerID peer.ID, repoID string) error {
	pc := &PeerConnect{}
}

func (pc *PeerConnection) RequestObject(ctx context.Context, objectID []byte) (*util.ObjectReader, error) {

}

func (pc *PeerConnection) Close() error {

}
