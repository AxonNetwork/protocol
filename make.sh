#!/bin/bash

if [ "$#" -eq 0 ]; then
    darwin=1
    windows=1
    linux=1
fi

while [[ "$#" > 0 ]]; do case $1 in
  # -m|--darwin) deploy="$2"; shift;;
  -g|--libgit) libgit=1;;
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
[[ -n $libgit  ]] && echo   - libgit2
[[ -n $darwin  ]] && echo   - darwin
[[ -n $windows ]] && echo   - windows
[[ -n $linux   ]] && echo   - linux


function build_native {
    mkdir -p build/native
    cd swarm/cmd
    GO111MODULE=on go build --tags "static" -ldflags "-s -w" -o main ./*.go
    mv main ../../build/native/conscience-node
    cd -

    mkdir -p build/native
    cd remote-helper
    GO111MODULE=on go build --tags "static" -ldflags "-s -w" -o main ./*.go
    mv main ../build/native/git-remote-conscience
    cd -

    # mkdir -p build/native
    # cd filters/encode
    # GO111MODULE=on go build --tags "static" -ldflags "-s -w" -o main ./*.go
    # mv main ../../build/native/conscience_encode
    # cd -

    # mkdir -p build/native
    # cd filters/decode
    # GO111MODULE=on go build --tags "static" -ldflags "-s -w" -o main ./*.go
    # mv main ../../build/native/conscience_decode
    # cd -

    # mkdir -p build/native
    # cd filters/diff
    # GO111MODULE=on go build --tags "static" -ldflags "-s -w" -o main ./*.go
    # mv main ../../build/native/conscience_diff
    # cd -

    # mkdir -p build/native
    # cd cmd
    # GO111MODULE=on go build --tags "static" -ldflags "-s -w" -o main ./*.go
    # mv main ../build/native/conscience
    # cd -
}

function checkout_libgit2 {
    local GIT2GO_PATH="vendor/github.com/libgit2/git2go"

    [[ -d $(GIT2GO_PATH) ]] ||
        mkdir -p vendor/github.com/libgit2 &&
        pushd vendor/github.com/libgit2 &&
        git clone https://github.com/Conscience/git2go &&
        pushd git2go &&
        git checkout 81a759a2593aeb28b7bb07439da9796489bfe3bb &&
        # git remote add lhchavez https://github.com/lhchavez/git2go &&
        # git fetch --all &&
        # git cherry-pick 122ccfadea1e219c819adf1e62534f0b869d82a3 &&
        touch go.mod &&
        git submodule update --init &&
        popd && popd
}

function build_libgit2 {
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
          -DCMAKE_BUILD_TYPE="RelWithDebInfo" \
          -DCMAKE_INSTALL_PREFIX=../install \
          .. && \
        cmake --build . &&
        popd && popd
}

[[ -n $libgit ]] && build_libgit2
[[ -n $darwin ]] && build_darwin
[[ -n $linux ]] && build_linux
[[ -n $windows ]] && build_windows
[[ -n $native ]] && build_native

[[ -n $copy ]] && cp -R ./build/* $DESKTOP_APP_BINARY_ROOT/

echo Build complete.
