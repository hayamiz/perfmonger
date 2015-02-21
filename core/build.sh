#!/bin/bash

READLINK=$(type -p greadlink readlink | head -1)
cd $(dirname $($READLINK -f $0))

TARGET=("linux 386" "linux amd64")

set -xe

for idx in $(seq 0 $((${#TARGET[@]}-1))); do
    set -- ${TARGET[$idx]}
    export GOOS=$1
    export GOARCH=$2

    go build -o ../lib/exec/perfmonger-recorder_${GOOS}_${GOARCH} \
        perfmonger-recorder.go
    go build -o ../lib/exec/perfmonger-player_${GOOS}_${GOARCH} \
        perfmonger-player.go
done

