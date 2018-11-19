package nodep2p

import (
	"context"
	"time"

	peerstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/util"
)

type peerPool struct {
	peers       chan *PeerConnection
	chProviders <-chan peerstore.PeerInfo
	needNewPeer chan struct{}

	peerList map[*PeerConnection]struct{} // This is only used to close peers when .Close() is called.

	ctx    context.Context
	cancel func()
}

func newPeerPool(ctx context.Context, node INode, repoID string, concurrentConns int) (*peerPool, error) {
	cid, err := util.CidForString(repoID)
	if err != nil {
		return nil, err
	}

	ctxInner, cancel := context.WithCancel(ctx)

	p := &peerPool{
		peers:       make(chan *PeerConnection, concurrentConns),
		chProviders: node.FindProvidersAsync(ctxInner, cid, 999),
		needNewPeer: make(chan struct{}),
		peerList:    make(map[*PeerConnection]struct{}),
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
					peerID = peerInfo.ID
				case <-p.ctx.Done():
					return
				}

				_peerConn, err := NewPeerConnection(p.ctx, node, peerID, repoID)
				if err != nil {
					log.Errorln("[peer pool] error opening NewPeerConnection", err)
					time.Sleep(1 * time.Second)
					continue
				}
				peerConn = _peerConn
				break
			}

			p.peerList[peerConn] = struct{}{}

			select {
			case p.peers <- peerConn:
			case <-p.ctx.Done():
				return
			}
		}
	}()

	// This goroutine fills the peer pool with the desired number of peers.
	go func() {
		for i := 0; i < concurrentConns; i++ {
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

	for conn := range p.peerList {
		err := conn.Close()
		if err != nil {
			log.Errorln("[peer pool] Close: error closing connection", err)
		}
	}

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
		if _, exists := p.peerList[conn]; exists {
			delete(p.peerList, conn)
		}

		err := conn.Close()
		if err != nil {
			log.Errorln("[peer pool] ReturnConn: error closing connection", err)
		}

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
