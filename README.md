
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
Host    = "unix:///tmp/conscience.sock"
```

4. Run the node with `conscience-node`.

## Config notes

`RPCClient.Host` can be things like:
- `unix:///tmp/conscience.sock`
- `0.0.0.0:1338`

## Environment variables

- `ENABLE_CONSOLE_LOGGING`: Controls whether the logger prints to stderr.  Any non-empty value will be treated as true.
- `BUGSNAG_ENABLED`: Controls whether the logger sends errors to Bugsnag.  Any non-empty value will be treated as true.
- `LOG_CHILD_PROCS`: Controls whether the logger logs stdout and stderr of any child process started by the node (primarily calls to git).  Any non-empty value will be treated as true.
- `CONSCIENCE_BINARIES_PATH`: Specifies the location of the Conscience binaries (git-remote-conscience, conscience_decode, conscience_encode, conscience_diff).  This location is added to the `PATH` of any child process invoked by the node so that Git can find these helpers.

## Weird bugs and other errata

- The `logrus` logging package seems to lock up completely when the logger is set to output to stderr and the node is being run from inside the Electron app.  A few initial messages will be sent out, and then everything will deadlock.  As a result, we've temporarily added the `ENABLE_CONSOLE_LOGGING` environment variable so that we can still get console logging during debug or CLI sessions.

- Keep an eye out for line endings.  The git remote helper (git-remote-conscience) was failing on Windows due to the presence of `\r` characters in the stream.



