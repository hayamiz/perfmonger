#!/bin/bash

this_dir=$(dirname $(readlink -f $0))

set -ev

mkdir -p $this_dir/centos/6/SRPMS/
mkdir -p $this_dir/centos/6/x86_64/

for rpmfile in $(find $this_dir/../rpm -type f -name "*.rpm"); do
    case $rpmfile in
        *.src.rpm)
            cp $rpmfile $this_dir/centos/6/SRPMS/
            ;;
        *.x86_64.rpm)
            cp $rpmfile $this_dir/centos/6/x86_64/
            ;;
    esac
done

for rpmdir in $(find $this_dir/centos -type f -name '*.rpm' -exec dirname {} \; | sort | uniq); do
    createrepo $rpmdir
done

rsync -av $this_dir/centos/ iwana:/var/www-package/centos/
