#! /bin/sh

LIBDIR=../tkey-libs

git clone https://github.com/tillitis/tkey-libs.git ../tkey-libs
git clone https://github.com/tillitis/tkey-device-signer.git ../tkey-device-signer

make -j -C ../tkey-libs
make -j -C ../tkey-device-signer

cp ../tkey-device-signer/signer/app.bin cmd/tkey-ssh-agent/app.bin

make -j
