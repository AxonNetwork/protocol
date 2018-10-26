all: build-dev

deps:
	go get github.com/whyrusleeping/gx
	go get github.com/whyrusleeping/gx-go
	gx install
	go get github.com/sirupsen/logrus
	go get github.com/BurntSushi/toml
	go get github.com/mitchellh/go-homedir
	go get github.com/pkg/errors
	go get github.com/ethereum/go-ethereum
	go get github.com/tyler-smith/go-bip39
	go get github.com/lunixbochs/struc
	go get github.com/btcsuite/btcd
	go get github.com/btcsuite/btcutil
	go get gopkg.in/src-d/go-git.v4
	go get github.com/urfave/cli
	go get github.com/aclements/go-rabin/rabin
	go get github.com/dustin/go-humanize
	go get github.com/brynbellomy/debugcharts
	go get github.com/golang/protobuf/proto
	go get golang.org/x/net/context
	go get google.golang.org/grpc
	go get github.com/brynbellomy/debugcharts
	go get github.com/Shopify/logrus-bugsnag
	go get github.com/bugsnag/bugsnag-go

gofmt:
	gofmt -s -w .

generate:
	go generate ./...

build-dev: BUILD_TAGS = -tags debug
build-dev: build
build-prod: build
build: gofmt deps build/conscience-node build/git-remote-conscience build/conscience_encode build/conscience_decode build/conscience_diff build/conscience

build/conscience-node: swarm/**/*.go
	mkdir -p build
	cd swarm/cmd; \
	go build $(BUILD_TAGS) -o main *.go; \
	mv main ../../build/conscience-node

build/git-remote-conscience: remote-helper/*.go
	mkdir -p build
	cd remote-helper; \
	go build $(BUILD_TAGS) -o main *.go; \
	mv main ../build/git-remote-conscience

build/conscience_encode: filters/encode/*.go
	mkdir -p build
	cd filters/encode; \
	go build $(BUILD_TAGS) -o main *.go; \
	mv main ../../build/conscience_encode

build/conscience_decode: filters/decode/*.go
	mkdir -p build
	cd filters/decode; \
	go build $(BUILD_TAGS) -o main *.go; \
	mv main ../../build/conscience_decode

build/conscience_diff: filters/diff/*.go
	mkdir -p build
	cd filters/diff; \
	go build $(BUILD_TAGS) -o main *.go; \
	mv main ../../build/conscience_diff

build/conscience: cmd/*.go
	mkdir -p build
	cd cmd; \
	go build $(BUILD_TAGS) -o main *.go; \
	mv main ../build/conscience


install:
	cp build/* /usr/local/bin/

