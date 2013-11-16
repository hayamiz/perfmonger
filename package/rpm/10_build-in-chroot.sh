#!/bin/bash

this_dir=$(dirname $(readlink -f $0))

set -ev

CHROOT_BASE=/var/lib/chroot/centos6-amd64

##
## Build tarball
##

rm -f $this_dir/../../perfmonger-*.tar.gz
make -C $this_dir/../.. dist > /dev/null


##
## Setup base system by rinse
##

sudo mkdir -p $CHROOT_BASE/dev
sudo mkdir -p $CHROOT_BASE/proc
sudo mkdir -p $CHROOT_BASE/sys

sudo rinse --arch amd64 --distribution centos-6 --directory $CHROOT_BASE

if ! grep ${CHROOT_BASE}/dev /etc/mtab; then
    sudo mount -o bind /dev ${CHROOT_BASE}/dev
fi
if ! grep ${CHROOT_BASE}/proc /etc/mtab; then
    sudo mount -t proc none ${CHROOT_BASE}/proc
fi
if ! grep ${CHROOT_BASE}/sys /etc/mtab; then
    sudo mount -o bind /sys ${CHROOT_BASE}/sys
fi


##
## Copy files for building package in chroot environment
##

cp $this_dir/build-rpm.sh $CHROOT_BASE/tmp/
cp $this_dir/perfmonger.spec $CHROOT_BASE/tmp/
cp $this_dir/../../perfmonger-*.tar.gz $CHROOT_BASE/tmp/


##
## Run build script with chroot
##

sudo chroot $CHROOT_BASE /tmp/build-rpm.sh


##
## Extract RPM files
##

cp $CHROOT_BASE/tmp/*.rpm $this_dir/


##
## Finished!
##
