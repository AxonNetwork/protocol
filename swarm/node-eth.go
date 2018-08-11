package swarm

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	// log "github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"

	"../config"
	"./ethcontracts"
	"./hdwallet"
)

const (
	POLL_INTERVAL = 1000
)

type nodeETH struct {
	ethClient        *ethclient.Client
	protocolContract *ethcontracts.Protocol
	prv              *ecdsa.PrivateKey
	auth             *bind.TransactOpts
	account          accounts.Account
	wallet           *hdwallet.Wallet
}

func initETH(ctx context.Context, cfg *config.Config) (*nodeETH, error) {
	ethClient, err := ethclient.DialContext(ctx, cfg.Node.EthereumHost)
	if err != nil {
		return nil, err
	}

	protocolContract, err := ethcontracts.NewProtocol(common.HexToAddress(cfg.Node.ProtocolContract), ethClient)
	if err != nil {
		return nil, err
	}

	account, wallet, err := initAccount(cfg.Node.EthereumBIP39Seed, "")
	if err != nil {
		return nil, err
	}

	prv, err := wallet.PrivateKey(account)
	if err != nil {
		return nil, err
	}

	auth := bind.NewKeyedTransactor(prv)

	return &nodeETH{
		ethClient:        ethClient,
		protocolContract: protocolContract,
		prv:              prv,
		auth:             auth,
		account:          account,
		wallet:           wallet,
	}, nil
}

func initAccount(mnemonic string, password string) (accounts.Account, *hdwallet.Wallet, error) {
	seed := bip39.NewSeed(mnemonic, password)
	wallet, err := hdwallet.NewFromSeed(seed)
	if err != nil {
		return accounts.Account{}, nil, err
	}

	path := hdwallet.MustParseDerivationPath("m/44'/60'/0'/0/0")
	account, err := wallet.Derive(path, false)
	if err != nil {
		return accounts.Account{}, nil, err
	}

	return account, wallet, nil
}

func (n *nodeETH) callOpts(ctx context.Context) *bind.CallOpts {
	return &bind.CallOpts{Context: ctx}
}

func (n *nodeETH) transactOpts(ctx context.Context) *bind.TransactOpts {
	return &bind.TransactOpts{
		From:    n.auth.From,
		Signer:  n.auth.Signer,
		Context: ctx,
	}
}

func (n *nodeETH) Close() error {
	n.ethClient.Close()
	return nil
}

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

func (tx Transaction) Await(ctx context.Context) chan TxResult {
	ch := make(chan TxResult)
	hash := tx.Hash()
	go func() {
		for {
			select {
			case <-ctx.Done():
				ch <- TxResult{nil, errors.New("deadline expired before transaction was mined")}
				return

			default:
				receipt, err := tx.c.TransactionReceipt(ctx, hash)
				if err != nil && err != ethereum.NotFound {
					ch <- TxResult{nil, err}
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

func (n *nodeETH) GetUsername(ctx context.Context) (string, error) {
	addr, err := n.wallet.Address(n.account)
	if err != nil {
		return "", err
	}
	return n.protocolContract.UsernamesByAddress(n.callOpts(ctx), addr)
}

func (n *nodeETH) EnsureUsername(ctx context.Context, username string) (*Transaction, error) {
	un, err := n.GetUsername(ctx)
	if err != nil {
		return nil, err
	} else if len(un) > 0 && un != username {
		// already set correctly
		return nil, errors.New("username has already been set to something else")
	} else if un == username {
		return nil, nil
	}
	return n.SetUsername(ctx, username)
}

func (n *nodeETH) SetUsername(ctx context.Context, username string) (*Transaction, error) {
	tx, err := n.protocolContract.SetUsername(n.transactOpts(ctx), username)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx, n.ethClient}, nil
}

func (n *nodeETH) EnsureRepo(ctx context.Context, repoID string) (*Transaction, error) {
	exists, err := n.protocolContract.RepoExists(n.callOpts(ctx), repoID)
	if err != nil {
		return nil, err
	} else if exists {
		return nil, nil
	}
	return n.CreateRepo(ctx, repoID)
}

func (n *nodeETH) CreateRepo(ctx context.Context, repoID string) (*Transaction, error) {
	tx, err := n.protocolContract.CreateRepo(n.transactOpts(ctx), repoID)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx, n.ethClient}, nil
}

func (n *nodeETH) UpdateRef(ctx context.Context, repoID string, refName string, commitHash string) (*Transaction, error) {
	tx, err := n.protocolContract.UpdateRef(n.transactOpts(ctx), repoID, refName, commitHash)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx, n.ethClient}, nil
}

func (n *nodeETH) GetNumRefs(ctx context.Context, repoID string) (int64, error) {
	num, err := n.protocolContract.NumRefs(n.callOpts(ctx), repoID)
	if err != nil {
		return 0, err
	}
	return num.Int64(), nil
}

func (n *nodeETH) GetRefs(ctx context.Context, repoID string, page int64) map[string]Ref {
	refs := map[string]Ref{}
	refsBytes, err := n.protocolContract.GetRefs(n.callOpts(ctx), repoID, big.NewInt(page))
	if err != nil {
		return refs
	}

	var read int64
	for read < int64(len(refsBytes)) {
		ref := Ref{}

		ref.NameLen = big.NewInt(0).SetBytes(refsBytes[read : read+32]).Int64()
		read += 32
		ref.Name = string(refsBytes[read : read+ref.NameLen])
		read += ref.NameLen

		ref.CommitLen = big.NewInt(0).SetBytes(refsBytes[read : read+32]).Int64()
		read += 32
		ref.Commit = string(refsBytes[read : read+ref.CommitLen])
		read += ref.CommitLen

		refs[ref.Name] = ref
	}

	return refs
}

func (n *nodeETH) AddressHasPullAccess(ctx context.Context, user common.Address, repoID string) (bool, error) {
	hasAccess, err := n.protocolContract.AddressHasPullAccess(n.callOpts(ctx), user, repoID)
	return hasAccess, err
}
