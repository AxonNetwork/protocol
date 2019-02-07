package p2pclient

import (
	"context"

	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodep2p"
	"github.com/Conscience/protocol/util"
)

type peerPool struct {
	peers       chan *PeerConnection
	chProviders <-chan peerstore.PeerInfo
	needNewPeer chan struct{}
	ctx         context.Context
	cancel      func()
	openStreams bool
}

func newPeerPool(ctx context.Context, node nodep2p.INode, repoID string, concurrentConns uint, openStreams bool) (*peerPool, error) {
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
		openStreams: openStreams,
	}

	// When a message is sent on the `needNewPeer` channel, this goroutine attempts to take a peer
	// from the `chProviders` channel, open a PeerConnection to it, and add it to the pool.
	go func() {
		defer close(p.peers)
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

				log.Infof("[peer pool] opening new peer connection")
				peerConn = NewPeerConnection(node, peerID, repoID)
				if openStreams {
					err = peerConn.OpenStream(p.ctx)
					if err != nil {
						log.Debugf("[peer pool] error opening stream: ", err)
					} else {
						// if err then move onto the next peer
						break
					}
				} else {
					break
				}

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
	go func() {
		for x := range p.peers {
			x.Close()
		}
		p.peers = nil
	}()

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

		conn.Close()

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
