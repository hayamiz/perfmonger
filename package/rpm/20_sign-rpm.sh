#!/bin/bash

this_dir=$(dirname $(readlink -f $0))

set -ev

for rpmfile in $(find $this_dir/centos -name '*.rpm' -type f); do
    rpm -D "_gpg_name 981A94C0" \
            --resign $rpmfile
done
