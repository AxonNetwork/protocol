#!/bin/bash

if [ "$#" -eq 0 ]; then
    darwin=1
    windows=1
    linux=1
fi

while [[ "$#" > 0 ]]; do case $1 in
  # -m|--darwin) deploy="$2"; shift;;
  -m|--darwin) darwin=1;;
  -w|--windows) windows=1;;
  -l|--linux) linux=1;;
  -n|--native) native=1;;
  -c|--copy) copy=1;;
  *) echo "Unknown parameter passed: $1"; exit 1;;
esac; shift; done

echo Running gofmt
gofmt -s -w .

echo Building:
[[ -n $darwin  ]] && echo   - darwin
[[ -n $windows ]] && echo   - windows
[[ -n $linux   ]] && echo   - linux


function get_deps {
    set -x
    # GO111MODULE=on go get github.com/whyrusleeping/gx
    # GO111MODULE=on go get github.com/whyrusleeping/gx-go
    # gx install

    # Install regular packages
    GO111MODULE=on go get github.com/sirupsen/logrus
    GO111MODULE=on go get github.com/BurntSushi/toml
    GO111MODULE=on go get github.com/mitchellh/go-homedir
    GO111MODULE=on go get github.com/pkg/errors
    GO111MODULE=on go get github.com/ethereum/go-ethereum
    GO111MODULE=on go get github.com/tyler-smith/go-bip39
    GO111MODULE=on go get github.com/lunixbochs/struc
    GO111MODULE=on go get github.com/btcsuite/btcd
    GO111MODULE=on go get github.com/btcsuite/btcutil
    GO111MODULE=on go get gopkg.in/src-d/go-git.v4
    GO111MODULE=on go get github.com/urfave/cli
    GO111MODULE=on go get github.com/aclements/go-rabin/rabin
    GO111MODULE=on go get github.com/dustin/go-humanize
    GO111MODULE=on go get github.com/golang/protobuf/proto
    GO111MODULE=on go get golang.org/x/net/context
    GO111MODULE=on go get google.golang.org/grpc
    GO111MODULE=on go get github.com/brynbellomy/debugcharts
    GO111MODULE=on go get github.com/Shopify/logrus-bugsnag
    GO111MODULE=on go get github.com/bugsnag/bugsnag-go
    set +x
}


function build_native {
    mkdir -p build/native
    cd swarm/cmd
    GO111MODULE=on go build -o main ./*.go
    mv main ../../build/native/conscience-node
    cd -

    mkdir -p build/native
    cd remote-helper
    GO111MODULE=on go build -o main ./*.go
    mv main ../build/native/git-remote-conscience
    cd -

    mkdir -p build/native
    cd filters/encode
    GO111MODULE=on go build -o main ./*.go
    mv main ../../build/native/conscience_encode
    cd -

    mkdir -p build/native
    cd filters/decode
    GO111MODULE=on go build -o main ./*.go
    mv main ../../build/native/conscience_decode
    cd -

    mkdir -p build/native
    cd filters/diff
    GO111MODULE=on go build -o main ./*.go
    mv main ../../build/native/conscience_diff
    cd -

    mkdir -p build/native
    cd cmd
    GO111MODULE=on go build -o main ./*.go
    mv main ../build/native/conscience
    cd -
}

function build_darwin {
    mkdir -p build/darwin
    cd swarm/cmd
    xgo --targets=darwin/amd64 -out main .
    mv main-darwin-10.6-amd64 ../../build/darwin/conscience-node
    cd -

    mkdir -p build/darwin
    cd remote-helper
    xgo --targets=darwin/amd64 -out main .
    mv main-darwin-10.6-amd64 ../build/darwin/git-remote-conscience
    cd -

    mkdir -p build/darwin
    cd filters/encode
    xgo --targets=darwin/amd64 -out main .
    mv main-darwin-10.6-amd64 ../../build/darwin/conscience_encode
    cd -

    mkdir -p build/darwin
    cd filters/decode
    xgo --targets=darwin/amd64 -out main .
    mv main-darwin-10.6-amd64 ../../build/darwin/conscience_decode
    cd -

    mkdir -p build/darwin
    cd filters/diff
    xgo --targets=darwin/amd64 -out main .
    mv main-darwin-10.6-amd64 ../../build/darwin/conscience_diff
    cd -

    mkdir -p build/darwin
    cd cmd
    xgo --targets=darwin/amd64 -out main .
    mv main-darwin-10.6-amd64 ../build/darwin/conscience
    cd -
}

function build_linux {
    mkdir -p build/linux
    cd swarm/cmd
    xgo --targets=linux/amd64 -out main .
    mv main-linux-amd64 ../../build/linux/conscience-node
    cd -

    mkdir -p build/linux
    cd remote-helper
    xgo --targets=linux/amd64 -out main .
    mv main-linux-amd64 ../build/linux/git-remote-conscience
    cd -

    mkdir -p build/linux
    cd filters/encode
    xgo --targets=linux/amd64 -out main .
    mv main-linux-amd64 ../../build/linux/conscience_encode
    cd -

    mkdir -p build/linux
    cd filters/decode
    xgo --targets=linux/amd64 -out main .
    mv main-linux-amd64 ../../build/linux/conscience_decode
    cd -

    mkdir -p build/linux
    cd filters/diff
    xgo --targets=linux/amd64 -out main .
    mv main-linux-amd64 ../../build/linux/conscience_diff
    cd -

    mkdir -p build/linux
    cd cmd
    xgo --targets=linux/amd64 -out main .
    mv main-linux-amd64 ../build/linux/conscience
    cd -
}

function build_windows {
    mkdir -p build/windows
    cd swarm/cmd
    xgo --targets=windows/amd64 -out main .
    mv main-windows-4.0-amd64.exe ../../build/windows/conscience-node.exe
    cd -

    mkdir -p build/windows
    cd remote-helper
    xgo --targets=windows/amd64 -out main .
    mv main-windows-4.0-amd64.exe ../build/windows/git-remote-conscience.exe
    cd -

    # mkdir -p build/windows
    # cd filters/encode
    # xgo --targets=windows/amd64 -out main .
    # mv main-windows-4.0-amd64.exe ../../build/windows/conscience_encode.exe
    # cd -

    # mkdir -p build/windows
    # cd filters/decode
    # xgo --targets=windows/amd64 -out main .
    # mv main-windows-4.0-amd64.exe ../../build/windows/conscience_decode.exe
    # cd -

    # mkdir -p build/windows
    # cd filters/diff
    # xgo --targets=windows/amd64 -out main .
    # mv main-windows-4.0-amd64.exe ../../build/windows/conscience_diff.exe
    # cd -

    # mkdir -p build/windows
    # cd cmd
    # xgo --targets=windows/amd64 -out main .
    # mv main-windows-4.0-amd64.exe ../build/windows/conscience.exe
    # cd -
}

get_deps

[[ -n $darwin ]] && build_darwin
[[ -n $linux ]] && build_linux
[[ -n $windows ]] && build_windows
[[ -n $native ]] && build_native

[[ -n $copy ]] && cp -R ./build/* $DESKTOP_APP_BINARY_ROOT/

echo Build complete.
