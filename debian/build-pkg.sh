#!/bin/bash
set -eu

# # TODO
#
# lintian ./tkey-ssh-agent_0.1-1_amd64.deb
# E: tkey-ssh-agent: no-changelog usr/share/doc/tkey-ssh-agent/changelog.Debian.gz (non-native package)

pkgname="tkey-ssh-agent"
upstream_version="0.1"
debian_revision="1"
pkgversion="$upstream_version-$debian_revision"
pkgmaintainer="Tillitis <hello@tillitis.se>"

if [[ "$(uname -m)" != "x86_64" ]]; then
  printf "expecting to build on x86_64, bailing out\n"
  exit 1
fi

cd "${0%/*}" || exit 1
destdir="$PWD/build"
rm -rf "$destdir"
mkdir "$destdir"

pushd ..
make tkey-ssh-agent
make DESTDIR="$destdir" \
     PREFIX=/usr \
     SYSTEMDDIR=/usr/lib/systemd \
     UDEVDIR=/usr/lib/udev \
     install
popd

install -Dm644 deb/copyright "$destdir"/usr/share/doc/tkey-ssh-agent/copyright
install -Dm644 deb/lintian--overrides "$destdir"/usr/share/lintian/overrides/tkey-ssh-agent
mkdir "$destdir/DEBIAN"
cp -af deb/postinst "$destdir/DEBIAN/"
sed -e "s/##VERSION##/$pkgversion/" \
    -e "s/##PACKAGE##/$pkgname/" \
    -e "s/##MAINTAINER##/$pkgmaintainer/" \
    deb/control.tmpl >"$destdir/DEBIAN/control"

dpkg-deb --root-owner-group -Zgzip --build "$destdir" .
