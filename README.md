
[![ci](https://github.com/tillitis/tillitis-key1-apps/actions/workflows/ci.yaml/badge.svg?branch=main&event=push)](https://github.com/tillitis/tillitis-key1-apps/actions/workflows/ci.yaml)

# Tillitis TKey Apps

This repository contains device applications to run on the TKey USB
security stick, as well as companion client apps (running on the host
computer). For testing and development purposes, the device apps can
also be run in QEMU; this is also explained in detail below.

Current list of device apps:

- The Ed25519 signer app. Used for root of trust and SSH authentication
- The RNG stream app. Providing arbitrarily high quality random numbers
- blink. A minimalistic example application
- nx. Test program for our execution monitor.

For more information about the apps, see the subsections below.

The documentation for the Go module and packages (along with this
README) can also be read at
https://pkg.go.dev/github.com/tillitis/tillitis-key1-apps

Note that development is ongoing. For example, changes might be made
to the signer app, causing the public/private keys it provides to
change. To avoid unexpected changes, please use a tagged release.

See [Release notes](docs/release_notes.md).

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
- as defined by the Linux Foundation's [SPDX project](https://spdx.org/) -
on its own source files, but not necessarily imported files. The line
in each individual source file identifies the license applicable to
that file.

The current set of valid, predefined SPDX identifiers can be found on
the SPDX License List at:

https://spdx.org/licenses/

All contributors must adhere to the [Developer Certificate of Origin](dco.md).

## Building device apps

You have two options, either our OCI image
`ghcr.io/tillitis/tkey-builder` for use with a rootless podman setup,
or native tools.

In either case you need the device libraries in a directory next to
this one. The device libraries are available in:

https://github.com/tillitis/tkey-libs

Clone them to the directory above this repo and build them first.

### Building with Podman

We provide an OCI image with all the tools needed to build the
tkey-libs and apps. If you have `make` and Podman installed, you
can us it for `tkey-libs` directory this directory as shown below:

```
make podman
```

and everything should be built. This assumes a working rootless
podman. On Ubuntu 22.10, running

```
apt install podman rootlesskit slirp4netns
```

should be enough to get you a working Podman setup.

### Building with host tools

To build with native tools, you need the `clang`, `llvm`, `lld`,
and `golang` packages installed. Version 15 or later of LLVM/Clang is
required (with riscv32 support and Zmmul extension). Ubuntu 22.10
(Kinetic) is known to have this and works. Please see
[toolchain_setup.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/toolchain_setup.md)
(in the tillitis-key1 repository) for detailed information on the
currently supported build and development environment.

Clone and build the device libraries first:

```
$ git clone https://github.com/tillitis/tkey-libs
$ cd tkey-libs
$ make
```

Then go back to this directory and build everything:

```
$ make
```

If you cloned `tkey-libs` to somewhere else then the default directory,
set `LIBDIR` to the path of that directory.

If the `objcopy` binary on your system is anything other than the default
`llvm-objcopy`, define `OBJCOPY` to whatever they're called on
your system.

The device apps can be run both on the hardware TKey, and on a QEMU
machine that emulates the platform. In both cases, the client apps
(the program that runs on your computer, for example `tkey-ssh-agent`)
will talk to the app over a serial port. There is a
separate section below which explains how to run it in QEMU.


## Running device apps

Plug the USB stick into your computer. If the LED in one of the outer
corners of the USB stick is a steady white, it has been
programmed with the standard FPGA bitstream (including the firmware).
If it is not, then please refer to
[quickstart.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/quickstart.md)
(in the tillitis-key1 repository) for instructions on how to
initialise the USB stick.

### Linux Users

Running `lsusb` should list the USB stick as `1207:8887 Tillitis
MTA1-USB-V1`. On Linux, the TKey's serial port device path is
typically `/dev/ttyACM0` (but it may end with another digit if you
have other devices plugged in). The client apps tries to auto-detect
serial ports of TKey USB sticks but if more than one is found, you'll
need to choose one explicitly using the `--port` flag.

You should make sure that you have the necessary privileges to read and
write to the serial port.

One way to accomplish this is by installing the provided
`system/60-tkey.rules` in `/etc/udev/rules.d/` and running `udevadm
control --reload`. Now when a TKey is plugged in, its device path
(like `/dev/ttyACM0`) should be read/writable by you who are logged in
locally (see `loginctl`).

Another way is to become a member of the group that owns the serial
port. On Ubuntu the group is `dialout` and you can do it like this:

```
$ id -un
exampleuser
$ ls -l /dev/ttyACM0
crw-rw---- 1 root dialout 166, 0 Sep 16 08:20 /dev/ttyACM0
$ sudo usermod -a -G dialout exampleuser
```

For the change to take effect everywhere, you need to logout from your
system and then log back in again. You can also run (in addition to the
example above) `newgrp dialout` in the terminal that you're working in.

Your TKey should now be running the firmware. Its LED should be a steady white,
indicating that it is ready to receive an app to run.

#### MacOS Users

The client apps tries to auto-detect serial ports of TKey USB sticks
but if more than one is found, you'll need to choose one explicitly using the
`--port` flag.

To find the serial ports device path manually, you can do `ls -l
/dev/cu.*`. There should be a device named like `/dev/cu.usbmodemN`
(where N is a number, for example 101). This is the device path that
might need to be passed as `--port` when running the client app.

You can verify that the OS has found and enumerated the USB stick by
running:

```
ioreg -p IOUSB -w0 -l
```

There should be an entry with `"USB Vendor Name" = "Tillitis"`.

### Running device apps in QEMU

For making development easier, we provide a container image and
support script for running a TKey/QEMU machine with the latest
firmware. The script assumes a working rootless Podman setup (and
socat). It currently only works on a Linux system (specifically, it
does not work when containers are run in Podman's virtual machine,
which is required on MacOS and Windows). On Ubuntu 22.10, running `apt
install podman rootlesskit slirp4netns socat` should be enough. Then
you can just run the script like:

    ./contrib/run-tkey-qemu

Among other advice, the script outputs the path which you need to pass
to the client app so that it can communicate with the virtualised TKey
inside the container. An example:

    ./tkey-runapp --port ./tkey-qemu-pty apps/signer/app.bin

#### Building and running QEMU manually

Build our [qemu](https://github.com/tillitis/qemu) fork. Use the `tk1`
branch. Please follow the
[toolchain_setup.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/toolchain_setup.md)
and install the packages listed there first.


```
$ git clone -b tk1 https://github.com/tillitis/qemu
$ mkdir qemu/build
$ cd qemu/build
$ ../configure --target-list=riscv32-softmmu --disable-werror
$ make -j $(nproc)
```

(Built with warnings-as-errors disabled, see [this
issue](https://github.com/tillitis/qemu/issues/3).)

You also need to build the firmware:

```
$ git clone https://github.com/tillitis/tillitis-key1
$ cd tillitis-key1/hw/application_fpga
$ make firmware.elf
```

Please refer to the mentioned
[toolchain_setup.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/toolchain_setup.md)
if you have any issues building it.

Then run the emulator, passing the firmware built to "-bios":

```
$ /path/to/qemu/build/qemu-system-riscv32 -nographic -M tk1,fifo=chrid -bios firmware.elf \
  -chardev pty,id=chrid
```

It tells you what serial port it is using, for instance `/dev/pts/1`.
This is what you need to pass to `--port` when running the client apps.

The TK1 machine running on QEMU (which in turn runs the firmware, and
then the device app) can output some memory access (and other)
logging. You can add `-d guest_errors` to the QEMU commandline to make
it send these to stderr.


## The ed25519 signer app

This is a message signer for root of trust and SSH authentication
using ed25519. There are two client apps which can communicate with
the app. `tkey-sign` just performs a complete test signing.
`tkey-ssh-agent` is an SSH agent that allow using the signer for SSH
remote access.

### Using runsign.sh, tkey-runapp, and tkey-sign

If you're running on hardware, the LED on the USB stick is expected to
be a steady white, indicating that firmware is ready to receive a
device app to run.

There's a script called `runsign.sh`, which runs `tkey-runapp` to load
the signer app onto TKey and start it. The script then runs
`tkey-sign`, which communicates with the signer app to make it sign a
message and then verifies the signature. You can use it like this:

```
./runsign.sh file-with-message
```

The signer app can sign messages of up to 4096 bytes. If the `--port`
flags needs to be used, you can pass it after the message argument.

The client app `tkey-runapp` only loads and starts a device app.
You'll then have to switch to a different client app that speaks your
app's specific protocol. For instance, the `tkey-sign` program provided
here.

To run `tkey-runapp`, you need to pass it the raw app binary that
should be run (and possibly `--port`, if the auto-detection is not
sufficient).

```
$ ./tkey-runapp apps/signer/app.bin
```
While the app is being loaded, the LED on the USB stick (in one of the
outer corners) will be turned off. `tkey-runapp` also supports sending
a User Supplied Secret (USS) to the firmware when loading the app. By
adding the`--uss` flag, you will be asked to type a phrase which will
be hashed to become the USS digest (the final newline is removed from
the phrase before hashing).

Alternatively, you may use `--uss-file=filename` to make it read the
contents of a file, which is then hashed as the USS. The filename can
be set to `-` to read from the stdin instead. Note that all data from
the file/stdin is read and hashed without any modification.

The firmware uses the USS digest, together with a hash digest of the
raw device app binary and the Unique Device Secret (UDS, unique per
physical device) to derive secrets for use by the application.

The practical result for users of the signer app is that the ed25519
public/private keys will change along with the USS. So if you enter a
different phrase (or pass a file with different contents), the derived
USS will change, and so will your identity. To learn more, read the
[system_description.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/system_description/system_description.md)
(in the tillitis-key1 repository).

`tkey-sign` assumes that `tkey-runapp` has been used to load the
signer app and can be used like this (again, `--port` is optional):

```
./tkey-sign file-with-message
```

If you're using real hardware, the LED on the USB stick becomes a steady
green while the app is receiving data to sign. The LED then flashes
green, indicating that you're required to physically touch the USB
stick for the signing to complete. The touch sensor is located next to
the flashing LED -- touch it and release. If running on QEMU, the
virtual device is always touched automatically.

The program should eventually output a signature and say that it was
verified.

When all is done, the LED on the hardware USB stick will become a steady
blue, indicating that it is ready to make (another) signature.

Note that, to load a new device app, the USB stick needs to be
unplugged and plugged in again. Similarly, QEMU would need to be
restarted (`Ctrl-a x` to quit). If you're using a setup with the USB
stick sitting in a programming jig and at the same time plugged into
the computer, then you need to unplug both the USB stick and the
programmer. Or alternatively run the `reset-tk1` script (in the
tillitis-key1 repo).

That was fun, now let's try the SSH agent!

### Using tkey-ssh-agent

This client app is a complete, alternative SSH agent with practical
use. The required signer device app binary gets built into the
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
binary, the USS, or the UDS in the physical USB stick change, the
private key will also change -- and thus the derived public key: your
public identity in the world of SSH.

If you copy-paste the public key into your `~/.ssh/authorized_keys`,
you can try to log onto your local computer (if sshd is running
there). The socket path set/output above is also needed by SSH in
`SSH_AUTH_SOCK`:

```
$ SSH_AUTH_SOCK=/path/to/agent.sock ssh -F /dev/null localhost
```

`-F /dev/null` is used to ignore your `~/.ssh/config`, which could
interfere with this test.

The tkey-ssh-agent also supports the `--uss` and `--uss-file` flags,
as described for `tkey-runapp` above.

You can use `--show-pubkey` (short flag: `-p`) to only output the
pubkey. The pubkey is printed to stdout for easy redirection, but some
messages are still present on stderr.

#### Installing tkey-ssh-agent

The [`Makefile`](Makefile) has an `install` target that installs
tkey-ssh-agent and the above mentioned `60-tkey.rules`. Run `make`,
`sudo make install`, and then `sudo make reload-rules` to apply the
rules to the running system. This also installs a man page which
contains some useful information; try `man ./system/tkey-ssh-agent.1`
to read it before installing.

There is also a Work In Progress Debian/Ubuntu package which can be
build using the script `debian/build-pkg.sh`.

#### Windows support

tkey-ssh-agent can be built for and run on Windows. The Makefile has a
`windows` target that produces `tkey-ssh-agent.exe` and
`tkey-ssh-agent-tray.exe`. The former is a regular command-line
program that can be used, for example, in PowerShell. The latter is a
small program (built for the `windowsgui` subsystem; no console) that
sets up a tray icon and launches `tkey-ssh-agent.exe` (which it
expects to find in the same directory as itself) with the arguments
provided. For automatically starting the SSH agent when logging
onto the computer, a shortcut to `tkey-ssh-agent-tray.exe`, with the
required arguments, can be added in your user's `Startup` folder.

When using the `--uss` option (as described for `tkey-runapp` above),
the Windows build defaults to the pinentry program from Gpg4win
for requesting the User-Supplied Secret. This package can be installed
using: `winget install GnuPG.Gpg4win`.

The SSH Agent supports being used by the native OpenSSH client
`ssh.exe` (part of Windows Optional Features and installable using
`winget`). The `SSH_AUTH_SOCK` environment variable should be set to
the complete path of the Named Pipe that tkey-ssh-agent listens on.

For example, if it is started using `./tkey-ssh-agent.exe -a
tkey-ssh-agent`, the environment variable could be set for the current
PowerShell like this:

```powershell
$env:SSH_AUTH_SOCK = '\\.\pipe\tkey-ssh-agent'
```

The System Control Panel can be used to set the environment variable
persistently for future PowerShell terminals, Visual Studio Code, and
other programs. Or using PowerShell:

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

The signer app normally requires the USB stick to be physically
touched for signing to complete. For special purposes it can be
compiled with this requirement removed by setting the environment
variable `TKEY_SIGNER_APP_NO_TOUCH` when building.
Example: `make TKEY_SIGNER_APP_NO_TOUCH=yesplease`.

The client apps will also stop displaying this requirement. Of course,
this changes the signer app binary and as a consequence, the derived
private key and identity will change.

## The RNG stream app

This app generates a continuous stream of high quality random numbers
that can be read from the TKey's serial port device. An example
use-case in Linux: `dd bs=1 count=1024 if=/dev/ttyACM0 of=rngdata` (or just
a plain `cat`).

The app can be loaded and started using the `tkey-runapp` as described
above.

The RNG is a Hash_DRBG implementation using the BLAKE2s hash function
as a primitive. The generator will extract at most 128 bits from each
hash operation, using 128 bits as an exclusive, evolving state. The RNG
will be reseeded after 1000 hash operations. Reseeding is done by
extracting 256 entropy bits from the TK1 TRNG core. Note that the
reseed rate can be changed during compile time by adjusting the
`RESEED_TIME` defined in main.c.


## Example blink app in assembler

In `blink/` there is also a very, very simple app written in
assembly, `blink.bin` (blink.S) that blinks the LED.


## Example Touch app

In `touch/` resides an example app of how to use the built in touch
feature of the TKey. The application simply waits for a touch from
the user while flashing the LED. A touch is confirmed by switching
the flashing color while starting to wait for a new touch - shifting
colors between green, red and blue.

Run the app by invoking

```
$ ./tkey-runapp apps/touch/app.bin
```


## Developing apps

Device apps and libraries are kept under the `apps` directory. A C
runtime is provided as `libcrt0.a` in the
[tkey-libs](https://github.com/tillitis/tkey-libs) which you can link
your C apps with.

### Memory

RAM starts at 0x4000\_0000 and ends at 0x4002\_0000 (128 KB). The app
will be loaded by firmware at the top of RAM. The stack for the app is
setup to start at the bottom of RAM (see
[apps/libcrt0/crt0.S](apps/libcrt0/crt0.S)). A larger app comes at a
compromise of it having a smaller stack.

There are no heap allocation functions, no `malloc()` and friends.

Special memory areas for memory mapped hardware functions are
available at base 0xc000\_0000 and an offset. See
[software.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/system_description/software.md)
(in the tillitis-key1 repository), and the include file
[tk1_mem.h](https://github.com/tillitis/tkey-libs/blob/main/include/tk1_mem.h)
(in the tkey-libs repository).

### Debugging

If you're running the device app on our qemu emulator, we have added a
debug port on 0xfe00\_1000 (`TK1_MMIO_QEMU_DEBUG`). Anything written
there will be printed as a character by qemu on the console.

See documentation in
[tkey-libs](https://github.com/tillitis/tkey-libs) for how to use the
convienence functions for interacting with this debug port.
