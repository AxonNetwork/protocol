package swarm

// import (
// 	"context"
// 	netp2p "gx/ipfs/QmPjvxTpVH8qJyQDnxnsxF9kv9jezKD1kozz1hs3fCGsNh/go-libp2p-net"
// 	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"
// 	"time"

// 	"github.com/Conscience/protocol/log"
// 	"github.com/pkg/errors"
// 	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
// )

// const PEER_QUEUE_LENGTH = 10
// const CONNECTED_PEERS_NUM = 4

// type RepoSwarmManager struct {
// 	peerQueue      []peer.ID
// 	connectedPeers []peer.ID
// 	peers          map[peer.ID]Peer
// 	objQueue       []gitplumbing.Hash
// 	node           *Node
// }

// func NewRepoSwarmManager(node *Node) *RepoSwarmManager {
// 	sm := &RepoSwarmManager{
// 		peerQueue:      []peer.ID{},
// 		connectedPeers: []peer.ID{},
// 		peers:          map[peer.ID]Peer{},
// 		objQueue:       []gitplumbing.Hash{},
// 		node:           node,
// 	}
// 	return sm
// }

// func (sm *RepoSwarmManager) FetchFromCommit(ctx context.Context, repoID string, commit string) error {
// 	for {
// 		if len(sm.peerQueue) == 0 {
// 			peers, err := sm.FindPeers(ctx, repoID, commit)
// 			if err != nil {
// 				return err
// 			}
// 			sm.peerQueue = peers
// 		}
// 		if len(sm.peerList) < CONNECTED_PEERS_NUM && len(sm.peerQueue) > 0 {

// 		}
// 	}
// 	return nil
// }

// func (sm *RepoSwarmManager) close() error {
// 	return nil
// }

// func (sm *RepoSwarmManager) FindPeers(ctx context.Context, repoID string, commit string) ([]peer.ID, error) {
// 	peerQueue := make([]peer.ID, 0)

// 	commitCid, err := cidForString(commit)
// 	if err != nil {
// 		return nil, err
// 	}
// 	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(sm.node.Config.Node.FindProviderTimeout))
// 	defer cancel()

// 	for provider := range sm.node.dht.FindProvidersAsync(ctxTimeout, commitCid, PEER_QUEUE_LENGTH+1) {
// 		log.Debugf("Found provider for %v : %v", repoID, commit)
// 		if provider.ID != n.host.ID() {
// 			peerQueue = append(peerQueue, provider.ID)
// 		}
// 	}

// 	if len(peerQueue) >= PEER_QUEUE_LENGTH {
// 		return peerQueue, nil
// 	}

// 	repoCid, err := cidForString(repoID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	for provider := range sm.node.dht.FindProvidersAsync(ctxTimeout, repoCid, PEER_QUEUE_LENGTH+1) {
// 		log.Debugf("Found provider for %v : %v", repoID, commit)
// 		if provider.ID != n.host.ID() && !containsPeer(peerQueue, provider.ID) {
// 			peerQueue = append(peerQueue, provider.ID)
// 		}
// 	}
// 	if len(peerQueue) == 0 {
// 		return nil, errors.Errorf("could not find provider for %v : %v", repoID, commit)
// 	}
// 	return peerQueue, nil
// }

// func containsPeer(peerList []peer.ID, peer peer.ID) bool {
// 	for i := range peerList {
// 		if peerList[i] == peer {
// 			return true
// 		}
// 	}
// 	return false
// }

// func (sm *RepoSwarmManager) AddNewPeer(stream netp2p.Stream) error {
// 	sm.peers = append(sm.Peers, Peer{
// 		stream:  stream,
// 		strikes: 0,
// 	})

// 	return nil
// }

// type Peer struct {
// 	stream        netp2p.Stream
// 	strikes       int
// 	currentCommit string
// }
