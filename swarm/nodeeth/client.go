package nodeeth

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"

	"../../config"
	"../wire"
)

type Client struct {
	ethClient        *ethclient.Client
	protocolContract *Protocol
	privateKey       *ecdsa.PrivateKey
	account          accounts.Account
	auth             *bind.TransactOpts
	wallet           *Wallet
}

func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	ethClient, err := ethclient.DialContext(ctx, cfg.Node.EthereumHost)
	if err != nil {
		return nil, err
	}

	protocolContract, err := NewProtocol(common.HexToAddress(cfg.Node.ProtocolContract), ethClient)
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

	return &Client{
		ethClient:        ethClient,
		protocolContract: protocolContract,
		privateKey:       prv,
		auth:             auth,
		account:          account,
		wallet:           wallet,
	}, nil
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

func (n *Client) AddrFromSignedHash(data, sig []byte) (common.Address, error) {
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
	tx, err := n.protocolContract.SetUsername(n.transactOpts(ctx), username)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx, n.ethClient}, nil
}

func (n *Client) EnsureRepoIDRegistered(ctx context.Context, repoID string) (*Transaction, error) {
	exists, err := n.protocolContract.RepoExists(n.callOpts(ctx), repoID)
	if err != nil {
		return nil, err
	} else if exists {
		return nil, nil
	}
	return n.RegisterRepoID(ctx, repoID)
}

func (n *Client) RegisterRepoID(ctx context.Context, repoID string) (*Transaction, error) {
	tx, err := n.protocolContract.CreateRepo(n.transactOpts(ctx), repoID)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx, n.ethClient}, nil
}

func (n *Client) UpdateRef(ctx context.Context, repoID string, refName string, commitHash string) (*Transaction, error) {
	tx, err := n.protocolContract.UpdateRef(n.transactOpts(ctx), repoID, refName, commitHash)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx, n.ethClient}, nil
}

func (n *Client) GetNumRefs(ctx context.Context, repoID string) (uint64, error) {
	num, err := n.protocolContract.NumRefs(n.callOpts(ctx), repoID)
	if err != nil {
		return 0, err
	}
	return num.Uint64(), nil
}

func (n *Client) GetRef(ctx context.Context, repoID string, refName string) (string, error) {
	return n.protocolContract.GetRef(n.callOpts(ctx), repoID, refName)
}

func (n *Client) GetRefs(ctx context.Context, repoID string, page int64) (map[string]wire.Ref, error) {
	refs := map[string]wire.Ref{}
	refsBytes, err := n.protocolContract.GetRefs(n.callOpts(ctx), repoID, big.NewInt(page))
	if err != nil {
		return nil, err
	}

	var read int64
	for read < int64(len(refsBytes)) {
		ref := wire.Ref{}

		nameLen := big.NewInt(0).SetBytes(refsBytes[read : read+32]).Int64()
		read += 32
		ref.RefName = string(refsBytes[read : read+nameLen])
		read += nameLen

		commitLen := big.NewInt(0).SetBytes(refsBytes[read : read+32]).Int64()
		read += 32
		ref.CommitHash = string(refsBytes[read : read+commitLen])
		read += commitLen

		refs[ref.RefName] = ref
	}

	return refs, nil
}

func (n *Client) AddressHasPullAccess(ctx context.Context, user common.Address, repoID string) (bool, error) {
	hasAccess, err := n.protocolContract.AddressHasPullAccess(n.callOpts(ctx), user, repoID)
	return hasAccess, err
}
