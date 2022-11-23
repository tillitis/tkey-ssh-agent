# Tillitis Key 1 Apps

This repository contains applications to run on the Tillitis Key 1.

Current list of apps:
- The Ed25519 signerapp. Used as root of trust and SSH authentication
- The random app.
- The RNG stream app. Providing arbitrary high quality random numbers
- fooapp. A minimalistic example application

For more information about the apps, see subsections below.

Note that development is ongoing. For example, changes might be made
to the signerapp, causing the public/private keys it provides to
change. To avoid unexpected changes, please use a tagged release.


## Building apps

To build you need the `clang`, `llvm`, `lld`, `golang` packages
installed. clang/llvm need to have riscv32 support, check this with
`llc --version | grep riscv32`. Ubuntu 22.10 (Kinetic) is known to
work. Please see
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

The signerapp can be run both on the hardware Tillitis Key 1, and on a
QEMU machine that emulates the platform. In both cases, the host
program (`runapp`, `tk-sign` or `tk-ssh-agent` running on your
computer) will talk to the app over a serial port, virtual or real.
There is a separate section below which explains running in QEMU.


## Running apps

Plug the USB stick into your computer. If the LED at in one of the
outer corners of the USB stick is flashing white, then it has been
programmed with the standard FPGA bitstream (including the firmware).
If it is not then please refer to
[quickstart.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/quickstart.md)
(in the tillitis-key1 repository) for instructions on initial
programming of the USB stick.

### Users on Linux

Running `lsusb` should list the USB stick as `1207:8887 Tillitis
MTA1-USB-V1`. On Linux, Tillitis Key 1's serial port device path is
typically `/dev/ttyACM0` (but it may end with another digit, if you
have other devices plugged in). The host programs tries to auto-detect
serial ports of Tillitis USB sticks, but if more than one is found
you'll need to choose one using the `--port` flag.

However, you should make sure that you can access the serial port as
your regular user. One way to do that is by becoming a member of the
group that owns the serial port. On Ubuntu that group is `dialout`,
and you can do it like this:

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

Your Tillitis Key 1 is now running the firmware. Its LED flashing
white, indicating that it is ready to receive an app to run.

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
branch:

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


## The ed25519 signerapp

This is a message signer, for root of trust and SSH authentication
using ed25519. There are two host programs which can communicate with
the app. `tk-sign` just performs a complete test signing.
`tk-ssh-agent` is an ssh-agent that allow using the signer for SSH
remote access.

### Using runsign.sh, runapp, and tk-sign

If you're running on hardware, the LED on the USB stick is expected to
be flashing white, indicating that firmware is ready to receive an app
to run.

There's a script called `runsign.sh` which runs `runapp` to load and
start the signerapp. It then runs `tk-sign` which asks the app to sign
a message and verifies it. You can use it like this:

```
./runsign.sh file-with-message
```

The file with the message can currently be at most 4096 bytes long. If
the `--port` flags needs to be used, you can pass it after the message
argument.

The host program `runapp` only loads and starts an app. Then you will
have to switch to a different program to speak your specific app
protocol, for instance the `tk-sign` program provided here.

To run `runapp` you need to specify both the serial port (unless
you're using the default `/dev/ttyACM0`) and the raw app binary that
should be run. The port used below is just an example.

```
$ ./runapp --port /dev/pts/1 --file apps/signerapp/app.bin
```

The `runapp` also supports sending a User Supplied Secret (USS) to the
firmware when loading the app. By adding the flag `--uss`, you will be
asked to type a phrase which will be hashed to become the USS digest
(the final newline is removed from the phrase before hashing).

Alternatively, you may use `--uss-file=filename` to make it read the
contents of a file, which is then hashed as the USS. The filename can
be `-` for reading from stdin. Note that all data in file/stdin is
read and hashed without any modification.

The firmware uses the USS digest, together with a hash digest of the
application binary, and the Unique Device Secret (UDS, unique per
physical device) to derive secrets for use by the application.

The practical result for users of the signerapp is that the ed25519
public/private keys will change along with the USS. So if you enter a
different phrase (or pass a file with different contents), the derived
USS will change, and so will your identity. To learn more, read the
[system_description.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/system_description/system_description.md)
(in the tillitis-key1 repository).

`tk-sign` is used in a similar way, assuming `runapp` has been used to
load the signerapp:

```
./tk-sign --port /dev/pts/1 --file file-with-message-to-sign
```

If you're using real hardware, the LED on the USB stick is a steady
green while the app is receiving data to sign. The LED then flashes
green, indicating that you're required to touch the USB stick for the
signing to complete. The touch sensor is located next to the flashing
LED -- touch it and release. If running on QEMU, the virtual device is
always touched automatically.

The program should eventually output a signature and say that it was
verified.

When all is done, the hardware USB stick will flash a nice blue,
indicating that it is ready to make (another) signature.

If `--file` is not passed, the app is assumed to be loaded and running
already, and signing is attempted right away.

Note that to load a new app, the USB stick needs to be unplugged and
plugged in again. Similarly, QEMU would need to be restarted (`Ctrl-a
x` to quit). If you're using the setup with the USB stick sitting in
the programming jig and at the same time plugged into the computer,
then you need to unplug both the USB stick and the programmer. Or
alternatively run the `reset-tk1` script (in the tillitis-key1 repo).

That was fun, now let's try the ssh-agent!

### Using tk-ssh-agent

This host program for the signerapp is a complete, alternative
ssh-agent with practical use. The signerapp binary gets built into the
tk-ssh-agent, which will load it onto USB stick when started. Like the
other host programs, tk-ssh-agent tries to auto-detect serial ports of
Tillitis USB sticks. If more than one is found, or if you're running
on QEMU, then you'll need to use the `--port` flag. An example of
that:

```
$ ./tk-ssh-agent -a ./agent.sock --port /dev/pts/1
```

This will start the ssh-agent and tell it to listen on the specified
socket `./agent.sock`.

It will also output the SSH ed25519 public key for this instance of
the app on this specific Tillitis Key USB stick. So again; if the
signerapp binary, the USS, or the UDS in the physical USB stick
change, then the private key will also change -- and thus the derived
public key, your public identity in the world of SSH.

If you copy-paste the public key into your `~/.ssh/authorized_keys`
you can try to log onto your local computer (if sshd is running
there). The socket path set/output above is also needed by SSH in
`SSH_AUTH_SOCK`:

```
$ SSH_AUTH_SOCK=/path/to/agent.sock ssh -F /dev/null localhost
```

`-F /dev/null` is used to ignore your ~/.ssh/config which could
interfere with this test.

The tk-ssh-agent also supports the `--uss` and `--uss-file` flags, as
described for `runapp` above.

You can use `-k` (long option: `--show-pubkey`) to only output the
pubkey. The pubkey is printed to stdout for easy redirection, but some
messages are still present on stderr.


## The random app and runrandom host program

The random app is a random number generator that uses Tillitis Key 1's
TRNG (True Random Number Generator). The hardware stick will flash the
red and blue LEDs while the app is running and show the same colors
steadily while generating/delivering the numbers.

The companion host program `runrandom` can be used to output random
numbers on stdout. The host program embeds the app binary and loads it
onto the USB stick if needed. If the stick is not in firmware mode, or
is running another app, you'll need to unplug and plug it in again.
You can build and use it like this:

```
$ make runrandom
$ ./runrandom -b 42 | hexdump
```


## The RNG stream app

This app generates a continuous stream of high quality random numbers
that can be read from the USB device endpoint (for example
/dev/ttyACM0 in Linux).

The app can be loaded and started using the `runapp` described above.

The RNG is a Hash_DRBG implementation using the BLAKE2s hash function
as primitive. The generator will extract at most 128 bits from each
hash operation, using 128 bits as exclusive evolving state. The RNG
will be reseeded after 1000 hash operations. Reseeding is done by
extracting 256 entropy bits from the TK1 TRNG core. Note that the
reseed rate can be changed during compile time by chaining the
RESEED_TIME define in main.c.


## fooapp

In `fooapp/` there is also a very, very simple app written in
assembler, foo.bin (foo.S) that blinks the LED.


## Developing apps

Device apps and libraries are kept under the `apps` directory. A C
runtime is provided as `apps/libcrt0/libcrt0.a` which you can link
your C apps with.

### Memory

RAM starts at 0x4000\_0000 and ends at 0x4002\_0000. The app will be
loaded by firmware at 0x4000\_7000 which means a maximum size
including `.data` and `.bss` of 100 KiB. It will have 28 KiB of stack,
from 0x4000\_6ff0 down to where RAM starts (see
[apps/libcrt0/crt0.S](apps/libcrt0/crt0.S)). A smaller app may want
to move itself in memory in order to have larger continuous memory.

There are no heap allocation functions, no `malloc()` and friends.

Special memory areas for memory mapped hardware functions are
available at base 0xc000\_0000 and an offset. See
[software.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/system_description/software.md)
(in the tillitis-key1 repository), and the include file `tk1_mem.h`.

### Debugging

If you're running the app on our qemu emulator we have added a debug
port on 0xfe00\_1000 (TK1_MMIO_QEMU_DEBUG). Anything written there
will be printed as a character by qemu on the console.

`putchar()`, `puts()`, `putinthex()`, `hexdump()` and friends (see
`apps/libcommon/lib.[ch]`) use this debug port to print stuff.

`libcommon` is compiled with no debug output by default. Rebuild
`libcommon` without `-DNODEBUG` to get the debug output.


## Licensing

See [LICENSES](./LICENSES/README.md) for more information about the projects'
licenses.
