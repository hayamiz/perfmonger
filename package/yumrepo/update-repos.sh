#!/bin/bash

this_dir=$(dirname $(readlink -f $0))

set -ev

mkdir -p $this_dir/centos/

rsync -av --delete $this_dir/../rpm/centos/ $this_dir/centos/

for rpmdir in $(find $this_dir/centos -type f -name '*.rpm' -exec dirname {} \; | sort | uniq); do
    createrepo $rpmdir
done

ssh iwana 'rm -rf /var/www-package/*.rpm'
rsync -av $this_dir/centos/ iwana:/var/www-package/centos/
scp $this_dir/hayamiz-repos-*.rpm iwana:/var/www-package/
