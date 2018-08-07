
# Protocol

## Installation

1. Install Go

2. Run `make && make install`

3. Add a file called `.consciencerc` to your home directory containing the following information:

```
[user]
Username = "bryn"

[node]
P2PListenPort       = 1337
P2PListenAddr       = "0.0.0.0"
RPCListenNetwork    = "unix"
RPCListenHost       = "/tmp/conscience.sock"
EthereumHost        = "https://rinkeby.infura.io/<your infura API key>"
ProtocolContract    = "0xe31e0e2e114bcb7be568e648746b12a19757307a"
EthereumBIP39Seed   = "your 12 word seed phrase goes here"
AnnounceInterval    = "5s"
FindProviderTimeout = "10s"
ReplicateRepos      = []
BootstrapPeers      = []

[rpcclient]
Network = "unix"
Host    = "/tmp/conscience.sock"
```

4. Run the node with `conscience-node`.

