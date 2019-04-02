
# Protocol

## Installation

1. Install Go

2. Run `./make.sh -n` to compile native binaries

3. Add a file called `.axonrc` to your home directory containing the following information:

```
[user]
Username = "bryn"

[node]
P2PListenPort       = 1337
P2PListenAddr       = "0.0.0.0"
RPCListenNetwork    = "unix"
RPCListenHost       = "/tmp/axon.sock"
EthereumHost        = "https://rinkeby.infura.io/<your infura API key>"
ProtocolContract    = "0xe31e0e2e114bcb7be568e648746b12a19757307a"
EthereumBIP39Seed   = "your 12 word seed phrase goes here"
AnnounceInterval    = "5s"
FindProviderTimeout = "10s"
ReplicateRepos      = []
BootstrapPeers      = []

[rpcclient]
Host    = "unix:///tmp/axon.sock"
```

4. Run the node with `CONSOLE_LOGGING=1 ./build/native/axon-node`.

## Config notes

`RPCClient.Host` can be things like:
- `unix:///tmp/axon.sock`
- `0.0.0.0:1338`

## Environment variables

- `CONSOLE_LOGGING`: Controls whether the logger prints to stderr.  Any non-empty value will be treated as true.
- `BUGSNAG_ENABLED`: Controls whether the logger sends errors to Bugsnag.  Any non-empty value will be treated as true.
- `LOG_CHILD_PROCS`: Controls whether the logger logs stdout and stderr of any child process started by the node (primarily calls to git).  Any non-empty value will be treated as true.
- `CONSCIENCE_BINARIES_PATH`: Specifies the location of the Conscience binaries (git-remote-axon, axon_decode, axon_encode, axon_diff).  This location is added to the `PATH` of any child process invoked by the node so that Git can find these helpers.

## Weird bugs and other errata

- Keep an eye out for line endings.  The git remote helper (git-remote-axon) was failing on Windows due to the presence of `\r` characters in the stream.


## Helpful profiling commands

```sh
pprof -http :1227 ~/projects/conscience/protocol/build/native/axon-node http://localhost:6060/debug/pprof/profile?seconds=18
```


## libgit2

- Only adds 4mb to compiled binaries
- Is compiled in statically
- Is currently based on v27, with one extra commit cherry-picked from https://github.com/lhchavez/git2go (`122ccfadea1e219c819adf1e62534f0b869d82a3`) and one extra commit to bring the first commit up to date (`81a759a2593aeb28b7bb07439da9796489bfe3bb`).  These extra commits add support for packfile writing and indexing.
- Gives cryptic errors when trying to compile if the libgit2 version is not correctly matched to the git2go version.  libgit2 is correctly vendored on git2go's versioned branches (v27, v28, etc), so if you see these errors, you're probably doing something weird.
- Resides in `vendor/github.com/libgit2/git2go`, but is not actually "vendored" in the traditional Go sense.  It has to be compiled and prepared manually (see the `build_libgit2` function in `make.sh` for more information).  Because of the weird "vendoring" situation, and because we're using Go modules, we've added the following line to `go.mod`, which tells Go how to find the library:

```
replace github.com/libgit2/git2go => ./vendor/github.com/libgit2/git2go
```

