package nodeeth

import (
	"context"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type MaybeUpdatedRefEvent struct {
	Event UpdatedRefEvent
	Error error
}

type UpdatedRefEvent struct {
	Commit      string
	RefHash     string
	RepoIDHash  common.Hash
	RepoID      string
	TxHash      string
	Time        uint64
	BlockNumber uint64
}

type UpdatedRefEventWatcher struct {
	Ch       chan MaybeUpdatedRefEvent
	repoIDs  []string
	repoIDCh chan string
}

func (rw *UpdatedRefEventWatcher) AddRepo(repoID string) {
	rw.repoIDCh <- repoID
}

func (rw *UpdatedRefEventWatcher) Close() {
	close(rw.Ch)
	close(rw.repoIDCh)
}

func (n *Client) WatchUpdatedRefEvents(ctx context.Context, repoIDs []string, start uint64) *UpdatedRefEventWatcher {
	cursor := start
	logsTimer := time.NewTicker(5 * time.Second)

	rw := &UpdatedRefEventWatcher{
		Ch:       make(chan MaybeUpdatedRefEvent),
		repoIDs:  repoIDs,
		repoIDCh: make(chan string),
	}

	go func() {
		defer rw.Close()

		repoIDByHash := make(map[string]string)
		for _, repoID := range rw.repoIDs {
			hash := crypto.Keccak256([]byte(repoID))
			repoIDByHash[string(hash)] = repoID
		}

		for {
			evts, err := n.GetUpdatedRefEvents(ctx, rw.repoIDs, cursor, nil)
			if err != nil {
				rw.Ch <- MaybeUpdatedRefEvent{Error: err}
				return
			}

			for _, evt := range evts {
				hashStr := string(evt.RepoIDHash[:])
				evt.RepoID = repoIDByHash[hashStr]
				rw.Ch <- MaybeUpdatedRefEvent{Event: evt}
				cursor = evt.BlockNumber + 1
			}

			select {
			case <-logsTimer.C:
			case repoID := <-rw.repoIDCh:
				rw.repoIDs = append(rw.repoIDs, repoID)
				hash := crypto.Keccak256([]byte(repoID))
				repoIDByHash[string(hash)] = repoID
			case <-ctx.Done():
				return
			}
		}
	}()

	return rw
}

func (n *Client) GetUpdatedRefEvents(ctx context.Context, repoIDs []string, start uint64, end *uint64) ([]UpdatedRefEvent, error) {
	opts := &bind.FilterOpts{
		Context: ctx,
		Start:   start,
	}
	users := []common.Address{}
	refs := []string{}

	iter, err := n.protocolContract.ProtocolFilterer.FilterLogUpdateRef(opts, users, repoIDs, refs)
	if err != nil {
		return []UpdatedRefEvent{}, err
	}

	evts := make([]UpdatedRefEvent, 0)
	for iter.Next() {
		blockNumber := iter.Event.Raw.BlockNumber
		block, err := n.ethClient.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
		if err != nil {
			return []UpdatedRefEvent{}, err
		}

		evt := UpdatedRefEvent{
			Commit:      iter.Event.CommitHash,
			RefHash:     hex.EncodeToString(iter.Event.RefName[:]),
			RepoIDHash:  iter.Event.RepoID,
			Time:        block.Time().Uint64(),
			TxHash:      hex.EncodeToString(iter.Event.Raw.TxHash[:]),
			BlockNumber: blockNumber,
		}

		evts = append(evts, evt)
	}

	return evts, nil
}
