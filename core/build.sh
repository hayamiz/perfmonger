#!/bin/bash

READLINK=$(type -p greadlink readlink | head -1)
cd $(dirname $($READLINK -f $0))

if [[ $1 = "-" ]]; then
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
    TARGET=("linux amd64"  "darwin amd64")
fi

set -e

GO_DEPS=$(ls subsystem/*.go; ls utils.go)

makefile=`mktemp`

cat <<EOF > $makefile
# DO NOT EDIT MANUALLY
# generated by build.sh

GO_DEPS := $(echo ${GO_DEPS})
GO_SRC := utils.go

.PHONY: all build clean

all: build

EOF

TARGETS=()

for idx in $(seq 0 $((${#TARGET[@]}-1))); do
    set -- ${TARGET[$idx]}
    export var_GOOS=$1
    export var_GOARCH=$2

    for subcmd in recorder player viewer summarizer plot-formatter; do
        TARGETS+=(../lib/exec/perfmonger-${subcmd}_${var_GOOS}_${var_GOARCH})

        cat <<EOF >> $makefile

../lib/exec/perfmonger-${subcmd}_${var_GOOS}_${var_GOARCH}: perfmonger-${subcmd}.go \$(GO_DEPS)
	go build -o \$@ perfmonger-$subcmd.go \$(GO_SRC)

EOF
    done

    # go build -o ../lib/exec/perfmonger-recorder_${var_GOOS}_${var_GOARCH} \
    #     perfmonger-recorder.go &
    # go build -o ../lib/exec/perfmonger-player_${var_GOOS}_${var_GOARCH} \
    #     perfmonger-player.go &
    # go build -o ../lib/exec/perfmonger-summarizer_${var_GOOS}_${var_GOARCH} \
    #    perfmonger-summarizer.go &

done

cat <<EOF >> $makefile

build: ${TARGETS[*]}

clean:
	rm -f ${TARGETS[*]}

EOF

mv $makefile ./Makefile

make
