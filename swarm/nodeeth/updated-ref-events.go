package nodeeth

// import (
// 	"context"
// 	"time"

// 	"github.com/ethereum/go-ethereum/crypto"
// )

// type UpdatedRefEventWatcher struct {
// 	chOut     chan MaybeUpdatedRefEvent
// 	chAddRepo chan string
// 	repoIDs   []string
// 	eth       *Client
// 	ctx       context.Context
// }

// type MaybeUpdatedRefEvent struct {
// 	Event UpdatedRefEvent
// 	Error error
// }

// func NewUpdatedRefEventWatcher(ctx context.Context, eth *Client, repoIDs []string, fromBlock uint64) *UpdatedRefEventWatcher {
// 	rw := &UpdatedRefEventWatcher{
// 		chOut:     make(chan MaybeUpdatedRefEvent),
// 		chAddRepo: make(chan string),
// 		ctx:       ctx,
// 	}

// 	go rw.runLoop(eth, repoIDs, fromBlock)

// 	return rw
// }

// func (rw *UpdatedRefEventWatcher) Events() <-chan MaybeUpdatedRefEvent {
// 	return rw.chOut
// }

// func (rw *UpdatedRefEventWatcher) AddRepo(ctx context.Context, repoID string) {
// 	select {
// 	case rw.chAddRepo <- repoID:
// 	case <-rw.ctx.Done():
// 	case <-ctx.Done():
// 	}
// }

// func (rw *UpdatedRefEventWatcher) runLoop(eth *Client, repoIDs []string, fromBlock uint64) {
// 	defer close(rw.chOut)

// 	repoIDByHash := make(map[string]string)
// 	for _, repoID := range repoIDs {
// 		hash := crypto.Keccak256([]byte(repoID))
// 		repoIDByHash[string(hash)] = repoID
// 	}

// 	cursor := fromBlock

// 	logsTimer := time.NewTicker(10 * time.Second) // @@TODO: make this configurable
// 	defer logsTimer.Stop()

// 	for {
// 		select {
// 		case <-rw.ctx.Done():
// 			return

// 		case <-logsTimer.C:
// 			ctx, cancel := context.WithTimeout(rw.ctx, 30*time.Second) // @@TODO: make this configurable
// 			defer cancel()

// 			evts, err := rw.eth.GetUpdatedRefEvents(ctx, rw.repoIDs, cursor, nil)
// 			if err != nil {
// 				select {
// 				case rw.chOut <- MaybeUpdatedRefEvent{Error: err}:
// 				case <-rw.ctx.Done():
// 				}
// 				return
// 			}
// 			cancel()

// 			for _, evt := range evts {
// 				hashStr := string(evt.RepoIDHash[:])
// 				evt.RepoID = repoIDByHash[hashStr]

// 				select {
// 				case rw.chOut <- MaybeUpdatedRefEvent{Event: evt}:
// 				case <-rw.ctx.Done():
// 				}

// 				cursor = evt.BlockNumber + 1
// 			}

// 		case repoID := <-rw.chAddRepo:
// 			rw.repoIDs = append(rw.repoIDs, repoID)
// 			hash := crypto.Keccak256([]byte(repoID))
// 			repoIDByHash[string(hash)] = repoID
// 		}
// 	}
// }
