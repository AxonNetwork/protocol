package config

import (
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"
)

type Config struct {
	Node      NodeConfig
	RPCClient RPCClientConfig
}

type NodeConfig struct {
	P2PListenPort       int
	RPCListenNetwork    string
	RPCListenHost       string
	AnnounceInterval    Duration
	FindProviderTimeout Duration
}

type RPCClientConfig struct {
	Network string
	Host    string
}

var DefaultConfig = Config{
	Node: NodeConfig{
		P2PListenPort:       1337,
		RPCListenNetwork:    "tcp",
		RPCListenHost:       "127.0.0.1:1338",
		AnnounceInterval:    Duration(10 * time.Second),
		FindProviderTimeout: Duration(10 * time.Second),
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
	if err != nil {
		// no-op, assume file wasn't there
	}
	return &cfg, nil
}

type Duration time.Duration

func (d *Duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}
