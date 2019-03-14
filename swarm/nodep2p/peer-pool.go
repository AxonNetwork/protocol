package nodep2p

import (
	"context"
	"sync"
	"time"

	netp2p "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/util"
)

type peerPool struct {
	keepalive bool
	node      INode

	chPeers       chan *peerConn
	chNeedNewPeer chan struct{}
	chProviders   <-chan peerstore.PeerInfo
	ctx           context.Context
	cancel        func()
	foundPeers    map[peer.ID]bool
}

func newPeerPool(ctx context.Context, node INode, repoID string, concurrentConns uint, protocolID protocol.ID, keepalive bool) (*peerPool, error) {
	cid, err := util.CidForString(repoID)
	if err != nil {
		return nil, err
	}

	ctxInner, cancel := context.WithCancel(ctx)
	foundPeers := make(map[peer.ID]bool)

	p := &peerPool{
		keepalive:     keepalive,
		node:          node,
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
		FindPeerLoop:
			for {
				var peerID peer.ID
				select {
				case peerInfo, open := <-p.chProviders:
					if !open {
						p.chProviders = node.FindProvidersAsync(p.ctx, cid, 999)
						continue FindPeerLoop
					}
					// if self
					if peerInfo.ID == node.ID() {
						continue FindPeerLoop
					} else if foundPeers[peerInfo.ID] {
						continue FindPeerLoop
					}
					peerID = peerInfo.ID
					foundPeers[peerID] = true

				case <-p.ctx.Done():
					return
				}

				log.Infof("[peer pool] found peer %v (repoID: %v, protocolID: %v)", peerID.String(), repoID, protocolID)
				conn = newPeerConn(ctxInner, p.node, peerID, repoID, protocolID)

				break
			}

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

	return nil
}

func (p *peerPool) GetConn() (*peerConn, error) {
	select {
	case conn := <-p.chPeers:

		err := conn.Open()
		return conn, err

	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	}
}

func (p *peerPool) ReturnConn(conn *peerConn, strike bool) {
	if strike {
		// Close the faulty connection
		conn.Close()

		// Try to obtain a new peer
		select {
		case p.chNeedNewPeer <- struct{}{}:
		case <-p.ctx.Done():
		}

	} else {
		if !p.keepalive {
			conn.Close()
		}

		// Return the peer to the pool
		select {
		case p.chPeers <- conn:
		case <-p.ctx.Done():
		}
	}
}

type peerConn struct {
	netp2p.Stream
	node       INode
	peerID     peer.ID
	repoID     string
	protocolID protocol.ID
	ctx        context.Context
	mu         *sync.Mutex
	done       bool
}

func newPeerConn(ctx context.Context, node INode, peerID peer.ID, repoID string, protocolID protocol.ID) *peerConn {
	conn := &peerConn{
		node:       node,
		peerID:     peerID,
		repoID:     repoID,
		protocolID: protocolID,
		Stream:     nil,
		ctx:        ctx,
		mu:         &sync.Mutex{},
		done:       false,
	}

	go func() {
		<-ctx.Done()
		conn.closeForever()
	}()

	return conn
}

func (conn *peerConn) Open() error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.done {
		log.Debugf("[peer conn] peerConn is done, refusing to reopen (repoID: %v, proto: %v)", conn.repoID, conn.protocolID)
		return nil
	}

	if conn.Stream == nil {
		log.Debugf("[peer conn] peerConn.Stream is nil, opening new connection (proto: %v)", conn.protocolID)

		// @@TODO: make context timeout configurable
		ctxConnect, cancel := context.WithTimeout(conn.ctx, 15*time.Second)
		defer cancel()

		stream, err := conn.node.NewStream(ctxConnect, conn.peerID, conn.protocolID)
		if err != nil {
			return err
		}

		log.Debugln("[peer conn] peerConn.Stream successfully opened")
		conn.Stream = stream
	}

	return nil
}

func (conn *peerConn) close() error {
	if conn.Stream != nil {
		log.Debugf("[peer conn] closing peerConn %v (repoID: %v, protocolID: %v)", conn.peerID, conn.repoID, conn.protocolID)

		err := conn.Stream.Close()
		if err != nil {
			log.Warnln("[peer conn] error closing peerConn:", err)
		}
		conn.Stream = nil

	} else {
		log.Debugf("[peer conn] already closed: peerConn %v (repoID: %v, protocolID: %v)", conn.peerID, conn.repoID, conn.protocolID)
	}
	return nil
}

func (conn *peerConn) closeForever() error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.done = true
	return conn.close()
}

func (conn *peerConn) Close() error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	return conn.close()
}
