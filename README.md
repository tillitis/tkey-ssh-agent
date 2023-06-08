
[![ci](https://github.com/tillitis/tillitis-key1-apps/actions/workflows/ci.yaml/badge.svg?branch=main&event=push)](https://github.com/tillitis/tillitis-key1-apps/actions/workflows/ci.yaml)

# Tillitis TKey Apps

This repository contains some applications for the
[Tillitis](https://tillitis.se/) TKey USB security stick.

Client apps:

- `tkey-runapp`: A simple development tool to load and start any TKey
  device app, like those below.
- `tkey-ssh-agent`: An OpenSSH compatible agent.
- `runtimer`: Control the `timer` device app.

Device apps:

- `rng_stream`: Outputs high quality random numbers directly. You can
  `cat` directly from the TKey device but see also
  [tkey-random-generator](https://github.com/tillitis/tkey-random-generator)
  for something more polished.
- `blink`: A minimalistic example in assembly.
- `nx`: Test program for the execution monitor.
- `timer`: Example/test app on how to use the hardware timer.
- `touch`: Example/test app for the touch sensor. Cycles between
  colours when touching.

See the [TKey Developer Handbook](https://dev.tillitis.se/) for how to
develop your own apps, how to run and debug them in the emulator or on
real hardware.

[Current list of known projects](https://dev.tillitis.se/projects/).

Go packages used with the client apps reside in their own
repositories:

- https://github.com/tillitis/tkeyclient [Go doc](https://pkg.go.dev/github.com/tillitis/tkeyclient)
- https://github.com/tillitis/tkeysign [Go doc](https://pkg.go.dev/github.com/tillitis/tkeysign)

Note that development is ongoing. For example, changes might be made
to [the signer device
app](https://github.com/tillitis/tkey-device-signer), causing the
public/private key it provides to change. To avoid unexpected changes
please use a tagged release.


See [Release notes](docs/release_notes.md).

## Building

You have two options, either our OCI image
`ghcr.io/tillitis/tkey-builder` for use with a rootless podman setup,
or native tools. See [the Devoloper
Handbook](https://dev.tillitis.se/) for setup.

With native tools you should be able to use our build script:

```
$ ./build.sh
```

which also clones and builds the [TKey device
libraries](https://github.com/tillitis/tkey-libs) and the [signer
device app](https://github.com/tillitis/tkey-device-signer) first.

If you want to do it manually, clone and build tkey-libs and
tkey-device-signer manually like this:

```
$ git clone -b v0.0.1 https://github.com/tillitis/tkey-libs
$ cd tkey-libs
$ make
$ cd ..
$ git clone -b v0.0.7 https://github.com/tillitis/tkey-device-signer
$ cd tkey-device-signer
$ make
$ cd ..
$ cp ../tkey-device-signer/signer/app.bin cmd/tkey-ssh-agent/app.bin
```

Then go back to this directory and build everything:

```
$ make
```

If you cloned `tkey-libs` to somewhere else then the default set
`LIBDIR` to the path of the directory.

If your available `objcopy` is anything other than the default
`llvm-objcopy`, then define `OBJCOPY` to whatever they're called on
your system.

If you want to use podman and you have `make` you can run:

```
$ podman pull ghcr.io/tillitis/tkey-builder:2
$ make podman
```

or run podman directly with

```
$ podman run --rm --mount type=bind,source=$(CURDIR),target=/src --mount type=bind,source=$(CURDIR)/../tkey-libs,target=/tkey-libs -w /src -it ghcr.io/tillitis/tkey-builder:2 make -j
```

To help prevent unpleasant surprises we keep a hash of the `signer` in
`cmd/tkey-ssh-agent/app.bin.sha512`. The compilation will fail if this
is not the expected binary.

### Using tkey-runapp

The client app `tkey-runapp` only loads and starts a device app. It's
mostly a development tool. You'll then have to switch to a different
client app that speaks your app's specific protocol. Run with `-h` to
get help.

### Using tkey-ssh-agent

This client app is a complete, alternative SSH agent with practical
use. The needed signer device app binary gets built into the
tkey-ssh-agent, which will load it onto USB stick when started. Like
the other client apps, tkey-ssh-agent tries to auto-detect serial
ports of TKey USB sticks. If more than one is found, or if you're
running on QEMU, then you'll need to use the `--port` flag. An example
of that:

```
$ ./tkey-ssh-agent -a ./agent.sock --port /dev/pts/1
```

This will start the SSH agent and tell it to listen on the specified
socket `./agent.sock`.

It will also output the SSH ed25519 public key for this instance of
the app on this specific TKey USB stick. So again; if the signer app
binary, the USS, or the UDS in the physical USB stick change, then the
private key will also change -- and thus the derived public key, your
public identity in the world of SSH.

If you copy-paste the public key into your `~/.ssh/authorized_keys`
you can try to log onto your local computer (if sshd is running
there). The socket path set/output above is also needed by SSH in
`SSH_AUTH_SOCK`:

```
$ SSH_AUTH_SOCK=/path/to/agent.sock ssh -F /dev/null localhost
```

`-F /dev/null` is used to ignore your ~/.ssh/config which could
interfere with this test.

The tkey-ssh-agent also supports the `--uss` and `--uss-file` flags,
as described for `tkey-runapp` above.

You can use `--show-pubkey` (short flag: `-p`) to only output the
pubkey. The pubkey is printed to stdout for easy redirection, but some
messages are still present on stderr.

#### Installing tkey-ssh-agent

The [`Makefile`](Makefile) has an `install` target that installs
tkey-ssh-agent and the above mentioned `60-tkey.rules`. First `make`
then `sudo make install`, then `sudo make reload-rules` to apply the
rules to the running system. This also installs a man page which
contains some useful information, try `man ./system/tkey-ssh-agent.1`
to read it before installing.

There is also a Work In Progress Debian/Ubuntu package which can be
build using the script `debian/build-pkg.sh`.

#### Windows support

tkey-ssh-agent can be built for and run on Windows. The Makefile has a
`windows` target that produces `tkey-ssh-agent.exe` and
`tkey-ssh-agent-tray.exe`. The former is a regular command-line
program that can be used for example in PowerShell. The latter is a
small program (built for the `windowsgui` subsystem; no console) that
sets up a tray icon and launches `tkey-ssh-agent.exe` (which it
expects to find next to itself) with the same arguments that it was
itself passed. For automatically starting the SSH agent when logging
onto the computer, a shortcut to `tkey-ssh-agent-tray.exe`, with the
required arguments, can be added in your user's `Startup` folder.

When using the `--uss` option (as described for `tkey-runapp` above),
the Windows build by default uses the pinentry program from Gpg4win
for requesting the User-Supplied Secret. This package can be installed
using: `winget install GnuPG.Gpg4win`.

The SSH Agent supports being used by the native OpenSSH client
`ssh.exe` (part of Windows Optional Features and installable using
`winget`). The environment variable `SSH_AUTH_SOCK` should be set to
the complete path of the Named Pipe that tkey-ssh-agent listens on.

For example, if it is started using `./tkey-ssh-agent.exe -a
tkey-ssh-agent` the environment variable could be set for the current
PowerShell like this:

```powershell
$env:SSH_AUTH_SOCK = '\\.\pipe\tkey-ssh-agent'
```

Setting this environment variable persistently, for future PowerShell
terminals, Visual Studio Code, and other programs can be done through
the System Control Panel. Or using PowerShell:

```powershell
[Environment]::SetEnvironmentVariable('SSH_AUTH_SOCK', '\\.\pipe\tkey-ssh-agent', 'User')
```

You can learn more about environment variables on Windows in [this
article](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_environment_variables).

The SSH Agent can also be used with the Git-for-Windows client
(`winget install Git.Git`). By default, it uses its own bundled
ssh-client. Run the following PowerShell commands to make `git.exe`
use the system's native ssh.exe:

```
$sshpath = (get-command ssh.exe).path -replace '\\','/'
git config --global core.sshCommand $sshpath
git config --global --get core.sshCommand
```

The last command should output something like
`C:/Windows/System32/OpenSSH/ssh.exe`.

For details on how we package and build an MSI installer, see
[system/windows/README.md](system/windows/README.md).

### Disabling touch requirement

The [signer device app](https://github.com/tkey-device-signer)
normally requires the USB stick to be physically touched for signing
to complete. For special purposes it can be compiled with this
requirement removed, by setting the environment variable
`TKEY_SIGNER_APP_NO_TOUCH` to some value when building. Example: `make
TKEY_SIGNER_APP_NO_TOUCH=yesplease`. 

*Note well*: You have to do this when building both the signer and the
client apps. The client apps will also stop displaying notifications
about touch if the variable is set.

Of course this changes the signer app binary and as a consequence the
derived private key and identity will change.

## Licenses and SPDX tags

Unless otherwise noted, the project sources are licensed under the
terms and conditions of the "GNU General Public License v2.0 only":

> Copyright Tillitis AB.
>
> These programs are free software: you can redistribute it and/or
> modify it under the terms of the GNU General Public License as
> published by the Free Software Foundation, version 2 only.
>
> These programs are distributed in the hope that it will be useful,
> but WITHOUT ANY WARRANTY; without even the implied warranty of
> MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
> General Public License for more details.

> You should have received a copy of the GNU General Public License
> along with this program. If not, see:
>
> https://www.gnu.org/licenses

See [LICENSE](LICENSE) for the full GPLv2-only license text.

External source code we have imported are isolated in their own
directories. They may be released under other licenses. This is noted
with a similar `LICENSE` file in every directory containing imported
sources.

The project uses single-line references to Unique License Identifiers
as defined by the Linux Foundation's [SPDX project](https://spdx.org/)
on its own source files, but not necessarily imported files. The line
in each individual source file identifies the license applicable to
that file.

The current set of valid, predefined SPDX identifiers can be found on
the SPDX License List at:

https://spdx.org/licenses/

All contributors must adhere to the [Developer Certificate of Origin](dco.md).
