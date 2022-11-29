#!/bin/bash
set -eu

# # TODO
#
# lintian ./tk-ssh-agent_0.1-1_amd64.deb
# E: tk-ssh-agent: no-changelog usr/share/doc/tk-ssh-agent/changelog.Debian.gz (non-native package)

pkgname="tk-ssh-agent"
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
make tk-ssh-agent
make DESTDIR="$destdir" \
     PREFIX=/usr \
     SYSTEMDDIR=/usr/lib/systemd \
     UDEVDIR=/usr/lib/udev \
     install
popd

install -Dm644 deb/copyright "$destdir"/usr/share/doc/tk-ssh-agent/copyright
install -Dm644 deb/lintian--overrides "$destdir"/usr/share/lintian/overrides/tk-ssh-agent
mkdir "$destdir/DEBIAN"
cp -af deb/postinst "$destdir/DEBIAN/"
sed -e "s/##VERSION##/$pkgversion/" \
    -e "s/##PACKAGE##/$pkgname/" \
    -e "s/##MAINTAINER##/$pkgmaintainer/" \
    deb/control.tmpl >"$destdir/DEBIAN/control"

dpkg-deb --root-owner-group --build "$destdir" .
