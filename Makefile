all: build

deps:
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

build: deps build/conscience-node build/git-remote-conscience build/conscience_encode build/conscience_decode build/conscience_diff build/conscience-init

build/conscience-node: swarm/**/*.go
	mkdir -p build
	cd swarm/cmd; \
	go build -o main *.go; \
	mv main ../../build/conscience-node

build/git-remote-conscience: remote-helper/*.go
	mkdir -p build
	cd remote-helper; \
	go build -o main *.go; \
	mv main ../build/git-remote-conscience

build/conscience_encode: filters/encode/*.go
	mkdir -p build
	cd filters/encode; \
	go build -o main *.go; \
	mv main ../../build/conscience_encode

build/conscience_decode: filters/decode/*.go
	mkdir -p build
	cd filters/decode; \
	go build -o main *.go; \
	mv main ../../build/conscience_decode

build/conscience_diff: filters/diff/*.go
	mkdir -p build
	cd filters/diff; \
	go build -o main *.go; \
	mv main ../../build/conscience_diff

build/conscience-init: cmd/init/*.go
	mkdir -p build
	cd cmd/init; \
	go build -o main *.go; \
	mv main ../../build/conscience-init


install: build
	cp build/* /usr/local/bin/