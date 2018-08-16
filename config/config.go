package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"
)

type Config struct {
	User      UserConfig      `toml:"user"`
	Node      NodeConfig      `toml:"node"`
	RPCClient RPCClientConfig `toml:"rpcclient"`

	configPath string
}

type UserConfig struct {
	Username string
}

type NodeConfig struct {
	P2PListenAddr       string
	P2PListenPort       int
	BootstrapPeers      []string
	RPCListenNetwork    string
	RPCListenHost       string
	EthereumHost        string
	ProtocolContract    string
	EthereumBIP39Seed   string
	AnnounceInterval    Duration
	FindProviderTimeout Duration
	LocalRepos          []string
	ReplicationRoot     string
	ReplicateRepos      []string
}

type RPCClientConfig struct {
	Network string
	Host    string
}

var DefaultConfig = Config{
	User: UserConfig{
		Username: "nobody",
	},
	Node: NodeConfig{
		P2PListenAddr:       "0.0.0.0",
		P2PListenPort:       1337,
		RPCListenNetwork:    "tcp",
		RPCListenHost:       "127.0.0.1:1338",
		EthereumHost:        "http://127.0.0.1:8545",
		ProtocolContract:    "",
		EthereumBIP39Seed:   "",
		AnnounceInterval:    Duration(10 * time.Second),
		FindProviderTimeout: Duration(10 * time.Second),
		ReplicationRoot:     "/tmp/repos", // @@TODO: probably a better choice for this
		ReplicateRepos:      []string{},
		BootstrapPeers:      []string{},
	},
	RPCClient: RPCClientConfig{
		Network: "tcp",
		Host:    "127.0.0.1:1338",
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

func (c *Config) Save() error {
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
