module github.com/Conscience/protocol

replace github.com/libgit2/git2go => ./vendor/github.com/libgit2/git2go

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/Shopify/logrus-bugsnag v0.0.0-20171204204709-577dee27f20d
	github.com/ThomsonReutersEikon/go-ntlm v0.0.0-20181130171125-cf23bd1ecf18 // indirect
	github.com/aclements/go-rabin v0.0.0-20170911142644-d0b643ea1a4c
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/allegro/bigcache v1.1.0 // indirect
	github.com/aristanetworks/goarista v0.0.0-20181109020153-5faa74ffbed7 // indirect
	github.com/brynbellomy/debugcharts v0.0.0-20180826220201-c3f57e57ea6f
	github.com/btcsuite/btcd v0.0.0-20181130015935-7d2daa5bfef2
	github.com/btcsuite/btcutil v0.0.0-20180706230648-ab6388e0c60a
	github.com/btcsuite/goleveldb v1.0.0 // indirect
	github.com/bugsnag/bugsnag-go v1.4.0
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/coreos/go-semver v0.2.0 // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/ethereum/go-ethereum v1.8.23
	github.com/fd/go-nat v1.0.0 // indirect
	github.com/git-lfs/git-lfs v0.0.0-20190228200031-78645419de54
	github.com/git-lfs/wildmatch v1.0.2
	github.com/gofrs/uuid v3.1.0+incompatible // indirect
	github.com/gogo/protobuf v1.1.1 // indirect
	github.com/golang/lint v0.0.0-20180702182130-06c8688daad7 // indirect
	github.com/golang/protobuf v1.2.0
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db // indirect
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/gxed/GoEndian v0.0.0-20160916112711-0f5c6873267e // indirect
	github.com/gxed/eventfd v0.0.0-20160916113412-80a92cca79a8 // indirect
	github.com/gxed/hashland v0.0.0-20180221191214-d9f6b97f8db2 // indirect
	github.com/hashicorp/golang-lru v0.5.0 // indirect
	github.com/ijc25/Gotty v0.0.0-20170406111628-a8b993ba6abd
	github.com/ipfs/go-cid v0.9.0
	github.com/ipfs/go-datastore v3.2.0+incompatible
	github.com/ipfs/go-ipfs-util v1.2.8 // indirect
	github.com/ipfs/go-log v1.5.7 // indirect
	github.com/ipfs/go-todocounter v1.0.1 // indirect
	github.com/jbenet/go-temp-err-catcher v0.0.0-20150120210811-aac704a3f4f2 // indirect
	github.com/jbenet/goprocess v0.0.0-20160826012719-b497e2f366b8 // indirect
	github.com/kardianos/osext v0.0.0-20170510131534-ae77be60afb1 // indirect
	github.com/kkdai/bstream v0.0.0-20181106074824-b3251f7901ec // indirect
	github.com/libgit2/git2go v0.0.0-20190215132637-bf1e8a433882
	github.com/libp2p/go-addr-util v2.0.7+incompatible // indirect
	github.com/libp2p/go-buffer-pool v0.1.1 // indirect
	github.com/libp2p/go-conn-security v0.1.15 // indirect
	github.com/libp2p/go-conn-security-multistream v0.1.15 // indirect
	github.com/libp2p/go-flow-metrics v0.2.0 // indirect
	github.com/libp2p/go-libp2p v6.0.23+incompatible
	github.com/libp2p/go-libp2p-circuit v2.3.2+incompatible // indirect
	github.com/libp2p/go-libp2p-crypto v2.0.1+incompatible
	github.com/libp2p/go-libp2p-host v3.0.15+incompatible
	github.com/libp2p/go-libp2p-interface-connmgr v0.0.21 // indirect
	github.com/libp2p/go-libp2p-interface-pnet v3.0.0+incompatible // indirect
	github.com/libp2p/go-libp2p-kad-dht v4.4.12+incompatible
	github.com/libp2p/go-libp2p-kbucket v2.2.12+incompatible
	github.com/libp2p/go-libp2p-loggables v1.1.24 // indirect
	github.com/libp2p/go-libp2p-metrics v2.1.7+incompatible
	github.com/libp2p/go-libp2p-nat v0.8.8 // indirect
	github.com/libp2p/go-libp2p-net v3.0.15+incompatible
	github.com/libp2p/go-libp2p-peer v2.4.0+incompatible
	github.com/libp2p/go-libp2p-peerstore v2.0.6+incompatible
	github.com/libp2p/go-libp2p-protocol v1.0.0
	github.com/libp2p/go-libp2p-record v4.1.7+incompatible // indirect
	github.com/libp2p/go-libp2p-routing v2.7.1+incompatible // indirect
	github.com/libp2p/go-libp2p-secio v2.0.17+incompatible // indirect
	github.com/libp2p/go-libp2p-swarm v3.0.22+incompatible // indirect
	github.com/libp2p/go-libp2p-transport v3.0.15+incompatible // indirect
	github.com/libp2p/go-libp2p-transport-upgrader v0.1.16 // indirect
	github.com/libp2p/go-maddr-filter v1.1.10 // indirect
	github.com/libp2p/go-mplex v0.2.30 // indirect
	github.com/libp2p/go-msgio v0.0.6 // indirect
	github.com/libp2p/go-reuseport v0.1.18 // indirect
	github.com/libp2p/go-reuseport-transport v0.1.11 // indirect
	github.com/libp2p/go-sockaddr v1.0.3 // indirect
	github.com/libp2p/go-stream-muxer v3.0.1+incompatible // indirect
	github.com/libp2p/go-tcp-transport v2.0.16+incompatible // indirect
	github.com/libp2p/go-ws-transport v2.0.15+incompatible // indirect
	github.com/lunixbochs/struc v0.0.0-20180408203800-02e4c2afbb2a
	github.com/mattn/go-colorable v0.0.9 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	github.com/minio/sha256-simd v0.0.0-20181005183134-51976451ce19 // indirect
	github.com/mitchellh/go-homedir v1.0.0
	github.com/mkevac/debugcharts v0.0.0-20180124214838-d3203a8fa926 // indirect
	github.com/mr-tron/base58 v1.1.0 // indirect
	github.com/multiformats/go-multiaddr v1.3.0
	github.com/multiformats/go-multiaddr-dns v0.2.5 // indirect
	github.com/multiformats/go-multiaddr-net v1.6.3 // indirect
	github.com/multiformats/go-multibase v0.3.0 // indirect
	github.com/multiformats/go-multihash v1.0.8
	github.com/multiformats/go-multistream v0.3.9 // indirect
	github.com/opentracing/opentracing-go v1.0.2 // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pkg/errors v0.8.0
	github.com/rjeczalik/notify v0.9.2 // indirect
	github.com/rs/cors v1.6.0 // indirect
	github.com/shirou/gopsutil v2.18.10+incompatible // indirect
	github.com/sirupsen/logrus v1.2.0
	github.com/spaolacci/murmur3 v0.0.0-20180118202830-f09979ecbc72 // indirect
	github.com/syndtr/goleveldb v0.0.0-20181105012736-f9080354173f // indirect
	github.com/tyler-smith/go-bip39 v1.0.0
	github.com/urfave/cli v1.20.0
	github.com/whyrusleeping/base32 v0.0.0-20170828182744-c30ac30633cc // indirect
	github.com/whyrusleeping/go-keyspace v0.0.0-20160322163242-5b898ac5add1 // indirect
	github.com/whyrusleeping/go-logging v0.0.0-20170515211332-0457bb6b88fc // indirect
	github.com/whyrusleeping/go-notifier v0.0.0-20170827234753-097c5d47330f // indirect
	github.com/whyrusleeping/go-smux-multiplex v3.0.16+incompatible // indirect
	github.com/whyrusleeping/go-smux-multistream v2.0.2+incompatible // indirect
	github.com/whyrusleeping/go-smux-yamux v2.0.8+incompatible // indirect
	github.com/whyrusleeping/mafmt v1.2.8 // indirect
	github.com/whyrusleeping/multiaddr-filter v0.0.0-20160516205228-e903e4adabd7 // indirect
	github.com/whyrusleeping/yamux v1.1.2 // indirect
	golang.org/x/crypto v0.0.0-20190219172222-a4c6cb3142f2 // indirect
	golang.org/x/net v0.0.0-20181207154023-610586996380
	google.golang.org/grpc v1.17.0
)
