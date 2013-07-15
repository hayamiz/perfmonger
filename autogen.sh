#!/bin/sh

run()
{
    $@
    if test $? -ne 0; then
        echo "Failed $@"
        exit 1
    fi
}

run aclocal ${ACLOCAL_ARGS}

case $(uname) in
    Darwin*)
        run glibtoolize --copy --force
        ;;
    *)
        run libtoolize --copy --force
        ;;
esac

run autoheader
run automake --add-missing --foreign --copy
run autoconf
