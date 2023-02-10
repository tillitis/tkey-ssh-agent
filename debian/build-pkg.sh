#!/bin/bash
set -eu

# # TODO
#
# - We currently dig out the version from a git tag, so we can't build from
#   tarball. Not great.
#
# - lintian ./tkey-ssh-agent_0.1-1_amd64.deb
#   E: tkey-ssh-agent: no-changelog usr/share/doc/tkey-ssh-agent/changelog.Debian.gz (non-native package)

pkgname="tkey-ssh-agent"
debian_revision="1"
pkgmaintainer="Tillitis <hello@tillitis.se>"

if [[ "$(uname -m)" != "x86_64" ]]; then
  printf "expecting to build on x86_64, bailing out\n"
  exit 1
fi

cd "${0%/*}" || exit 1
destdir="$PWD/build"
rm -rf "$destdir"
mkdir "$destdir"

pushd >/dev/null ..

# upstream_version is the version of the program we're packaging
upstream_version="$(git describe --dirty --always | sed -n "s/^v\(.*\)/\1/p")"
if [[ -z "$upstream_version" ]]; then
  printf "found no tag (with v-prefix) to use for upstream_version\n"
  exit 1
fi
if [[ ! "$upstream_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  printf "%s: repo has commit after last tag, or git tree is dirty\n" "$upstream_version"
  exit 1
fi
pkgversion="$upstream_version-$debian_revision"

make clean
make -j TKEY_SSH_AGENT_VERSION="$upstream_version" tkey-ssh-agent
make -C apps check-signer-hash
make DESTDIR="$destdir" \
     PREFIX=/usr \
     SYSTEMDDIR=/usr/lib/systemd \
     UDEVDIR=/usr/lib/udev \
     install

popd >/dev/null

install -Dm644 deb/copyright "$destdir"/usr/share/doc/tkey-ssh-agent/copyright
install -Dm644 deb/lintian--overrides "$destdir"/usr/share/lintian/overrides/tkey-ssh-agent
mkdir "$destdir/DEBIAN"
cp -af deb/postinst "$destdir/DEBIAN/"
sed -e "s/##VERSION##/$pkgversion/" \
    -e "s/##PACKAGE##/$pkgname/" \
    -e "s/##MAINTAINER##/$pkgmaintainer/" \
    deb/control.tmpl >"$destdir/DEBIAN/control"

dpkg-deb --root-owner-group -Zgzip --build "$destdir" .

for f in *.deb; do
  sha512sum "$f" >"$f".sha512
done
