#!/bin/bash
set -eu

# This script uses tkey-runapp to load a signer app that has been patched to
# disable the touch requirement. Then it runs tkey-sign forever, signing 128
# bytes of new random data on every iteration.
#
# User is expected to first run this script once with the argument "patch",
# which will patch the sources to disable the touch requirement, and compile
# the binaries.
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

is_commented() {
  file="$1"
  line="$2"
  if grep -q "^$line$" "$file"; then
    return 1
  fi
  if ! grep -q "^//$line$" "$file"; then
    # It doesn't have either $line or //$line
    printf "%s is neither patched nor patchable, maybe we became outdated?\n" "$file"
    exit 1
  fi
  return 0
}

commentout() {
  file="$1"
  line="$2"
  if is_commented "$file" "$line"; then
    return
  fi
  tmpf=$(mktemp)
  cp -af "$file" "$tmpf"
  sed -i "s,^$line$,//&," "$tmpf"
  mv -f "$tmpf" "$file"
}

file1=cmd/tkey-sign/main.go
line1="[[:space:]]*le.Print.*will.flash.*touch.*required.*"
file2=apps/signer/main.c
line2="[[:space:]]*wait_touch_.*"

if [[ "${1:-}" = "patch" ]]; then
  commentout "$file1" "$line1"
  commentout "$file2" "$line2"
  make -C apps
  make tkey-runapp tkey-sign
  if ! is_commented "$file1" "$line1" \
      || ! is_commented "$file2" "$line2"; then
    printf "Something went wrong when patching.\n"
    exit 1
  fi
  exit 0
fi

if ! is_commented "$file1" "$line1" \
    || ! is_commented "$file2" "$line2"; then
  printf "The touch requirement is still present, not patched.\n"
  printf "Please run this once first: %s patch\n" "$0"
  exit 1
fi


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
  # 128 bytes becomes 1 msg with 127 bytes and 1 msg with 1 byte
  dd 2>/dev/null if=/dev/urandom of="$msgf" bs=128 count=1
  if ! ./tkey-sign "$@" "$msgf"; then
    exit 1
  fi
  c=$(( c+1 ))
  now=$(date +%s)
  printf "loop count: %d, seconds passed: %d\n" "$c" "$((now - start))"
done
