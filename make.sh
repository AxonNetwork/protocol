#!/bin/bash

__dirname="$(CDPATH= cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

function read_cli_flags {
    if [ "$#" -eq 0 ]; then
        native=1
    fi

    while [[ "$#" > 0 ]]; do case $1 in
        # -m|--darwin) deploy="$2"; shift;;
        -g|--libgit) libgit=1;;
        -d|--docker) docker=1;;
        -n|--native) native=1;;
        -c|--copy) copy=1;;
        *) echo "Unknown parameter passed: $1"; exit 1;;
    esac; shift; done
}


function check_os {
    unameOut="$(uname -s)"
    case "${unameOut}" in
        Linux*)     os=linux;;
        Darwin*)    os=mac;;
        CYGWIN*)    os=windows;;
        MINGW*)     os=windows;;
        *)          os="UNKNOWN:${unameOut}"
    esac
}

function build_native {

    if [[ $os == "windows" ]]; then
        PWD=$(pwd)
        GIT2GO_PATH="${PWD}/vendor/github.com/libgit2/git2go"
        LIBGIT2_BUILD="${GIT2GO_PATH}/vendor/libgit2/build"
        #FLAGS="-lws2_32"
        FLAGS="-lwinhttp -lcrypt32 -lrpcrt4 -lole32 -lws2_32"
        export CGO_LDFLAGS="${LIBGIT2_BUILD}/libgit2.a -L${LIBGIT2_BUILD} ${FLAGS}"
    fi

    mkdir -p build/native
    cd cmd/axon-node
    GO111MODULE=on go build --tags "static" -ldflags "-s -w" -o main ./*.go
    mv main ../../build/native/axon-node
    cd -

    # mkdir -p build/native
    # cd remote-helper
    # GO111MODULE=on go build --tags "static" -ldflags "-s -w" -o main ./*.go
    # mv main ../build/native/git-remote-axon
    # cd -

    # mkdir -p build/native
    # cd filters/encode/cmd
    # GO111MODULE=on go build --tags "static" -ldflags "-s -w" -o main ./*.go
    # mv main ../../../build/native/axon_encode
    # cd -

    # mkdir -p build/native
    # cd filters/decode/cmd
    # GO111MODULE=on go build --tags "static" -ldflags "-s -w" -o main ./*.go
    # mv main ../../../build/native/axon_decode
    # cd -

    # mkdir -p build/native
    # cd filters/diff
    # GO111MODULE=on go build --tags "static" -ldflags "-s -w" -o main ./*.go
    # mv main ../../build/native/axon_diff
    # cd -

    # mkdir -p build/native
    # cd cmd
    # GO111MODULE=on go build --tags "static" -ldflags "-s -w" -o main ./*.go
    # mv main ../build/native/axon
    # cd -
}

function build_docker {
    rm -rf ./build/docker &&
    mkdir -p build/docker &&
    pushd $__dirname/cmd/axon-node &&
    GO111MODULE=on go build --tags "static" -o main ./*.go &&
    mv main ../../build/docker/axon-node &&
    popd
}

function checkout_libgit2 {
    [[ -d vendor/github.com/libgit2/git2go ]] ||
        (mkdir -p vendor/github.com/libgit2 &&
        pushd vendor/github.com/libgit2 &&
        git clone https://github.com/Conscience/git2go &&
        pushd git2go &&
        git checkout 27d682f350318f017dc4ae1f5a6f70808a88f595 &&
        # git remote add lhchavez https://github.com/lhchavez/git2go &&
        # git fetch --all &&
        # git cherry-pick 122ccfadea1e219c819adf1e62534f0b869d82a3 &&
        touch go.mod &&
        git submodule update --init &&
        popd && popd)
}

function build_libgit2 {
    checkout_libgit2

    if [[ $os == "windows" ]]; then
        makefiles="MinGW Makefiles"
    else
        makefiles="Unix Makefiles"
    fi

    pushd vendor/github.com/libgit2/git2go/vendor/libgit2 &&
    mkdir -p install/lib &&
    mkdir -p build &&
    pushd build &&

    cmake -DTHREADSAFE=ON \
      -DBUILD_CLAR=OFF \
      -DBUILD_SHARED_LIBS=OFF \
      -DCMAKE_C_FLAGS=-fPIC \
      -DUSE_SSH=OFF \
      -DCURL=OFF \
      -DUSE_HTTPS=OFF \
      -DUSE_BUNDLED_ZLIB=ON \
      -DUSE_EXT_HTTP_PARSER=OFF \
      -DCMAKE_BUILD_TYPE="RelWithDebInfo" \
      -DCMAKE_INSTALL_PREFIX=../install \
      -DWINHTTP=OFF \
      -G "$makefiles" \
      .. && \
    cmake --build . &&
    popd && popd &&
    cat <<EOF

libgit2: Build complete.

  ===================================================================================
  | If you've just rebuilt libgit2 and are expecting git2go to pick up your changes |
  | next time you compile, please note that you'll need to use go build's "-a" flag |
  | like so:                                                                        |
  |                                                                                 |
  |    go build -a --tags "static" .                                                |
  |                                                                                 |
  | Subsequent builds will not require this flag unless you modify libgit2 again.   |
  ===================================================================================

EOF
}

check_os
read_cli_flags $@

echo Running gofmt
gofmt -s -w .

echo Building:
[[ -n $libgit  ]] && echo   - libgit2
[[ -n $docker  ]] && echo   - docker
[[ -n $native  ]] && echo   - native \($os\)


[[ -n $libgit ]] && (build_libgit2 || exit 1;)
[[ -n $docker ]] && (build_docker  || exit 1;)
[[ -n $native ]] && (build_native  || exit 1;)

[[ -n $copy ]] && echo Copying binaries to $DESKTOP_APP_BINARY_ROOT
[[ -n $copy ]] && cp -R ./build/native/* $DESKTOP_APP_BINARY_ROOT/

echo Done.
