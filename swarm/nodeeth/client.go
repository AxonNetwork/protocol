package nodeeth

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/libgit2/git2go"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/repo"
)

type Client struct {
	ethClient            *ethclient.Client
	protocolContract     *Protocol
	protocolContractAddr Address
	protocolContractABI  abi.ABI
	privateKey           *ecdsa.PrivateKey
	account              accounts.Account
	auth                 *bind.TransactOpts
	wallet               *Wallet
}

type Address = common.Address

func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	ethClient, err := ethclient.DialContext(ctx, cfg.Node.EthereumHost)
	if err != nil {
		return nil, err
	}

	protocolContractAddr := common.HexToAddress(cfg.Node.ProtocolContract)

	protocolContract, err := NewProtocol(protocolContractAddr, ethClient)
	if err != nil {
		return nil, err
	}

	protocolContractABI, err := abi.JSON(strings.NewReader(ProtocolABI))
	if err != nil {
		return nil, err
	}

	mnemonic, err := getOrCreateMnemonic(cfg)
	if err != nil {
		return nil, err
	}

	account, wallet, err := initAccount(mnemonic, "")
	if err != nil {
		return nil, err
	}

	prv, err := wallet.PrivateKey(account)
	if err != nil {
		return nil, err
	}

	auth := bind.NewKeyedTransactor(prv)

	return &Client{
		ethClient:            ethClient,
		protocolContract:     protocolContract,
		protocolContractAddr: protocolContractAddr,
		protocolContractABI:  protocolContractABI,
		privateKey:           prv,
		auth:                 auth,
		account:              account,
		wallet:               wallet,
	}, nil
}

func getOrCreateMnemonic(cfg *config.Config) (string, error) {
	if len(cfg.Node.EthereumBIP39Seed) > 0 && bip39.IsMnemonicValid(cfg.Node.EthereumBIP39Seed) {
		return cfg.Node.EthereumBIP39Seed, nil
	}
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", err
	}

	err = cfg.Update(func() error {
		cfg.Node.EthereumBIP39Seed = mnemonic
		return nil
	})
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}

func initAccount(mnemonic string, password string) (accounts.Account, *Wallet, error) {
	seed := bip39.NewSeed(mnemonic, password)
	wallet, err := NewWalletFromSeed(seed)
	if err != nil {
		return accounts.Account{}, nil, err
	}

	path := MustParseDerivationPath("m/44'/60'/0'/0/0")
	account, err := wallet.Derive(path, false)
	if err != nil {
		return accounts.Account{}, nil, err
	}

	return account, wallet, nil
}

func (n *Client) callOpts(ctx context.Context) *bind.CallOpts {
	return &bind.CallOpts{Context: ctx}
}

func (n *Client) transactOpts(ctx context.Context) *bind.TransactOpts {
	return &bind.TransactOpts{
		From:    n.auth.From,
		Signer:  n.auth.Signer,
		Context: ctx,
	}
}

func (n *Client) Close() error {
	n.ethClient.Close()
	return nil
}

func (n *Client) Address() common.Address {
	return n.account.Address
}

func (n *Client) SignHash(data []byte) ([]byte, error) {
	hash := crypto.Keccak256(data)
	return crypto.Sign(hash, n.privateKey)
}

func (n *Client) AddrFromSignedHash(data, sig []byte) (Address, error) {
	hash := crypto.Keccak256(data)
	pubkey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pubkey), nil
}

func (n *Client) GetUsername(ctx context.Context) (string, error) {
	addr, err := n.wallet.Address(n.account)
	if err != nil {
		return "", err
	}
	return n.protocolContract.UsernamesByAddress(n.callOpts(ctx), addr)
}

func (n *Client) EnsureUsername(ctx context.Context, username string) (*Transaction, error) {
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

func (n *Client) SetUsername(ctx context.Context, username string) (*Transaction, error) {
	opts := n.transactOpts(ctx)

	// Manually set a gas limit so that we get a clearer error when the tx fails
	opts.GasLimit = 73000

	tx, err := n.protocolContract.SetUsername(opts, username)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx, n.ethClient}, nil
}

func (n *Client) IsRepoIDRegistered(ctx context.Context, repoID string) (bool, error) {
	return n.protocolContract.RepoExists(n.callOpts(ctx), repoID)
}

func (n *Client) RegisterRepoID(ctx context.Context, repoID string) (*Transaction, error) {
	tx, err := n.protocolContract.CreateRepo(n.transactOpts(ctx), repoID)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx, n.ethClient}, nil
}

func (n *Client) UpdateRemoteRef(ctx context.Context, repoID string, refName string, oldCommitHash, newCommitHash git.Oid) (*Transaction, error) {
	tx, err := n.protocolContract.UpdateRef(n.transactOpts(ctx), repoID, refName, oldCommitHash, newCommitHash)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx, n.ethClient}, nil
}

func (n *Client) GetNumRemoteRefs(ctx context.Context, repoID string) (uint64, error) {
	num, err := n.protocolContract.NumRefs(n.callOpts(ctx), repoID)
	if err != nil {
		return 0, err
	}
	return num.Uint64(), nil
}

func (n *Client) GetRemoteRef(ctx context.Context, repoID string, refName string) (git.Oid, error) {
	ref, err := n.protocolContract.GetRef(n.callOpts(ctx), repoID, refName)
	if err != nil {
		// The geth codebase (which handles ABI unpacking) doesn't currently seem to understand a
		// transaction/call return value containing a Solidity require/assert error message
		// (as in `require(someCondition, "condition not met")`).  @@TODO?
		if strings.Contains(err.Error(), "abi: unmarshalling empty output") || strings.Contains(err.Error(), "abi: improperly formatted output") {
			return git.Oid{}, nil
		} else {
			return git.Oid{}, err
		}
	}
	return ref, nil
}

func (n *Client) GetRemoteRefs(ctx context.Context, repoID string, pageSize uint64, page uint64) (map[string]repo.Ref, uint64, error) {
	x, err := n.protocolContract.GetRefs(n.callOpts(ctx), repoID, big.NewInt(0).SetUint64(pageSize), big.NewInt(0).SetUint64(page))
	if err != nil {
		// The geth codebase (which handles ABI unpacking) doesn't currently seem to understand a
		// transaction/call return value containing a Solidity require/assert error message
		// (as in `require(someCondition, "condition not met")`).  @@TODO?
		if strings.Contains(err.Error(), "abi: unmarshalling empty output") || strings.Contains(err.Error(), "abi: improperly formatted output") {
			return map[string]repo.Ref{}, 0, nil
		} else {
			return nil, 0, err
		}
	}

	refs := map[string]repo.Ref{}

	var read int64
	for read < int64(len(x.Data)) {
		ref := repo.Ref{}

		nameLen := big.NewInt(0).SetBytes(x.Data[read : read+32]).Int64()
		read += 32
		ref.RefName = string(x.Data[read : read+nameLen])
		read += nameLen

		commitLen := big.NewInt(0).SetBytes(x.Data[read : read+32]).Int64()
		read += 32
		ref.CommitHash = hex.EncodeToString(x.Data[read : read+commitLen])
		read += commitLen

		refs[ref.RefName] = ref
	}

	return refs, x.Total.Uint64(), nil
}

func (n *Client) ForEachRemoteRef(ctx context.Context, repoID string, fn func(repo.Ref) (bool, error)) error {
	var page uint64
	var scanned uint64
	var total uint64
	var err error
	var refmap map[string]repo.Ref

	for {
		refmap, total, err = n.GetRemoteRefs(ctx, repoID, 10, page)
		if err != nil {
			return err
		}

		for _, ref := range refmap {
			proceed, err := fn(ref)
			if err != nil {
				return err
			} else if !proceed {
				return nil
			}
			scanned++
		}

		if scanned >= total {
			break
		}
		page++
	}
	return nil
}

// func (n *Client) GetRefLogs(ctx context.Context, repoID string) (map[string]uint64, error) {
// 	opts := &bind.FilterOpts{Context: ctx}
// 	users := []common.Address{}
// 	repoIDs := []string{repoID}
// 	refs := []string{}

// 	iter, err := n.protocolContract.ProtocolFilterer.FilterLogUpdateRef(opts, users, repoIDs, refs)
// 	if err != nil {
// 		return map[string]uint64{}, err
// 	}
// 	defer iter.Close()

// 	logs := make(map[string]uint64)
// 	for iter.Next() {
// 		block, err := n.ethClient.BlockByNumber(ctx, big.NewInt(int64(iter.Event.Raw.BlockNumber)))
// 		if err != nil {
// 			return map[string]uint64{}, err
// 		}
// 		logs[iter.Event.CommitHash] = block.Time().Uint64()
// 	}
// 	return logs, nil
// }

func (n *Client) SetRepoPublic(ctx context.Context, repoID string, isPublic bool) (*Transaction, error) {
	tx, err := n.protocolContract.SetPublic(n.transactOpts(ctx), repoID, isPublic)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx, n.ethClient}, nil
}

func (n *Client) IsRepoPublic(ctx context.Context, repoID string) (bool, error) {
	return n.protocolContract.IsRepoPublic(n.callOpts(ctx), repoID)
}

type UserType uint8

const (
	UserType_Admin  UserType = 0
	UserType_Puller UserType = 1
	UserType_Pusher UserType = 2
)

func (n *Client) GetRepoUsers(ctx context.Context, repoID string, whichUsers UserType, pageSize uint64, page uint64) ([]string, uint64, error) {
	x, err := n.protocolContract.GetRepoUsers(n.callOpts(ctx), repoID, uint8(whichUsers), big.NewInt(0).SetUint64(pageSize), big.NewInt(0).SetUint64(page))
	if err != nil {
		return nil, 0, err
	}

	users := []string{}

	var read int64
	for read < int64(len(x.Data)) {
		nameLen := big.NewInt(0).SetBytes(x.Data[read : read+32]).Int64()
		read += 32
		user := string(x.Data[read : read+nameLen])
		read += nameLen

		users = append(users, user)
	}

	return users, x.Total.Uint64(), nil
}

func (n *Client) AddressHasPullAccess(ctx context.Context, user Address, repoID string) (bool, error) {
	hasAccess, err := n.protocolContract.AddressHasPullAccess(n.callOpts(ctx), user, repoID)
	return hasAccess, err
}

type UserPermissions struct {
	Puller bool
	Pusher bool
	Admin  bool
}

func (n *Client) SetUserPermissions(ctx context.Context, repoID string, username string, perms UserPermissions) (*Transaction, error) {
	tx, err := n.protocolContract.SetUserPermissions(n.transactOpts(ctx), repoID, username, perms.Puller, perms.Pusher, perms.Admin)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx, n.ethClient}, nil
}

func (n *Client) CurrentBlock(ctx context.Context) (uint64, error) {
	header, err := n.ethClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}

	return header.Number.Uint64(), nil
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

func (c *Client) GetUpdatedRefEvents(ctx context.Context, repoIDs []string, start uint64, end *uint64) ([]UpdatedRefEvent, error) {
	opts := &bind.FilterOpts{
		Context: ctx,
		Start:   start,
	}
	users := []Address{}
	refs := []string{}

	iter, err := c.protocolContract.ProtocolFilterer.FilterLogUpdateRef(opts, users, repoIDs, refs)
	if err != nil {
		return nil, err
	}

	evts := make([]UpdatedRefEvent, 0)
	for iter.Next() {
		blockNumber := iter.Event.Raw.BlockNumber
		block, err := c.ethClient.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
		if err != nil {
			return nil, err
		}

		evt := UpdatedRefEvent{
			Commit:      hex.EncodeToString(iter.Event.CommitHash[:]),
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

// type UpdatedRefEvent struct {
// 	RepoID        string
// 	RefName       string
// 	OldCommitHash [20]byte
// 	NewCommitHash [20]byte
// }

// func (c *Client) WatchUpdatedRefEvents(ctx context.Context, repoIDs []string, fromBlock uint64) (<-chan UpdatedRefEvent, error) {
// 	chOut := make(chan UpdatedRefEvent)

// 	go func() {
// 		opts := &bind.WatchOpts{
// 			Start:   &fromBlock,
// 			Context: ctx,
// 		}

// 		sink := make(chan *ProtocolLogUpdateRef)

// 		subscription := c.protocolContract.ProtocolFilterer.WatchLogUpdateRef(opts, sink, nil, repoIDs, nil)
// 		defer subscription.Close()

// 		for e := range sink {
// 			tx, _, err := c.ethClient.TransactionByHash(ctx, e.Raw.TxHash)
// 			if err != nil {
// 				panic(err)
// 			}

// 			// function updateRef(string memory repoID, string memory refName, bytes20 oldCommitHash, bytes20 newCommitHash) public {
// 			result := struct {
// 				RepoID        string   `abi:"repoID"`
// 				RefName       string   `abi:"refName"`
// 				OldCommitHash [20]byte `abi:"oldCommitHash"`
// 				NewCommitHash [20]byte `abi:"newCommitHash"`
// 			}{}

// 			err = c.protocolContractABI.Methods["updateRef"].Inputs.Unpack(&result, tx.Data())
// 			if err != nil {
// 				panic(err)
// 			}

// 			chOut <- UpdatedRefEvent{
// 				RepoID:        result.RepoID,
// 				RefName:       result.RefName,
// 				OldCommitHash: result.OldCommitHash,
// 				NewCommitHash: result.NewCommitHash,
// 			}
// 		}
// 	}()

// 	return chOut
// }
