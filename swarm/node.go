package swarm

import (
	"context"

	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config"
	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm/nodeeth"
	"github.com/Conscience/protocol/swarm/nodeevents"
	"github.com/Conscience/protocol/swarm/nodep2p"
)

type Node struct {
	host        *nodep2p.Host
	ethClient   *nodeeth.Client
	eventBus    *nodeevents.EventBus
	repoManager *repo.Manager
	Config      *config.Config
	Shutdown    chan struct{}
}

func NewNode(ctx context.Context, cfg *config.Config) (*Node, error) {
	if cfg == nil {
		cfg = &config.DefaultConfig
	}

	eventBus := nodeevents.NewEventBus()

	// Initialize the on-disk repo manager
	repoManager, err := repo.NewManager(eventBus, cfg)
	if err != nil {
		return nil, err
	}

	// Initialize the Ethereum client
	ethClient, err := nodeeth.NewClient(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize Ethereum client")
	}

	username, err := ethClient.GetUsername(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch username from Ethereum smart contract")
	}
	log.SetField("username", username)

	// Initialize the p2p host
	host, err := nodep2p.NewHost(ctx, repoManager, ethClient, eventBus, cfg)
	if err != nil {
		return nil, err
	}

	n := &Node{
		host:        host,
		ethClient:   ethClient,
		repoManager: repoManager,
		eventBus:    eventBus,
		Config:      cfg,
		Shutdown:    make(chan struct{}),
	}

	return n, nil
}

func (n *Node) Close() error {
	close(n.Shutdown)

	err := n.host.Close()
	if err != nil {
		return errors.Wrap(err, "could not .Close P2P host")
	}

	err = n.ethClient.Close()
	if err != nil {
		return errors.Wrap(err, "could not .Close Ethereum client")
	}

	return nil
}

func (n *Node) P2PHost() *nodep2p.Host {
	return n.host
}

func (n *Node) RepoManager() *repo.Manager {
	return n.repoManager
}

func (n *Node) EthereumClient() *nodeeth.Client {
	return n.ethClient
}

func (n *Node) EventBus() *nodeevents.EventBus {
	return n.eventBus
}
