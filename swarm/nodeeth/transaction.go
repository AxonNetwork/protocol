package nodeeth

import (
	"context"
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
				if err != nil && err != ethereum.NotFound {
					ch <- TxResult{nil, errors.WithStack(err)}
					return
				} else if receipt != nil {
					ch <- TxResult{receipt, nil}
					return
				}
				time.Sleep(time.Millisecond * POLL_INTERVAL)
			}

			// @@TODO: make configurable
			time.Sleep(2 * time.Second)
		}
	}()

	return ch
}
