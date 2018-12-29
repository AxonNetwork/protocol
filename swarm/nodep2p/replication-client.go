package nodep2p

import (
	"context"
	"sync"
	"time"

	"github.com/bugsnag/bugsnag-go/errors"
	peer "github.com/libp2p/go-libp2p-peer"

	"github.com/Conscience/protocol/log"
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

type MaybePeerProgress struct {
	Fetched int64
	ToFetch int64
	Error   error
	Done    bool
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

	peerChs := make(map[peer.ID](chan MaybePeerProgress))
	for provider := range chProviders {
		if provider.ID == n.ID() {
			continue
		}

		peerCh := make(chan MaybePeerProgress)
		go requestPeerReplication(ctx, n, repoID, provider.ID, peerCh)
		peerChs[provider.ID] = peerCh
	}

	go combinePeerChs(peerChs, progressCh)

	return progressCh
}

func requestPeerReplication(ctx context.Context, n INode, repoID string, peerID peer.ID, peerCh chan MaybePeerProgress) {
	var err error

	defer func() {
		defer close(peerCh)
		if err != nil {
			log.Errorf("[pull error: %v]", err)
			peerCh <- MaybePeerProgress{Error: err}
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
		}

		peerCh <- MaybePeerProgress{
			Fetched: resp.Fetched,
			ToFetch: resp.ToFetch,
			Done:    resp.Done,
			Error:   err,
		}
		if resp.Done == true || err != nil {
			return
		}
	}
}

func combinePeerChs(peerChs map[peer.ID]chan MaybePeerProgress, progressCh chan MaybeReplProgress) {
	defer close(progressCh)

	if len(peerChs) == 0 {
		err := errors.Errorf("no replicators available")
		progressCh <- MaybeReplProgress{Error: err}
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
				progressCh <- MaybeReplProgress{Percent: maxPercent}
			}
		}
	}()

	for _, peerCh := range peerChs {
		wg.Add(1)
		go func(peerCh chan MaybePeerProgress) {
			defer wg.Done()
			for progress := range peerCh {
				if progress.Done == true {
					someoneFinished = true
				}
				if progress.Error != nil {
					return
				}
				percent := 0
				if progress.ToFetch > 0 {
					percent = int(100 * progress.Fetched / progress.ToFetch)
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
		progressCh <- MaybeReplProgress{Error: err}
	}
}
