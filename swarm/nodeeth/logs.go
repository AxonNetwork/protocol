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

type MaybeRefLog struct {
	Log   RefLog
	Error error
}

type RefLog struct {
	Commit      string
	RefHash     string
	RepoIDHash  common.Hash
	RepoID      string
	TxHash      string
	Time        uint64
	BlockNumber uint64
}

type RefLogWatcher struct {
	Ch       chan MaybeRefLog
	repoIDs  []string
	repoIDCh chan string
}

func (rw *RefLogWatcher) AddRepo(repoID string) {
	rw.repoIDCh <- repoID
}

func (rw *RefLogWatcher) Close() {
	close(rw.Ch)
	close(rw.repoIDCh)
}

func (n *Client) WatchRefLogs(ctx context.Context, repoIDs []string, start uint64) *RefLogWatcher {

	cursor := start
	logsTimer := time.NewTicker(5 * time.Second)

	rw := &RefLogWatcher{
		Ch:       make(chan MaybeRefLog),
		repoIDs:  repoIDs,
		repoIDCh: make(chan string),
	}

	// get logs
	go func() {
		defer rw.Close()

		repoIDByHash := make(map[string]string)
		for _, repoID := range rw.repoIDs {
			hash := crypto.Keccak256([]byte(repoID))
			repoIDByHash[string(hash)] = repoID
		}

		for {
			logs, err := n.GetRefLogs(ctx, rw.repoIDs, cursor, nil)
			if err != nil {
				rw.Ch <- MaybeRefLog{Error: err}
				return
			}

			for _, reflog := range logs {
				hashStr := string(reflog.RepoIDHash[:])
				reflog.RepoID = repoIDByHash[hashStr]
				rw.Ch <- MaybeRefLog{Log: reflog}
				cursor = reflog.BlockNumber + 1
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

func (n *Client) GetRefLogs(ctx context.Context, repoIDs []string, start uint64, repoIDByHash map[string]string) ([]RefLog, error) {
	opts := &bind.FilterOpts{
		Context: ctx,
		Start:   start,
	}
	users := []common.Address{}
	refs := []string{}

	iter, err := n.protocolContract.ProtocolFilterer.FilterLogUpdateRef(opts, users, repoIDs, refs)
	if err != nil {
		return []RefLog{}, err
	}

	logs := make([]RefLog, 0)
	for iter.Next() {
		blockNumber := iter.Event.Raw.BlockNumber
		block, err := n.ethClient.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
		if err != nil {
			return []RefLog{}, err
		}

		// logs[iter.Event.CommitHash] = block.Time().Uint64()
		reflog := RefLog{
			Commit:      iter.Event.CommitHash,
			RefHash:     hex.EncodeToString(iter.Event.RefName[:]),
			RepoIDHash:  iter.Event.RepoID,
			Time:        block.Time().Uint64(),
			TxHash:      hex.EncodeToString(iter.Event.Raw.TxHash[:]),
			BlockNumber: blockNumber,
		}

		logs = append(logs, reflog)
	}

	return logs, nil
}
