package swarm

import (
	"context"
	"math/big"

	// log "github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tyler-smith/go-bip39"

	"../config"
	"./ethcontracts"
	"./hdwallet"
)

type nodeETH struct {
	ethClient        *ethclient.Client
	protocolContract *ethcontracts.Protocol
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

	account, wallet, err := initAccount(
		"candy maple cake sugar pudding cream honey rich smooth crumble sweet treat",
		"")
	if err != nil {
		return nil, err
	}

	sk, err := wallet.PrivateKey(account)
	if err != nil {
		return nil, err
	}

	auth := bind.NewKeyedTransactor(sk)

	return &nodeETH{
		ethClient:        ethClient,
		protocolContract: protocolContract,
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

func (n *nodeETH) Close() error {
	n.ethClient.Close()
	return nil
}

func (n *nodeETH) CheckAndSetUsername(ctx context.Context, username string) (*types.Transaction, error) {
	un, err := n.GetUsername(ctx)
	if err != nil {
		return nil, err
	}
	if un == username {
		// already set correctly
		return nil, nil
	}
	tx, err := n.SetUsername(username)
	return tx, err
}

func (n *nodeETH) GetUsername(ctx context.Context) (string, error) {
	addr, err := n.wallet.Address(n.account)
	if err != nil {
		return "", err
	}

	username, err := n.protocolContract.UsernamesByAddress(&bind.CallOpts{Context: ctx}, addr)
	return username, err
}

func (n *nodeETH) SetUsername(username string) (*types.Transaction, error) {
	tx, err := n.protocolContract.SetUsername(n.auth, username)
	return tx, err
}

func (n *nodeETH) GetRefsX(ctx context.Context, repoID string, page int64) ([]Ref, error) {
	refsBytes, err := n.protocolContract.GetRefs(&bind.CallOpts{Context: ctx}, repoID, big.NewInt(page))
	if err != nil {
		return nil, err
	}

	refs := []Ref{}

	var read int64
	for read < int64(len(refsBytes)) {
		ref := Ref{}

		nameLen := big.NewInt(0).SetBytes(refsBytes[read : read+32]).Int64()
		read += 32
		ref.Name = string(refsBytes[read : read+nameLen])
		read += nameLen

		commitLen := big.NewInt(0).SetBytes(refsBytes[read : read+32]).Int64()
		read += 32
		ref.Commit = string(refsBytes[read : read+commitLen])
		read += commitLen

		refs = append(refs, ref)
	}

	return refs, nil
}
