package nodeeth

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

const (
	// @@TODO: make this configurable
	POLL_INTERVAL = 1000
)

type Transaction struct {
	*types.Transaction
	c interface {
		TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
	}
}

type TxResult struct {
	Receipt *types.Receipt
	Err     error
}

func (tx Transaction) Await(ctx context.Context) <-chan TxResult {
	ch := make(chan TxResult)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ch <- TxResult{nil, errors.WithStack(ctx.Err())}
				return

			default:
				receipt, err := tx.c.TransactionReceipt(ctx, tx.Hash())
				if err != nil && strings.Contains(err.Error(), "missing required field 'transactionHash' for Log") {
					// This means we're talking to a Parity node and we don't have the receipt yet.
					// no-op.

				} else if err != nil && err != ethereum.NotFound {
					ch <- TxResult{nil, errors.WithStack(err)}
					return

				} else if receipt != nil {
					ch <- TxResult{receipt, nil}
					return
				}
				time.Sleep(time.Millisecond * POLL_INTERVAL)
			}
		}
	}()

	return ch
}
