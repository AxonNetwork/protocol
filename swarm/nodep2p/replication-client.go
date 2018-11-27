package nodep2p

import (
	"context"
	"sync"
	"time"

	"github.com/bugsnag/bugsnag-go/errors"
	peer "github.com/libp2p/go-libp2p-peer"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/swarm/nodegit"
	. "github.com/Conscience/protocol/swarm/wire"
	"github.com/Conscience/protocol/util"
)

func RequestBecomeReplicator(ctx context.Context, n INode, repoID string) error {
	cfg := n.GetConfig()
	for _, pubkeyStr := range cfg.Node.KnownReplicators {
		peerID, err := peer.IDB58Decode(pubkeyStr)
		if err != nil {
			log.Errorf("RequestBecomeReplicator: bad pubkey string '%v': %v", pubkeyStr, err)
			continue
		}

		stream, err := n.NewStream(ctx, peerID, BECOME_REPLICATOR_PROTO)
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

type MaybeReplProgress struct {
	Percent int
	Error   error
}

// Finds replicator nodes on the network that are hosting the given repository and issues requests
// to them to pull from our local copy.
func RequestReplication(ctx context.Context, n INode, repoID string) <-chan MaybeReplProgress {
	progressCh := make(chan MaybeReplProgress)
	c, err := util.CidForString("replicate:" + repoID)
	if err != nil {
		go func() {
			defer close(progressCh)
			progressCh <- MaybeReplProgress{Error: err}
		}()
		return progressCh
	}

	// @@TODO: configurable timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	chProviders := n.FindProvidersAsync(ctxTimeout, c, 8)

	peerChs := make(map[peer.ID](chan nodegit.MaybeProgress))
	for provider := range chProviders {
		if provider.ID == n.ID() {
			continue
		}

		peerCh := make(chan nodegit.MaybeProgress)
		go requestPeerReplication(ctx, n, repoID, provider.ID, peerCh)
		peerChs[provider.ID] = peerCh
	}

	go combinePeerChs(peerChs, progressCh)

	return progressCh
}

func requestPeerReplication(ctx context.Context, n INode, repoID string, peerID peer.ID, ch chan nodegit.MaybeProgress) {
	defer close(ch)
	var err error
	defer func() {
		defer close(ch)
		if err != nil {
			log.Errorf("[pull error: %v]", err)
			ch <- nodegit.MaybeProgress{Error: err}
		}
	}()
	stream, err := n.NewStream(ctx, peerID, REPLICATION_PROTO)
	if err != nil {
		return
	}
	defer stream.Close()

	err = WriteStructPacket(stream, &ReplicationRequest{RepoID: repoID})
	if err != nil {
		return
	}

	for {
		resp := ReplicationProgress{}
		err = ReadStructPacket(stream, &resp)
		if err != nil {
			return
		}
		if resp.Error != "" {
			err = errors.Errorf(resp.Error)
			return
		}
		if resp.Done == true {
			return
		}
		ch <- nodegit.MaybeProgress{
			Fetched: resp.Fetched,
			ToFetch: resp.ToFetch,
		}
	}
}
func combinePeerChs(peerChs map[peer.ID](chan nodegit.MaybeProgress), progressCh chan MaybeReplProgress) {
	defer close(progressCh)
	if len(peerChs) == 0 {
		err := errors.Errorf("no replicators available")
		progressCh <- MaybeReplProgress{Error: err}
	}
	maxPercent := 0
	percentMutex := &sync.Mutex{}
	done := false
	wg := &sync.WaitGroup{}
	for _, ch := range peerChs {
		go func() {
			defer wg.Done()
			wg.Add(1)
			for progress := range ch {
				if done {
					return
				}
				if progress.Error != nil {
					return
				}
				percent := int(progress.Fetched / progress.ToFetch)
				if percent > maxPercent {
					percentMutex.Lock()
					maxPercent = percent
					percentMutex.Unlock()
					progressCh <- MaybeReplProgress{Percent: percent}
				}
			}
			// peer successfully replicated repo
			done = true
		}()
	}
	wg.Wait()
	if !done {
		err := errors.Errorf("every replicator failed to replicate repo")
		progressCh <- MaybeReplProgress{Error: err}
	}
}
