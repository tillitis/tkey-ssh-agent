@echo off

set "TKEY_SSH_AGENT_VERSION=%~1"

if "%TKEY_SSH_AGENT_VERSION%"=="" (
  echo Please pass a version number to build as, like: 0.0.6
  echo This is our typical tagged version. We will make the
  echo actual version 0.0.6.0 per windows convention.
  exit
)

set TKEY_SSH_AGENT_VERSION=%TKEY_SSH_AGENT_VERSION%.0

echo Going to build: %TKEY_SSH_AGENT_VERSION%

set WIXPATH="C:\Program Files (x86)\WiX Toolset v3.11\bin"

%WIXPATH%\candle.exe tkey-ssh-agent.wxs

%WIXPATH%\light.exe -ext WixUIExtension ^
  -o tkey-ssh-agent-%TKEY_SSH_AGENT_VERSION%.msi ^
  tkey-ssh-agent.wixobj
