package config

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/Conscience/protocol/config/env"
	"github.com/Conscience/protocol/log"
)

type Config struct {
	Node      *NodeConfig      `toml:"node"`
	RPCClient *RPCClientConfig `toml:"rpcclient"`

	configPath string
	mu         sync.Mutex
}

type NodeConfig struct {
	PrivateKeyFile          string
	P2PListenAddr           string
	P2PListenPort           int
	BootstrapPeers          []string
	RPCListenNetwork        string
	RPCListenHost           string
	HTTPListenAddr          string
	HTTPUsername            string
	HTTPPassword            string
	EthereumHost            string
	ProtocolContract        string
	EthereumBIP39Seed       string
	ContentAnnounceInterval Duration
	ContentRequestInterval  Duration
	FindProviderTimeout     Duration
	LocalRepos              []string
	ReplicationRoot         string
	ReplicateRepos          []string
	KnownReplicators        []string
	ReplicateEverything     bool
	MaxConcurrentPeers      uint
}

type RPCClientConfig struct {
	Host string
}

var DefaultConfig = Config{
	Node: &NodeConfig{
		PrivateKeyFile: filepath.Join(env.HOME, ".axon.key"),
		P2PListenAddr:  "0.0.0.0",
		P2PListenPort:  1337,
		// RPCListenNetwork:        "unix",
		// RPCListenHost:           "/tmp/axon.sock",
		RPCListenNetwork:        "tcp",
		RPCListenHost:           "0.0.0.0:1338",
		HTTPListenAddr:          ":8081",
		HTTPUsername:            "admin",
		HTTPPassword:            "password",
		EthereumHost:            "http://geth.conscience.network:80",
		ProtocolContract:        "0x0501fc316bc5e138763c90f07dcb67a9a3b1e95d",
		EthereumBIP39Seed:       "",
		ContentAnnounceInterval: Duration(15 * time.Second),
		ContentRequestInterval:  Duration(15 * time.Second),
		FindProviderTimeout:     Duration(10 * time.Second),
		LocalRepos:              []string{},
		ReplicationRoot:         filepath.Join(env.HOME, "axon"),
		ReplicateRepos:          []string{},
		BootstrapPeers:          []string{"/dns4/node.conscience.network/tcp/1337/p2p/16Uiu2HAkvcdAFKchv9uGPeRguQubPxA4wrzyZDf1jLhhHiQ7qBbH"},
		KnownReplicators:        []string{"16Uiu2HAkvcdAFKchv9uGPeRguQubPxA4wrzyZDf1jLhhHiQ7qBbH"},
		ReplicateEverything:     false,
		MaxConcurrentPeers:      4,
	},
	RPCClient: &RPCClientConfig{
		// Host: "unix:///tmp/axon.sock",
		Host: "0.0.0.0:1338",
	},
}

func ReadConfig() (*Config, error) {
	configPath, err := filepath.Abs(filepath.Join(env.HOME, ".axonrc"))
	if err != nil {
		return nil, errors.Wrapf(err, "[config] calling filepath.Abs")
	}

	return ReadConfigAtPath(configPath)
}

func ReadConfigAtPath(configPath string) (*Config, error) {
	log.Debugf("[config] reading config at %v", configPath)

	// Copy the default config
	cfg := DefaultConfig

	// Decode the config file on top of the defaults
	_, err := toml.DecodeFile(configPath, &cfg)
	// If the file can't be found, we ignore the error.  Otherwise, return it.
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	cfg.configPath = configPath
	return &cfg, nil
}

func AttachToLogger(cfg *Config) {
	log.SetField("config path", cfg.configPath)
	log.SetField("config.Node.PrivateKeyFile", cfg.Node.PrivateKeyFile)
	log.SetField("config.Node.P2PListenAddr", cfg.Node.P2PListenAddr)
	log.SetField("config.Node.P2PListenPort", cfg.Node.P2PListenPort)
	log.SetField("config.Node.BootstrapPeers", cfg.Node.BootstrapPeers)
	log.SetField("config.Node.RPCListenNetwork", cfg.Node.RPCListenNetwork)
	log.SetField("config.Node.RPCListenHost", cfg.Node.RPCListenHost)
	log.SetField("config.Node.HTTPListenAddr", cfg.Node.HTTPListenAddr)
	log.SetField("config.Node.EthereumHost", cfg.Node.EthereumHost)
	log.SetField("config.Node.ProtocolContract", cfg.Node.ProtocolContract)
	log.SetField("config.Node.EthereumBIP39Seed", cfg.Node.EthereumBIP39Seed)
	log.SetField("config.Node.ContentAnnounceInterval", cfg.Node.ContentAnnounceInterval)
	log.SetField("config.Node.ContentRequestInterval", cfg.Node.ContentRequestInterval)
	log.SetField("config.Node.FindProviderTimeout", cfg.Node.FindProviderTimeout)
	log.SetField("config.Node.LocalRepos", cfg.Node.LocalRepos)
	log.SetField("config.Node.ReplicationRoot", cfg.Node.ReplicationRoot)
	log.SetField("config.Node.ReplicateRepos", cfg.Node.ReplicateRepos)
	log.SetField("config.Node.KnownReplicators", cfg.Node.KnownReplicators)
	log.SetField("config.Node.ReplicateEverything", cfg.Node.ReplicateEverything)
	log.SetField("config.RPCClient.Host", cfg.RPCClient.Host)
}

func (c *Config) Update(fn func() error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := fn()
	if err != nil {
		return err
	}

	return c.save()
}

func (c *Config) save() error {
	f, err := os.Create(c.configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	err = toml.NewEncoder(f).Encode(c)
	if err != nil {
		return err
	}
	return nil
}

type Duration time.Duration

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

func (d *Duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}
