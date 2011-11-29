#!/bin/sh

export BASE_DIR="`dirname $0`"
export TEST_DIR="$(readlink -f $(dirname $0))"
top_dir="$BASE_DIR/.."

if test -z "$NO_MAKE"; then
    make -C $top_dir > /dev/null || exit 1
fi

if test -z "$CUTTER"; then
    CUTTER="`make -s -C $BASE_DIR echo-cutter`"
fi

if ! test -z "$DEBUG"; then
    WRAPPER="gdb --args"
elif ! test -z "$MEMCHECK"; then
    VGLOG=`mktemp`
    WRAPPER="valgrind --leak-check=summary --track-origins=yes --log-file=${VGLOG}"
fi

# MOCK_SO=$(readlink -f $(find -name mock.so | head -n1))
# if ! test -z "$MOCK_SO"; then
#     export LD_PRELOAD="$MOCK_SO"
# fi

$WRAPPER $CUTTER --keep-opening-modules -s $BASE_DIR "$@" $BASE_DIR
RET=$?

# show log file of valgrind
if ! test -z "$MEMCHECK"; then
    if ! test -z "$PAGER"; then
	$PAGER $VGLOG
    else
	cat $VGLOG
    fi
fi

exit $RET