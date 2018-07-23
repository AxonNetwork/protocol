package swarm

import (
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"
)

type Config struct {
	Node NodeConfig
}

type NodeConfig struct {
	P2PListenPort       int
	RPCListenNetwork    string
	RPCListenHost       string
	AnnounceInterval    duration
	FindProviderTimeout duration
}

var defaultNodeConfig = NodeConfig{
	P2PListenPort:       1337,
	RPCListenNetwork:    "tcp",
	RPCListenHost:       "127.0.0.1:1338",
	AnnounceInterval:    duration(10 * time.Second),
	FindProviderTimeout: duration(10 * time.Second),
}

func ReadConfig() (*Config, error) {
	dir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(dir, ".consciencerc")

	cfg := &Config{}
	_, err = toml.DecodeFile(configPath, cfg)
	if err != nil {
		// no-op, assume file wasn't there
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Node.P2PListenPort == 0 {
		cfg.Node.P2PListenPort = defaultNodeConfig.P2PListenPort
	}
	if cfg.Node.RPCListenNetwork == "" {
		cfg.Node.RPCListenNetwork = defaultNodeConfig.RPCListenNetwork
	}
	if cfg.Node.RPCListenHost == "" {
		cfg.Node.RPCListenHost = defaultNodeConfig.RPCListenHost
	}
	if cfg.Node.AnnounceInterval == 0 {
		cfg.Node.AnnounceInterval = defaultNodeConfig.AnnounceInterval
	}
	if cfg.Node.FindProviderTimeout == 0 {
		cfg.Node.FindProviderTimeout = defaultNodeConfig.FindProviderTimeout
	}
}

type duration time.Duration

func (d *duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = duration(dur)
	return nil
}
