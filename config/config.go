package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
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
	KnownReplicators        []string
	ReplicateEverything     bool
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
		PrivateKeyFile: filepath.Join(os.Getenv("HOME"), ".conscience.key"),
		P2PListenAddr:  "0.0.0.0",
		P2PListenPort:  1337,
		// RPCListenNetwork:        "unix",
		// RPCListenHost:           "/tmp/conscience.sock",
		RPCListenNetwork:        "tcp",
		RPCListenHost:           "0.0.0.0:1338",
		HTTPListenAddr:          ":8081",
		EthereumHost:            "http://geth.conscience.network:80",
		ProtocolContract:        "0x0501fc316bc5e138763c90f07dcb67a9a3b1e95d",
		EthereumBIP39Seed:       "",
		ContentAnnounceInterval: Duration(15 * time.Second),
		ContentRequestInterval:  Duration(15 * time.Second),
		FindProviderTimeout:     Duration(10 * time.Second),
		LocalRepos:              []string{},
		ReplicationRoot:         filepath.Join(os.Getenv("HOME"), "conscience"),
		ReplicateRepos:          []string{},
		BootstrapPeers:          []string{"/dns4/node.conscience.network/tcp/1337/p2p/16Uiu2HAkvcdAFKchv9uGPeRguQubPxA4wrzyZDf1jLhhHiQ7qBbH"},
		KnownReplicators:        []string{"16Uiu2HAkvcdAFKchv9uGPeRguQubPxA4wrzyZDf1jLhhHiQ7qBbH"},
		ReplicateEverything:     false,
	},
	RPCClient: &RPCClientConfig{
		// Host: "unix:///tmp/conscience.sock",
		Host: "0.0.0.0:1338",
	},
}

func ReadConfig() (*Config, error) {
	dir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}
	log.Printf("home dir = %v", dir)

	configPath, err := filepath.Abs(filepath.Join(dir, ".consciencerc"))
	if err != nil {
		log.Printf("error filepath.Abs = %v", err)
		return nil, err
	}

	return ReadConfigAtPath(configPath)
}

func ReadConfigAtPath(configPath string) (*Config, error) {
	log.Warnf("reading config at %v", configPath)
	cfg := DefaultConfig

	bs, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Errorf("could not read config!")
	} else {
		log.Printf("reaad config.  contents = %v", string(bs))
	}

	_, err = toml.DecodeFile(configPath, &cfg)
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
