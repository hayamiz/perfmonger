#!/bin/bash

this_dir=$(dirname $(readlink -f $0))

set -ev

##
## Create source for rpm packaging
##

tar czvf hayamiz-repos.tar.gz RPM-GPG-KEY-hayamiz hayamiz.repo


##
## Setup rpm build env
##

echo "%_topdir $HOME/rpm" > ~/.rpmmacros
mkdir -p ~/rpm/{SOURCES,SPECS,BUILD,RPMS,SRPMS}
cp hayamiz-repos.tar.gz ~/rpm/SOURCES/
cp hayamiz-repos.spec ~/rpm/SPECS/


##
## Build rpm
##

rpmbuild -ba ~/rpm/SPECS/hayamiz-repos.spec
cp ~/rpm/RPMS/noarch/hayamiz-repos-*.rpm $this_dir/
