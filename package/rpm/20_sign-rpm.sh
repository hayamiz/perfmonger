#!/bin/bash

set -ev

this_dir=$(dirname $(readlink -f $0))

for rpmfile in $(find $this_dir -name '*.rpm' -type f); do
    rpm -D "_gpg_name 981A94C0" --resign $rpmfile
done
