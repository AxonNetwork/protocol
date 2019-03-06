package p2pclient

import (
	"context"
	"time"

	netp2p "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodep2p"
	"github.com/Conscience/protocol/util"
)

type peerPool struct {
	protocol  protocol.ID
	keepalive bool
	node      nodep2p.INode

	chPeers       chan *peerConn
	chNeedNewPeer chan struct{}
	chProviders   <-chan peerstore.PeerInfo
	ctx           context.Context
	cancel        func()
}

type peerConn struct {
	peerID peer.ID
	repoID string
	netp2p.Stream
}

func newPeerPool(ctx context.Context, node nodep2p.INode, repoID string, concurrentConns uint, protocol protocol.ID, keepalive bool) (*peerPool, error) {
	cid, err := util.CidForString(repoID)
	if err != nil {
		return nil, err
	}

	ctxInner, cancel := context.WithCancel(ctx)

	p := &peerPool{
		keepalive:     keepalive,
		node:          node,
		protocol:      protocol,
		chPeers:       make(chan *peerConn, concurrentConns),
		chNeedNewPeer: make(chan struct{}),
		chProviders:   node.FindProvidersAsync(ctxInner, cid, 999),
		ctx:           ctxInner,
		cancel:        cancel,
	}

	// When a message is sent on the `needNewPeer` channel, this goroutine attempts to take a peer
	// from the `chProviders` channel, open a peerConn to it, and add it to the pool.
	go func() {
		defer close(p.chPeers)
		for {
			select {
			case <-p.chNeedNewPeer:
			case <-p.ctx.Done():
				return
			}

			var conn *peerConn
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

				log.Infof("[peer pool] found peer")
				conn = &peerConn{
					peerID: peerID,
					repoID: repoID,
					Stream: nil,
				}
				break
			}

			// p.peerList[conn.peerID] = conn

			select {
			case p.chPeers <- conn:
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
			case p.chNeedNewPeer <- struct{}{}:
			}
		}
	}()

	return p, nil
}

func (p *peerPool) Close() error {
	p.cancel()

	p.chNeedNewPeer = nil
	p.chProviders = nil
	go func() {
		for x := range p.chPeers {
			if x.Stream != nil {
				x.Stream.Close()
			}
		}
		p.chPeers = nil
	}()

	return nil
}

func (p *peerPool) GetConn() (*peerConn, error) {
	select {
	case conn := <-p.chPeers:

		if conn.Stream == nil {
			log.Debugf("[peer pool] peerConn.Stream is nil, opening new connection (proto: %v)", p.protocol)

			// @@TODO: make context timeout configurable
			ctx, cancel := context.WithTimeout(p.ctx, 15*time.Second)
			defer cancel()

			stream, err := p.node.NewStream(ctx, conn.peerID, p.protocol)
			if err != nil {
				return nil, err
			}
			log.Debugln("[peer pool] peerConn.Stream successfully opened")
			conn.Stream = stream
		}
		return conn, nil

	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	}
}

func (p *peerPool) ReturnConn(conn *peerConn, strike bool) {
	log.Println("Return Conn: ", conn)
	if strike {
		// if _, exists := p.peerList[conn.peerID]; exists {
		// 	delete(p.peerList, conn.peerID)
		// }

		conn.Close()
		conn.Stream = nil

		select {
		case p.chNeedNewPeer <- struct{}{}:
		case <-p.ctx.Done():
		}

	} else {
		if !p.keepalive {
			conn.Close()
			conn.Stream = nil
		}

		select {
		case p.chPeers <- conn:
		case <-p.ctx.Done():
		}
	}
}
