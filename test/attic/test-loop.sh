#!/bin/bash
# Copyright (C) 2022 - Tillitis AB
# SPDX-License-Identifier: BSD-2-Clause
#
set -eu

# This script uses tkey-runapp to load a signer app which has been built with
# the touch requirement removed. Then it runs tkey-sign forever, signing 512
# bytes of new random data on every iteration.
#
# The script expects that the TKey is in firmware mode, so it can load the
# correct signer app.
#
# Arguments to this script will be passed to tkey-runapp and tkey-sign, so
# --port and --speed can be used.
#
# If the environment variable USB_DEVICE is set, --port $USB_DEVICE is passed
# to these programs.

cd "${0%/*}/.."

if [[ -e tkey-sign ]]; then
  if ! go version -m tkey-sign | grep -q main.signerAppNoTouch=indeed; then
    printf "We need to build with the touch requirement removed.\n"
    printf "Please first do: make -C ../ clean\n"
    exit 1
  fi
fi
make TKEY_SIGNER_APP_NO_TOUCH=indeed -C apps
make TKEY_SIGNER_APP_NO_TOUCH=indeed tkey-runapp tkey-sign

if [[ -n "${USB_DEVICE:-}" ]]; then
  # Passing this last to make it override
  set -- "$@" --port "$USB_DEVICE"
fi

# We expect to load the app ourselves, exiting if we couldn't
if ! ./tkey-runapp "$@" apps/signer/app.bin; then
  exit 1
fi

msgf=$(mktemp)
cleanup() {
  rm -f "$msgf"
}
trap cleanup EXIT

c=0
start=$(date +%s)
while true; do
  # 512 bytes becomes 1 msg with 511 bytes and 1 msg with 1 byte
  dd 2>/dev/null if=/dev/urandom of="$msgf" bs=512 count=1
  if ! ./tkey-sign "$@" "$msgf"; then
    exit 1
  fi
  c=$(( c+1 ))
  now=$(date +%s)
  printf "loop count: %d, seconds passed: %d\n" "$c" "$((now - start))"
done
