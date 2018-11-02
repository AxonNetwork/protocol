#!/bin/sh

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
  *) echo "Unknown parameter passed: $1"; exit 1;;
esac; shift; done

echo Building:
[[ -n $darwin ]]  && echo   - darwin
[[ -n $windows ]] && echo   - windows
[[ -n $linux ]]   && echo   - linux




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

    mkdir -p build/windows
    cd filters/encode
    xgo --targets=windows/amd64 -out main .
    mv main-windows-4.0-amd64.exe ../../build/windows/conscience_encode.exe
    cd -

    mkdir -p build/windows
    cd filters/decode
    xgo --targets=windows/amd64 -out main .
    mv main-windows-4.0-amd64.exe ../../build/windows/conscience_decode.exe
    cd -

    mkdir -p build/windows
    cd filters/diff
    xgo --targets=windows/amd64 -out main .
    mv main-windows-4.0-amd64.exe ../../build/windows/conscience_diff.exe
    cd -

    mkdir -p build/windows
    cd cmd
    xgo --targets=windows/amd64 -out main .
    mv main-windows-4.0-amd64.exe ../build/windows/conscience.exe
    cd -
}


[[ -n $darwin ]] && build_darwin
[[ -n $linux ]] && build_linux
[[ -n $windows ]] && build_windows

cp -R ./build/* $DESKTOP_APP_BINARY_ROOT/