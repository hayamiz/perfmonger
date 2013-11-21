#!/bin/bash

this_dir=$(dirname $(readlink -f $0))

set -ev

function build_in_chroot() {
    if [ $# -lt 2 ]; then
        echo "Usage: $0 CENTOS_VERSION ARCH"
        return 1
    fi

    [ $1 = 5 ] || [ $1 = 6 ]
    [ $2 = i386 ] || [ $2 = amd64 ]

    set -ev

    local version=$1
    local arch=$2
    local chroot_base=/var/lib/chroot/centos-$version-$arch

    ##
    ## Setup base system by rinse
    ##

    sudo mkdir -p $chroot_base/dev
    sudo mkdir -p $chroot_base/proc
    sudo mkdir -p $chroot_base/sys

    sudo rinse --arch $arch --distribution centos-$version --directory $chroot_base

    if ! grep ${chroot_base}/dev /etc/mtab; then
        sudo mount -o bind /dev ${chroot_base}/dev
    fi
    if ! grep ${chroot_base}/proc /etc/mtab; then
        sudo mount -t proc none ${chroot_base}/proc
    fi
    if ! grep ${chroot_base}/sys /etc/mtab; then
        sudo mount -o bind /sys ${chroot_base}/sys
    fi

    arch=${arch/amd64/x86_64}

    ##
    ## Put files for building package in chroot environment
    ##

    cp $this_dir/build-rpm.sh $chroot_base/tmp/
    cp $this_dir/perfmonger.spec $chroot_base/tmp/
    cp $this_dir/vendor/ruby193.spec $chroot_base/tmp/
    cp $this_dir/../../perfmonger-*.tar.gz $chroot_base/tmp/
    echo $version > $chroot_base/tmp/dist-version
    echo $arch > $chroot_base/tmp/dist-arch

    ##
    ## Run build script with chroot
    ##

    sudo chroot $chroot_base /tmp/build-rpm.sh


    ##
    ## Extract RPM files
    ##

    mkdir -p $this_dir/centos/$version/$arch
    mkdir -p $this_dir/centos/$version/SRPMS

    cp $chroot_base/tmp/*.$arch.rpm $this_dir/centos/$version/$arch/
    cp $chroot_base/tmp/*.src.rpm   $this_dir/centos/$version/SRPMS/
}


##
## Build tarball
##
rm -f $this_dir/../../perfmonger-*.tar.gz
make -C $this_dir/../.. > /dev/null
make -C $this_dir/../.. dist > /dev/null

##
## Build for each version/archtecture
##

build_in_chroot 5 amd64
build_in_chroot 6 amd64

##
## Finished!
##
