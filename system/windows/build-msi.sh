#!/bin/bash
# SPDX-FileCopyrightText: 2023 Tillitis AB <tillitis.se>
# SPDX-License-Identifier: BSD-2-Clause

set -e

SEMVER="${1:-}"
if [[ ! "$SEMVER" =~ ^(([1-9][0-9]*|0)\.){2}([1-9][0-9]*|0)$ ]]; then
  printf "Expected a semver in 1st arg, like: 0.0.6\n"
  exit 1
fi
shift

wxsf="${1:-}"
if [[ ! -e "$wxsf" ]] || [[ ! "$wxsf" =~ \.wxs ]]; then
  printf "Expected a .wxs file in 2nd arg\n"
  exit 1
fi
shift

export SEMVER="$SEMVER.0"
base="${wxsf%.wxs}"

printf "Going to build: %s\n" "$SEMVER"

wine /usr/local/wix/candle.exe "$wxsf"

wine /usr/local/wix/light.exe -sval -ext WixUIExtension \
  -o "$base-$SEMVER.msi" "$base.wixobj"
