package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/Conscience/protocol/config/env"
	"github.com/Conscience/protocol/log"
)

type Config struct {
	Node      *NodeConfig      `yaml:"Node"`
	RPCClient *RPCClientConfig `yaml:"RPCClient"`

	configPath string     `yaml:"-"`
	mu         sync.Mutex `yaml:"-"`
}

type NodeConfig struct {
	PrivateKeyFile          string                       `yaml:"PrivateKeyFile"`
	P2PListenAddr           string                       `yaml:"P2PListenAddr"`
	P2PListenPort           int                          `yaml:"P2PListenPort"`
	BootstrapPeers          []string                     `yaml:"BootstrapPeers"`
	RPCListenNetwork        string                       `yaml:"RPCListenNetwork"`
	RPCListenHost           string                       `yaml:"RPCListenHost"`
	HTTPListenAddr          string                       `yaml:"HTTPListenAddr"`
	HTTPUsername            string                       `yaml:"HTTPUsername"`
	HTTPPassword            string                       `yaml:"HTTPPassword"`
	EthereumHost            string                       `yaml:"EthereumHost"`
	ProtocolContract        string                       `yaml:"ProtocolContract"`
	EthereumBIP39Seed       string                       `yaml:"EthereumBIP39Seed"`
	ContentAnnounceInterval Duration                     `yaml:"ContentAnnounceInterval"`
	ContentRequestInterval  Duration                     `yaml:"ContentRequestInterval"`
	FindProviderTimeout     Duration                     `yaml:"FindProviderTimeout"`
	LocalRepos              []string                     `yaml:"LocalRepos"`
	ReplicationRoot         string                       `yaml:"ReplicationRoot"`
	ReplicationPolicies     map[string]ReplicationPolicy `yaml:"ReplicationPolicies"`
	MaxConcurrentPeers      uint                         `yaml:"MaxConcurrentPeers"`
}

type RPCClientConfig struct {
	Host string `yaml:"Host"`
}

type ReplicationPolicy struct {
	MaxBytes int64 `yaml:"MaxBytes"`
	Bare     bool  `yaml:"Bare"`
	// Shallow  bool
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
		EthereumHost:            "http://hera.axon.science:8545",
		ProtocolContract:        "0x80e1E5bC2d6933a4C7F832036e140c0609dDC997",
		EthereumBIP39Seed:       "",
		ContentAnnounceInterval: Duration(15 * time.Second),
		ContentRequestInterval:  Duration(15 * time.Second),
		FindProviderTimeout:     Duration(10 * time.Second),
		LocalRepos:              []string{},
		ReplicationRoot:         filepath.Join(env.HOME, "axon"),
		BootstrapPeers: []string{
			"/dns4/jupiter.axon.science/tcp/1337/p2p/16Uiu2HAm4cL1W1yHcsQuDp9R19qeyAewekCdqyVM39WMykjVL2mt",
			"/dns4/saturn.axon.science/tcp/1337/p2p/16Uiu2HAkvVjg6mskzSQc6RKYY9YKFd1NQyTmWA8Cif7x5DAtAkFb",
		},
		ReplicationPolicies: map[string]ReplicationPolicy{},
		MaxConcurrentPeers:  4,
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

	bs, err := ioutil.ReadFile(configPath)
	// If the file can't be found, we ignore the error.  Otherwise, return it.
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Decode the config file on top of the defaults
	err = yaml.Unmarshal(bs, &cfg)
	if err != nil {
		return nil, err
	}

	bs, err = yaml.Marshal(cfg)
	if err != nil {
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
	log.SetField("config.Node.ReplicationPolicies", cfg.Node.ReplicationPolicies)
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

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)

	err = encoder.Encode(c)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) Path() string {
	return c.configPath
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
