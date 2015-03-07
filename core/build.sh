#!/bin/bash

READLINK=$(type -p greadlink readlink | head -1)
cd $(dirname $($READLINK -f $0))

if [ $1 = "-" ]; then
    # do self build
    case `uname -s` in
        (Linux)
            os="linux"
            ;;
        (Darwin)
            os="darwin"
            ;;
        (*)
            os=""
            ;;
    esac
    case `uname -m` in
        (x86_64|amd64)
            arch="amd64"
            ;;
        (*)
            arch=""
            ;;
    esac

    TARGET=("${os} ${arch}")
else
    # cross build
    TARGET=("linux 386" "linux amd64"  "darwin amd64")
fi

set -xe

for idx in $(seq 0 $((${#TARGET[@]}-1))); do
    set -- ${TARGET[$idx]}
    export GOOS=$1
    export GOARCH=$2

    go build -o ../lib/exec/perfmonger-recorder_${GOOS}_${GOARCH} \
        perfmonger-recorder.go
    go build -o ../lib/exec/perfmonger-player_${GOOS}_${GOARCH} \
        perfmonger-player.go
    go build -o ../lib/exec/perfmonger-summarizer_${GOOS}_${GOARCH} \
        perfmonger-summarizer.go
done

