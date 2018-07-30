package swarm

import (
	"context"
	"math/big"

	// log "github.com/sirupsen/logrus"

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
