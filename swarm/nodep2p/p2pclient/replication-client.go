package p2pclient

import (
	"context"
	"sync"
	"time"

	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodep2p"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

func RequestBecomeReplicator(ctx context.Context, n nodep2p.INode, repoID string) error {
	cfg := n.GetConfig()
	for _, pubkeyStr := range cfg.Node.KnownReplicators {
		peerID, err := peer.IDB58Decode(pubkeyStr)
		if err != nil {
			log.Errorf("RequestBecomeReplicator: bad pubkey string '%v': %v", pubkeyStr, err)
			continue
		}

		stream, err := n.NewStream(ctx, peerID, nodep2p.BECOME_REPLICATOR_PROTO)
		if err != nil {
			log.Errorf("RequestBecomeReplicator: error connecting to peer %v: %v", peerID, err)
			continue
		}

		err = WriteStructPacket(stream, &BecomeReplicatorRequest{RepoID: repoID})
		if err != nil {
			log.Errorf("RequestBecomeReplicator: error writing request: %v", err)
			continue
		}

		resp := BecomeReplicatorResponse{}
		err = ReadStructPacket(stream, &resp)
		if err != nil {
			log.Errorf("RequestBecomeReplicator: error reading response: %v", err)
			continue
		}

		if resp.Error == "" {
			log.Infof("RequestBecomeReplicator: peer %v agreed to replicate %v", peerID, repoID)
		} else {
			log.Infof("RequestBecomeReplicator: peer %v refused to replicate %v (err: %v)", peerID, repoID, resp.Error)
		}
	}
	return nil
}

// Finds replicator nodes on the network that are hosting the given repository and issues requests
// to them to pull from our local copy.
func RequestReplication(ctx context.Context, n nodep2p.INode, repoID string) <-chan nodep2p.MaybeReplProgress {
	progressCh := make(chan nodep2p.MaybeReplProgress)
	c, err := util.CidForString("replicate:" + repoID)
	if err != nil {
		go func() {
			defer close(progressCh)
			progressCh <- nodep2p.MaybeReplProgress{Error: err}
		}()
		return progressCh
	}

	// @@TODO: configurable timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	chProviders := n.FindProvidersAsync(ctxTimeout, c, 8)

	peerChs := make(map[peer.ID]chan Progress)
	for provider := range chProviders {
		if provider.ID == n.ID() {
			continue
		}

		peerCh := make(chan Progress)
		go requestPeerReplication(ctx, n, repoID, provider.ID, peerCh)
		peerChs[provider.ID] = peerCh
	}

	go combinePeerChs(peerChs, progressCh)

	return progressCh
}

func requestPeerReplication(ctx context.Context, n nodep2p.INode, repoID string, peerID peer.ID, peerCh chan Progress) {
	var err error

	defer func() {
		defer close(peerCh)
		if err != nil {
			log.Errorf("[pull error: %v]", err)
			peerCh <- Progress{Error: err}
		}
	}()

	stream, err := n.NewStream(ctx, peerID, nodep2p.REPLICATION_PROTO)
	if err != nil {
		return
	}
	defer stream.Close()

	err = WriteStructPacket(stream, &ReplicationRequest{RepoID: repoID})
	if err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			peerCh <- Progress{Error: ctx.Err()}
			return
		default:
		}

		p := Progress{}
		err = ReadStructPacket(stream, &p)
		if err != nil {
			return
		}

		// Convert from an over-the-wire error (a string) to a native Go error
		if p.ErrorMsg != "" {
			p.Error = errors.New(p.ErrorMsg)
		}

		peerCh <- p

		if p.Done == true || p.Error != nil {
			return
		}
	}
}

func combinePeerChs(peerChs map[peer.ID]chan Progress, progressCh chan nodep2p.MaybeReplProgress) {
	defer close(progressCh)

	if len(peerChs) == 0 {
		err := errors.Errorf("no replicators available")
		progressCh <- nodep2p.MaybeReplProgress{Error: err}
		return
	}

	maxPercent := 0
	chPercent := make(chan int)
	someoneFinished := false
	wg := &sync.WaitGroup{}
	chDone := make(chan struct{})

	go func() {
		defer close(chDone)
		for p := range chPercent {
			if maxPercent < p {
				maxPercent = p
				progressCh <- nodep2p.MaybeReplProgress{Percent: maxPercent}
			}
		}
	}()

	for _, peerCh := range peerChs {
		wg.Add(1)
		go func(peerCh chan Progress) {
			defer wg.Done()
			for progress := range peerCh {
				if progress.Done == true {
					someoneFinished = true
				}
				if progress.Error != nil {
					return
				}
				percent := 0
				if progress.Total > 0 {
					percent = int(100 * progress.Current / progress.Total)
				}
				chPercent <- percent
			}
		}(peerCh)
	}

	wg.Wait()
	close(chPercent)
	<-chDone

	if !someoneFinished {
		err := errors.Errorf("every replicator failed to replicate repo")
		progressCh <- nodep2p.MaybeReplProgress{Error: err}
	}
}
