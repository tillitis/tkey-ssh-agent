
[![ci](https://github.com/tillitis/tillitis-key1-apps/actions/workflows/ci.yaml/badge.svg?branch=main&event=push)](https://github.com/tillitis/tillitis-key1-apps/actions/workflows/ci.yaml)

# Tillitis TKey Apps

This repository contains applications to run on the TKey USB security
stick. For testing and development purposes the apps can also be run
in QEMU, this is also explained in detail below.

Current list of apps:

- The Ed25519 signer app. Used as root of trust and SSH authentication
- The RNG stream app. Providing arbitrary high quality random numbers
- blink. A minimalistic example application
- nx. Test program for our execution monitor.

For more information about the apps, see subsections below.

The documentation for the Go module and packages (along with this
README) can also be read over at
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
as defined by the Linux Foundation's [SPDX project](https://spdx.org/)
on its own source files, but not necessarily imported files. The line
in each individual source file identifies the license applicable to
that file.

The current set of valid, predefined SPDX identifiers can be found on
the SPDX License List at:

https://spdx.org/licenses/

All contributors must adhere to the [Developer Certificate of Origin](dco.md).

## Building apps

To build you need the `clang`, `llvm`, `lld`, `golang` packages
installed. Version 15 or later of LLVM/Clang is required (with riscv32
support and Zmmul extension). Ubuntu 22.10 (Kinetic) is known to have
this and work. Please see
[toolchain_setup.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/toolchain_setup.md)
(in the tillitis-key1 repository) for detailed information on the
currently supported build and development environment.

Build everything:

```
$ make
```

If your available `objcopy` is anything other than the default
`llvm-objcopy`, then define `OBJCOPY` to whatever they're called on
your system.

The apps can be run both on the hardware TKey, and on a QEMU machine
that emulates the platform. In both cases, the host program (the
program that runs on your computer, for example `tkey-ssh-agent`) will
talk to the app over a serial port, virtual or real. There is a
separate section below which explains running in QEMU.


## Running apps

Plug the USB stick into your computer. If the LED in one of the outer
corners of the USB stick is a steady white, then it has been
programmed with the standard FPGA bitstream (including the firmware).
If it is not then please refer to
[quickstart.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/quickstart.md)
(in the tillitis-key1 repository) for instructions on initial
programming of the USB stick.

### Users on Linux

Running `lsusb` should list the USB stick as `1207:8887 Tillitis
MTA1-USB-V1`. On Linux, the TKey's serial port device path is
typically `/dev/ttyACM0` (but it may end with another digit, if you
have other devices plugged in). The host programs tries to auto-detect
serial ports of TKey USB sticks, but if more than one is found you'll
need to choose one using the `--port` flag.

However, you should make sure that you can read and write to the
serial port as your regular user.

One way to accomplish this is by installing the provided
`system/60-tkey.rules` in `/etc/udev/rules.d/` and running `udevadm
control --reload`. Now when a TKey is plugged in, its device path
(like `/dev/ttyACM0`) should be read/writable by you who are logged in
locally (see `loginctl`).

Another way is becoming a member of the group that owns the serial
port. On Ubuntu that group is `dialout`, and you can do it like this:

```
$ id -un
exampleuser
$ ls -l /dev/ttyACM0
crw-rw---- 1 root dialout 166, 0 Sep 16 08:20 /dev/ttyACM0
$ sudo usermod -a -G dialout exampleuser
```

For the change to take effect everywhere you need to logout from your
system, and then log back in again. Then logout from your system and
log back in again. You can also (following the above example) run
`newgrp dialout` in the terminal that you're working in.

Your TKey is now running the firmware. Its LED is a steady white,
indicating that it is ready to receive an app to run.

#### User on MacOS

You can check that the OS has found and enumerated the USB stick by
running:

```
ioreg -p IOUSB -w0 -l
```

There should be an entry with `"USB Vendor Name" = "Tillitis"`.

Looking in the `/dev` directory, there should be a device named like
`/dev/tty.usbmodemXYZ`. Where XYZ is a number, for example 101. This
is the device path that might need to be passed as `--port` when
running the host programs.

### Running apps in QEMU

Build our [qemu](https://github.com/tillitis/qemu). Use the `tk1`
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
if you have any issues building.

Then run the emulator, passing using the built firmware to "-bios":

```
$ /path/to/qemu/build/qemu-system-riscv32 -nographic -M tk1,fifo=chrid -bios firmware.elf \
  -chardev pty,id=chrid
```

It tells you what serial port it is using, for instance `/dev/pts/1`.
This is what you need to use as `--port` when running the host
programs.

The TK1 machine running on QEMU (which in turn runs the firmware, and
then the app) can output some memory access (and other) logging. You
can add `-d guest_errors` to the qemu commandline To make QEMU send
these to stderr.


## The ed25519 signer app

This is a message signer, for root of trust and SSH authentication
using ed25519. There are two host programs which can communicate with
the app. `tkey-sign` just performs a complete test signing.
`tkey-ssh-agent` is an SSH agent that allow using the signer for SSH
remote access.

### Using runsign.sh, tkey-runapp, and tkey-sign

If you're running on hardware, the LED on the USB stick is expected to
be a steady white, indicating that firmware is ready to receive an app
to run.

There's a script called `runsign.sh` which runs `tkey-runapp` to load
the signer app onto TKey and start it. The script then runs
`tkey-sign` which communicates with the signer app to make it sign a
message and then verifies the signature. You can use it like this:

```
./runsign.sh file-with-message
```

The signer app can sign messages of up to 4096 bytes. If the `--port`
flags needs to be used, you can pass it after the message argument.

The host program `tkey-runapp` only loads and starts an app. You'll
then have to switch to a different program that speaks your app's
specific protocol. For instance the `tkey-sign` program provided here.

To run `tkey-runapp` you need to pass it the raw app binary that
should be run (and possibly `--port`, if the auto-detection is not
sufficient).

```
$ ./tkey-runapp apps/signer/app.bin
```
While the app is being loaded, the LED on the USB stick (in one of the
outer corners) will be turned off. `tkey-runapp` also supports sending
a User Supplied Secret (USS) to the firmware when loading the app. By
adding the flag `--uss`, you will be asked to type a phrase which will
be hashed to become the USS digest (the final newline is removed from
the phrase before hashing).

Alternatively, you may use `--uss-file=filename` to make it read the
contents of a file, which is then hashed as the USS. The filename can
be `-` for reading from stdin. Note that all data in file/stdin is
read and hashed without any modification.

The firmware uses the USS digest, together with a hash digest of the
application binary, and the Unique Device Secret (UDS, unique per
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

If you're using real hardware, the LED on the USB stick is a steady
green while the app is receiving data to sign. The LED then flashes
green, indicating that you're required to physically touch the USB
stick for the signing to complete. The touch sensor is located next to
the flashing LED -- touch it and release. If running on QEMU, the
virtual device is always touched automatically.

The program should eventually output a signature and say that it was
verified.

When all is done, the hardware USB stick will flash a nice blue,
indicating that it is ready to make (another) signature.

Note that to load a new app, the USB stick needs to be unplugged and
plugged in again. Similarly, QEMU would need to be restarted (`Ctrl-a
x` to quit). If you're using the setup with the USB stick sitting in
the programming jig and at the same time plugged into the computer,
then you need to unplug both the USB stick and the programmer. Or
alternatively run the `reset-tk1` script (in the tillitis-key1 repo).

That was fun, now let's try the SSH agent!

### Using tkey-ssh-agent

This host program for the signer app is a complete, alternative SSH
agent with practical use. The signer app binary gets built into the
tkey-ssh-agent, which will load it onto USB stick when started. Like
the other host programs, tkey-ssh-agent tries to auto-detect serial
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

### Disabling touch requirement

The signer app normally requires the USB stick to be physically
touched for signing to complete. For special purposes it can be
compiled with this requirement removed, by setting the environment
variable `TKEY_SIGNER_APP_NO_TOUCH` to some value when building.
Example: `make TKEY_SIGNER_APP_NO_TOUCH=yesplease`.

The host apps will also stop displaying this requirement. Of course
this changes the signer app binary and as a consequence the derived
private key and identity will change.

## The RNG stream app

This app generates a continuous stream of high quality random numbers
that can be read from the TKey's serial port device. In Linux for
example like: `dd bs=1 count=1024 if=/dev/ttyACM0 of=rngdata` (or just
a plain `cat`).

The app can be loaded and started using the `tkey-runapp` as described
above.

The RNG is a Hash_DRBG implementation using the BLAKE2s hash function
as primitive. The generator will extract at most 128 bits from each
hash operation, using 128 bits as exclusive evolving state. The RNG
will be reseeded after 1000 hash operations. Reseeding is done by
extracting 256 entropy bits from the TK1 TRNG core. Note that the
reseed rate can be changed during compile time by adjusting the
RESEED_TIME define in main.c.


## Example blink app in assembler

In `blink/` there is also a very, very simple app written in
assembler, `blink.bin` (blink.S) that blinks the LED.


## Developing apps

Device apps and libraries are kept under the `apps` directory. A C
runtime is provided as `apps/libcrt0/libcrt0.a` which you can link
your C apps with.

### Memory

RAM starts at 0x4000\_0000 and ends at 0x4002\_0000 (128 KB). The app
will be loaded by firmware at RAM start. The stack for the app is
setup to start just below RAM end (see
[apps/libcrt0/crt0.S](apps/libcrt0/crt0.S)). A larger app comes at a
compromise of it having a smaller stack.

There are no heap allocation functions, no `malloc()` and friends.

Special memory areas for memory mapped hardware functions are
available at base 0xc000\_0000 and an offset. See
[software.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/system_description/software.md)
(in the tillitis-key1 repository), and the include file `tk1_mem.h`.

### Debugging

If you're running the app on our qemu emulator we have added a debug
port on 0xfe00\_1000 (TK1_MMIO_QEMU_DEBUG). Anything written there
will be printed as a character by qemu on the console.

`qemu_putchar()`, `qemu_puts()`, `qemu_putinthex()`, `qemu_hexdump()`
and friends (see `apps/libcommon/lib.[ch]`) use this debug port to
print stuff.

`libcommon` is compiled with no debug output by default. Rebuild
`libcommon` without `-DNODEBUG` to get the debug output.
