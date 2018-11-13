package swarm

import (
	"context"
	netp2p "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
	"time"

	"github.com/Conscience/protocol/log"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

type RepoSwarmManager struct {
	peers    []Peer
	objQueue []gitplumbing.Hash
	node     *Node
}

func NewRepoSwarmManager(node *Node) *RepoSwarmManager {
	sm := &RepoSwarmManager{
		peers:    []Peer{},
		objQueue: []gitplumbing.Hash{},
		node:     node,
	}
	return sm
}

func (sm *RepoSwarmManager) FindPeersWithCommit(ctx context.Context, repoID string, commit string) error {
	repoCid, err := cidForString(repoID)
	if err != nil {
		return err
	}
	commitCid, err := cidForString(commit)
	if err != nil {
		return err
	}

	ctxTimeout, cancel := context.WithTImeout(ctx, time.Duration(sm.node.Config.Node.FindProviderTimeout))
	defer cancel()
	syncedPeers := make([]Peer, 0)
	for provider := range sm.node.dht.FindProvidersAsync(ctxTimeout, commitCid, 10) {
		log.Debugf("Found provider for %v : %v", repoID, commit)
		if provider.ID != n.host.ID() {
			// We found a peer with the object
			commit, err := n.handshake(ctx, provider.ID, repoID)
			if err != nil {
				log.Warnln("[p2p swarm client] error handshaking peer:", err)
				continue
			}
			return nil
		}
	}
}

func (sm *RepoSwarmManager) AddNewPeer(stream netp2p.Stream) error {
	sm.peers = append(sm.Peers, Peer{
		stream:  stream,
		strikes: 0,
	})

	return nil
}

type Peer struct {
	stream        netp2p.Stream
	strikes       int
	currentCommit string
}
