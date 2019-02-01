package p2pclient

import (
	"context"

	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"

	"github.com/Conscience/protocol/swarm/nodep2p"
	"github.com/Conscience/protocol/util"
)

type peerPool struct {
	peers       chan *PeerConnection
	chProviders <-chan peerstore.PeerInfo
	needNewPeer chan struct{}
	ctx         context.Context
	cancel      func()
}

func newPeerPool(ctx context.Context, node nodep2p.INode, repoID string, concurrentConns uint) (*peerPool, error) {
	cid, err := util.CidForString(repoID)
	if err != nil {
		return nil, err
	}

	ctxInner, cancel := context.WithCancel(ctx)

	p := &peerPool{
		peers:       make(chan *PeerConnection, concurrentConns),
		chProviders: node.FindProvidersAsync(ctxInner, cid, 999),
		needNewPeer: make(chan struct{}),
		ctx:         ctxInner,
		cancel:      cancel,
	}

	// When a message is sent on the `needNewPeer` channel, this goroutine attempts to take a peer
	// from the `chProviders` channel, open a PeerConnection to it, and add it to the pool.
	go func() {
		for {
			select {
			case <-p.needNewPeer:
			case <-p.ctx.Done():
				return
			}

			var peerConn *PeerConnection
			for {
				var peerID peer.ID
				select {
				case peerInfo, open := <-p.chProviders:
					if !open {
						p.chProviders = node.FindProvidersAsync(p.ctx, cid, 999)
						continue
					}
					// if self
					if peerInfo.ID == node.ID() {
						continue
					}
					peerID = peerInfo.ID
				case <-p.ctx.Done():
					return
				}

				// if _, exists := p.peerList[peerID]; exists {
				// 	continue
				// }

				peerConn = NewPeerConnection(node, peerID, repoID)
				break
			}

			// p.peerList[peerConn.peerID] = peerConn

			select {
			case p.peers <- peerConn:
			case <-p.ctx.Done():
				return
			}
		}
	}()

	// This goroutine fills the peer pool with the desired number of peers.
	go func() {
		for i := uint(0); i < concurrentConns; i++ {
			select {
			case <-p.ctx.Done():
				return
			case p.needNewPeer <- struct{}{}:
			}
		}
	}()

	return p, nil
}

func (p *peerPool) Close() error {
	p.cancel()

	p.needNewPeer = nil
	p.chProviders = nil
	p.peers = nil

	return nil
}

func (p *peerPool) GetConn() *PeerConnection {
	select {
	case x := <-p.peers:
		return x
	case <-p.ctx.Done():
		return nil
	}
}

func (p *peerPool) ReturnConn(conn *PeerConnection, strike bool) {
	if strike {
		// if _, exists := p.peerList[conn.peerID]; exists {
		// 	delete(p.peerList, conn.peerID)
		// }

		select {
		case p.needNewPeer <- struct{}{}:
		case <-p.ctx.Done():
		}

	} else {
		select {
		case p.peers <- conn:
		case <-p.ctx.Done():
		}
	}
}
