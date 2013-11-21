#!/bin/bash

set -ev

DIST_VERSION=`cat /tmp/dist-version`
DIST_ARCH=`cat /tmp/dist-arch`
BUILD_SCRIPT=/tmp/build-user.sh
BUILD_RUBY_SCRIPT=/tmp/build-user-ruby.sh

yum install -y tar make intltool gcc rpm-build \
               wget readline readline-devel ncurses ncurses-devel gdbm \
               gdbm-devel glibc-devel tcl-devel unzip openssl-devel db4-devel \
               byacc libyaml-devel

if ! id perfmonger-build; then
    useradd -m perfmonger-build
fi

cat <<EOF > $BUILD_RUBY_SCRIPT
#!/bin/bash

cat <<EOM > ~/.rpmmacros
%_topdir \$HOME/rpm
EOM

DIST_VERSION=`cat /tmp/dist-version`
DIST_ARCH=`cat /tmp/dist-arch`

rm -rf ~/rpm/
mkdir -p ~/rpm/SOURCES
mkdir -p ~/rpm/SPECS
mkdir -p ~/rpm/BUILD
mkdir -p ~/rpm/RPMS
mkdir -p ~/rpm/SRPMS

cp /tmp/ruby193.spec ~/rpm/SPECS/
wget http://cache.ruby-lang.org/pub/ruby/1.9/ruby-1.9.3-p448.tar.gz -O ~/rpm/SOURCES/ruby-1.9.3-p448.tar.gz
rpmbuild -ba ~/rpm/SPECS/ruby193.spec
EOF

cat <<EOF > $BUILD_SCRIPT
#!/bin/bash

cat <<EOM > ~/.rpmmacros
%_topdir \$HOME/rpm
EOM

DIST_VERSION=`cat /tmp/dist-version`
DIST_ARCH=`cat /tmp/dist-arch`

rm -rf ~/rpm/
mkdir -p ~/rpm/SOURCES
mkdir -p ~/rpm/SPECS
mkdir -p ~/rpm/BUILD
mkdir -p ~/rpm/RPMS
mkdir -p ~/rpm/SRPMS

cp /tmp/perfmonger.spec ~/rpm/SPECS/
cp /tmp/perfmonger-*.tar.gz ~/rpm/SOURCES/
rpmbuild -ba ~/rpm/SPECS/perfmonger.spec
EOF


chmod +x $BUILD_SCRIPT $BUILD_RUBY_SCRIPT

if [ $DIST_VERSION = 5 ]; then
    su perfmonger-build -c $BUILD_RUBY_SCRIPT
    rpm -Uvh ~perfmonger-build/rpm/RPMS/$DIST_ARCH/ruby-*.rpm
    cp ~perfmonger-build/rpm/RPMS/$DIST_ARCH/ruby-*.rpm /tmp/
else
    yum install -y ruby ruby-devel rubygems
fi

su perfmonger-build -c $BUILD_SCRIPT
rpm -Uvh ~perfmonger-build/rpm/RPMS/$DIST_ARCH/perfmonger-*.rpm
perfmonger --version
rpm --erase perfmonger

if [ $DIST_VERSION = 5 ]; then
    rpm --erase ruby
fi

cp ~perfmonger-build/rpm/RPMS/$DIST_ARCH/perfmonger-*.rpm /tmp/
cp ~perfmonger-build/rpm/SRPMS/perfmonger-*.src.rpm /tmp/

