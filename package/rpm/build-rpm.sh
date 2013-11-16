#!/bin/bash

set -ev

BUILD_USER_SCRIPT=/tmp/build-user.sh

yum install -y tar make intltool gcc ruby ruby-devel rubygems rpm-build

if ! id perfmonger-build; then
    useradd -m perfmonger-build
fi

cat <<EOF > $BUILD_USER_SCRIPT
#!/bin/bash

cat <<EOM > ~/.rpmmacros
%_topdir \$HOME/rpm
EOM

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

chmod +x $BUILD_USER_SCRIPT
su perfmonger-build -c $BUILD_USER_SCRIPT
rpm -Uvh ~perfmonger-build/rpm/RPMS/*/perfmonger-*.rpm
perfmonger --version
rpm --erase perfmonger

cp ~perfmonger-build/rpm/RPMS/*/perfmonger-*.rpm /tmp/
cp ~perfmonger-build/rpm/SRPMS/perfmonger-*.src.rpm /tmp/

