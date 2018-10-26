package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"

	"github.com/Conscience/protocol/log"
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

var HOME = func() string {
	dir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	log.SetField("Calculated HOME", dir)
	return dir
}()

var DefaultConfig = Config{
	User: &UserConfig{
		Username: "nobody",
		JWT:      "",
	},
	Node: &NodeConfig{
		PrivateKeyFile: filepath.Join(HOME, ".conscience.key"),
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
		ReplicationRoot:         filepath.Join(HOME, "conscience"),
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
		log.Errorf("ReadConfig error on filepath.Abs: %v", err)
		return nil, err
	}

	return ReadConfigAtPath(configPath)
}

func ReadConfigAtPath(configPath string) (*Config, error) {
	log.Debugf("reading config at %v", configPath)
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

func AttachToLogger(cfg *Config) {
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
