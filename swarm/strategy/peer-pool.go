package strategy

import (
	"context"

	peerstore "gx/ipfs/QmZR2XWVVBCtbgBWnQhWk2xcQfaR3W8faQPriAiaaj7rsr/go-libp2p-peerstore"
	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/util"
)

type peerPool struct {
	peers       chan IPeerConnection
	chProviders <-chan peerstore.PeerInfo
	needNewPeer chan struct{}
	ctx         context.Context
	cancel      func()
}

func newPeerPool(ctx context.Context, node INode, repoID string, concurrentConns int) (*peerPool, error) {
	cid, err := util.CidForString(repoID)
	if err != nil {
		return nil, err
	}

	ctxInner, cancel := context.WithCancel(ctx)

	p := &peerPool{
		peers:       make(chan IPeerConnection, concurrentConns),
		chProviders: node.FindProvidersAsync(ctxInner, cid, 999),
		needNewPeer: make(chan struct{}),
		ctx:         ctxInner,
		cancel:      cancel,
	}

	go func() {
		for {
			select {
			case <-p.needNewPeer:
			case <-p.ctx.Done():
				return
			}

			var peerConn IPeerConnection
			for {
				var peerID peer.ID
				select {
				case peerInfo := <-p.chProviders:
					peerID = peerInfo.ID
				case <-p.ctx.Done():
					return
				}

				_peerConn, err := NewPeerConnection(node, peerID, repoID)
				if err != nil {
					log.Errorln("[peer pool] error opening NewPeerConnection", err)
					continue
				}
				peerConn = _peerConn
				break
			}

			select {
			case p.peers <- peerConn:
			case <-p.ctx.Done():
				return
			}
		}
	}()

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

	return nil
}

func (p *peerPool) GetConn() IPeerConnection {
	select {
	case x := <-p.peers:
		return x
	case <-p.ctx.Done():
		return nil
	}
}

func (p *peerPool) ReturnConn(conn IPeerConnection, strike bool) {
	if strike {
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
