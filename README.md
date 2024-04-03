[![ci](https://github.com/tillitis/tillitis-key1-apps/actions/workflows/ci.yaml/badge.svg?branch=main&event=push)](https://github.com/tillitis/tillitis-key1-apps/actions/workflows/ci.yaml)

# TKey SSH Agent

`tkey-ssh-agent` is an OpenSSH-compatible agent for use with the
[Tillitis](https://tillitis.se/) TKey USB security token.

**Warning**: Please use tagged releases for any real use. Development
on main might mean we change which version of the [the signer device
app](https://github.com/tillitis/tkey-device-signer) we use which
would cause the SSH key pair to change!

See [Release notes](docs/release_notes.md).

## Installing

tkey-ssh-agent might be available in your operating system's package
system.

If not, see [Tillitis' application page for the
agent as well as instructions](https://tillitis.se/app/tkey-ssh-agent/).

If there's no official package for your system the easiest way to
install is probably to:

```
$ go install github.com/tillitis/tkey-ssh-agent/cmd/tkey-ssh-agent@latest
```

After this the `tkey-ssh-agent` command should be available in your
`$GOBIN` directory.

Note that installing with `go install` doesn't set the version like
building with other methods does. See building the agent below.

You will also have to install these manually if you use go install:

- Manual page `system/tkey-ssh-agent.1`.
- udev rules, see `system/60-tkey.rules` (Linux).
- systemd service unit, see `system/tkey-ssh-agent.service.tmpl` and
  change `##BINDIR##` to where you installed `tkey-ssh-agent` (some
  Linux dists).

If you're building from source (see below) there is a `make install`
target that installs the agent and the udev rules. Please see
`Makefile` to see that everything ends up where you want to. Typically
you will have to do:

```
$ sudo make install
$ sudo make reload-rules
```

## Using tkey-ssh-agent

`tkey-ssh-agent` tries to auto-detect the TKey. If more than one is
found, or if you're running on QEMU, then you'll need to use the
`--port` flag:

```
$ ./tkey-ssh-agent -a ./agent.sock --port /dev/pts/1
```

This will start the SSH agent and tell it to listen on the specified
socket `./agent.sock`.

**Nota bene**: If the signer app binary, the USS, or the UDS in the
physical USB stick change your key pair will change.

If you copy-paste the public key into your `~/.ssh/authorized_keys`
you can try to log onto your local computer (if sshd is running
there). The socket path set/output above is also needed by SSH in
`SSH_AUTH_SOCK`:

```
$ SSH_AUTH_SOCK=/path/to/agent.sock ssh -F /dev/null localhost
```

`-F /dev/null` is used to ignore your ~/.ssh/config which could
interfere with this test.

The tkey-ssh-agent also supports the `--uss` and `--uss-file` flags to
enter a User Supplied Secret.

You can use `--show-pubkey` (short flag: `-p`) to only output the
pubkey. The pubkey is printed to stdout for easy redirection, but some
messages are still present on stderr.

## Building the agent

If you have Go and make installed, a simple:

```
$ make
```

or, for a Windows executable,

```
$ make tkey-ssh-agent.exe
```

should build the agent. A pre-compiled signer device app binary is
included in the repo and will be automatically embedded.

Cross compiling the usual Go way with `GOOS` and `GOARCH` environment
variables works for most targets but currently doesn't work for
`GOOS=darwin` since the `go.bug.st/serial` package relies on macOS
shared libraries for port enumeration.

### Building agent with tkey-builder

If you want to use our tkey-builder image and you have `make` you can
run:

```
$ podman pull ghcr.io/tillitis/tkey-builder:4
$ make podman
```

or run it directly with Podman:

```
$ podman run --rm --mount type=bind,source=$(CURDIR),target=/src --mount type=bind,source=$(CURDIR)/../tkey-libs,target=/tkey-libs -w /src -it ghcr.io/tillitis/tkey-builder:4 make -j
```

Note that building with Podman like this by default creates a Linux
binary. Set `GOOS` and `GOARCH` with `-e` in the call to `podman run`
to desired target. Again, this won't work with a macOS target.

### Building with another signer

For convenience, and to be able to support `go install`, a precompiled
[signer device app](https://github.com/tkey-device-signer) binary is
included under `cmd/tkey-ssh-agent`.

If you want to replace the signer used by the agent you have to:

1. Compile your own signer and place it in the `cmd/tkey-ssh-agent`
   directory.
2. Change the path to the embedded signer in
   `cmd/tkey-ssh-agent/signer.go`. Look for `go:embed...`.
3. Change the `appName` directly under the `go:embed` to whatever your
   signer is called so the agent reports this correctly with
   `--version`.
4. Compute a new SHA-512 hash digest for your binary, typically by
   something like `sha512sum cmd/tkey-ssh-agent/signer.bin-v0.0.7` and
   put the resulting output in the file `signer.bin.sha512` at the top
   level.
5. `make` in the top level.

### Disabling touch requirement

The [signer device app](https://github.com/tkey-device-signer)
normally requires the TKey to be physically touched to make a
signature. For special purposes it can be compiled without this
requirement by setting the environment variable
`TKEY_SIGNER_APP_NO_TOUCH` to some value when building. Example: `make
TKEY_SIGNER_APP_NO_TOUCH=yesplease`.

*Note well*: You have to do this when building both the signer and the
client apps. `tkey-ssh-agent` will also stop displaying notifications
about touch if the variable is set.

**Warning**: Of course changing the code also changes the signer
binary and as a consequence the SSH key pair will also change.

## Building the signer

1. See [the Devoloper Handbook](https://dev.tillitis.se/) for setup of
   development tools. We recommend you use tkey-builder.
2. See the instructions in the [tkey-device-signer
   repo](https://github.com/tillitis/tkey-device-signer).
3. Copy its `signer/app.bin` to
   `cmd/tkey-sign/signer.bin-${signer_version}` and run `make`.

To help prevent unpleasant surprises we keep a digest of the signer in
`cmd/tkey-ssh-agent/signer.bin.sha512`. The compilation will fail if
this is not the expected binary. If you really intended to build with
another signer, see [Building with another
signer](#building-with-another-signer) above.

## Windows support

`tkey-ssh-agent` can be built for Windows. The Makefile has a
`windows` target that produces `tkey-ssh-agent.exe` and
`tkey-ssh-agent-tray.exe`. The former is a regular command-line
application, suitable for use in environments like PowerShell. The
latter is a small application built for the `windowsgui`
subsystem, meaning it operates without a console. Its primary function
is to create a tray icon and initiate `tkey-ssh-agent.exe` with the
identical arguments it received. They are assumed to be located in the
same directory. For automatically starting the SSH agent when logging
onto the computer, a shortcut to `tkey-ssh-agent-tray.exe`, with the
required arguments, can be added in your user's `Startup` folder.

When using the `--uss` option the Windows build by default uses the
pinentry program from Gpg4win for requesting the User-Supplied Secret.
This package can be installed using: `winget install GnuPG.Gpg4win`.

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

You can learn more about environment variables on Windows in [Microsoft's
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

