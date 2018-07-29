package swarm

import (
	"context"
	"math/big"

	log "github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"../config"
	"./ethcontracts"
)

type nodeETH struct {
	ethClient        *ethclient.Client
	protocolContract *ethcontracts.Protocol
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

	return &nodeETH{
		ethClient:        ethClient,
		protocolContract: protocolContract,
	}, nil
}

func (n *nodeETH) Close() error {
	n.ethClient.Close()
	return nil
}

func (n *nodeETH) GetRefsX(ctx context.Context, repoID string, page int64) (string, error) {
	refs, err := n.protocolContract.GetRefs(&bind.CallOpts{Context: ctx}, repoID, big.NewInt(page))
	if err != nil {
		return "", err
	}
	return refs, nil
}
