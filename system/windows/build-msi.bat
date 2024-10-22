@echo off
:: Copyright (C) 2023 - Tillitis AB
:: SPDX-License-Identifier: BSD-2-Clause

set "SEMVER=%~1"

if "%SEMVER%"=="" (
  echo Please pass a version number to build as, like: 0.0.6
  echo This is our typical tagged version. We will make the
  echo actual version 0.0.6.0 per windows convention.
  exit
)

set SEMVER=%SEMVER%.0

echo Going to build: %SEMVER%

set WIXPATH="C:\Program Files (x86)\WiX Toolset v3.11\bin"

%WIXPATH%\candle.exe tkey-ssh-agent.wxs

%WIXPATH%\light.exe -ext WixUIExtension ^
  -o tkey-ssh-agent-%SEMVER%.msi ^
  tkey-ssh-agent.wixobj
