package config

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"
)

type Config struct {
	User      *UserConfig      `toml:"user"`
	Node      *NodeConfig      `toml:"node"`
	RPCClient *RPCClientConfig `toml:"rpcclient"`

	configPath string
	mu         sync.Mutex
}

type UserConfig struct {
	Username string
	JWT      string
}

type NodeConfig struct {
	PrivateKeyFile          string
	P2PListenAddr           string
	P2PListenPort           int
	BootstrapPeers          []string
	RPCListenNetwork        string
	RPCListenHost           string
	HTTPListenAddr          string
	EthereumHost            string
	ProtocolContract        string
	EthereumBIP39Seed       string
	ContentAnnounceInterval Duration
	ContentRequestInterval  Duration
	FindProviderTimeout     Duration
	LocalRepos              []string
	ReplicationRoot         string
	ReplicateRepos          []string
}

type RPCClientConfig struct {
	Host string
}

var DefaultConfig = Config{
	User: &UserConfig{
		Username: "nobody",
		JWT:      "",
	},
	Node: &NodeConfig{
		PrivateKeyFile:          filepath.Join(os.Getenv("HOME"), ".conscience.key"),
		P2PListenAddr:           "0.0.0.0",
		P2PListenPort:           1337,
		RPCListenNetwork:        "unix",
		RPCListenHost:           "/tmp/conscience.sock",
		HTTPListenAddr:          ":8081",
		EthereumHost:            "http://127.0.0.1:8545",
		ProtocolContract:        "",
		EthereumBIP39Seed:       "",
		ContentAnnounceInterval: Duration(15 * time.Second),
		ContentRequestInterval:  Duration(15 * time.Second),
		FindProviderTimeout:     Duration(10 * time.Second),
		ReplicationRoot:         "/tmp/repos", // @@TODO: probably a better choice for this
		ReplicateRepos:          []string{},
		BootstrapPeers:          []string{},
	},
	RPCClient: &RPCClientConfig{
		Host: "unix:///tmp/conscience.sock",
	},
}

func ReadConfig() (*Config, error) {
	dir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(dir, ".consciencerc")

	return ReadConfigAtPath(configPath)
}

func ReadConfigAtPath(configPath string) (*Config, error) {
	cfg := DefaultConfig

	_, err := toml.DecodeFile(configPath, &cfg)
	// If the file can't be found, we ignore the error.  Otherwise, return it.
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	cfg.configPath = configPath
	return &cfg, nil
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
