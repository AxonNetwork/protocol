package strategy

import (
	"context"

	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"

	"github.com/Conscience/protocol/util"
)

type peerPool struct {
	peers       chan IPeerConnection
	chProviders chan peer.ID
	needNewPeer chan struct{}
	chClose     chan struct{}
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
		chClose:     make(chan struct{}),
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

			var peerID peer.ID
			select {
			case peerID = <-p.chProviders:
			case <-p.ctx.Done():
				return
			}

			// ... make peer connection ...

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
		case p.ctx.Done():
		}

	} else {
		select {
		case p.peers <- conn:
		case p.ctx.Done():
		}
	}
}
