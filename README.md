# signerapp

An ed25519 signer app written in C to run on the Tillitis Key 1.

To build you need the `clang`, `llvm` and `lld` packages installed. And they
need to have risc32 support, check this with `llc --version | grep riscv32`.
Please see
[toolchain_setup.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/toolchain_setup.md)
(in the tillitis-key1 repository) for information on the currently supported
build and development environment.

Build everything:

```
$ make
```

If your available `objcopy` is anything other than the default
`llvm-objcopy-14`, then define `OBJCOPY` to whatever they're called on your
system.

## Real hardware or QEMU platform

The signerapp can be run both on the hardware Tillitis Key 1, and on a QEMU
machine that emulates the platform. In both cases, the host program (`runapp`
or `mkdf-ssh-agent` running on your computer) will talk to the app over a
serial port, virtual or real. Please continue below in the hardware or QEMU
section.

### Running on hardware device -- Tillitis Key 1

Plug the USB device into your computer. If the LED at in one of the outer
corners of the device is flashing white, then it has been programmed with the
standard FPGA bitstream (including the firmware). If it is not then please
refer to
[quickstart.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/quickstart.md)
(in the tillitis-key1 repository) for instructions on initial programming of
the device.

Running `lsusb` should list the device as `1207:8887 Tillitis MTA1-USB-V1`. On
Linux, Tillitis Key 1's serial port path is typically `/dev/ttyACM0` (but it
may end with another digit, if you have other devices plugged in). This is also
the default path that the host programs use to talk to it. You can list the
possible paths using `mkdf-ssh-agent --list-ports`.

You also need to be sure that you can access the serial port as your regular
user. One way to do that is by becoming a member of the group that owns the
serial port. You can do that like this:

```
$ id -un
exampleuser
$ ls -l /dev/ttyACM0
crw-rw---- 1 root dialout 166, 0 Sep 16 08:20 /dev/ttyACM0
$ sudo usermod -a -G dialout exampleuser
```

For the change to take effect everywhere you need to logout from your system,
and then log back in again. Then logout from your system and log back in
again. You can also (following the above example) run `newgrp dialout` in the
terminal that you're working in.

Your Tillitis Key 1 is now running the firmware. Its LED flashing white,
indicating that it is ready to receive an app to run. You have also learned
what serial port path to use for accessing it. You may need to pass this as
`--port` when running the host programs. Continue in the section below, "Using
runapp".


### Running on MacOS
After building the tillitis-key1-apps (see above) and connected a Tillitis Key 1
device with the firmware, you should be able to use the device.

You can check that the OS has found and enumerated the device by running:
Kommando för att lista USB devices in MacOS:

```
ioreg -p IOUSB -w0 -l

```

There should be an entry with:

```
"USB Vendor Name" = "Tillitis"
```

Looking in the dev directory, there should be a device:

```
/dev/tty.usbmodemXYZ
```

Where XYZ is a number, for example 101.

You should now be able to load and run an application
on the device. For example:

```
 ./runapp --port /dev/tty.usbmodem101 --file signerapp/app.bin
```


### Running on QEMU

Build our [qemu](https://github.com/tillitis/qemu). Use the `mta1` branch:

```
$ git clone -b mta1 https://github.com/tillitis/qemu
$ mkdir qemu/build
$ cd qemu/build
$ ../configure --target-list=riscv32-softmmu
$ make -j $(nproc)
```

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
$ <path-to-qemu>/build/qemu-system-riscv32 -nographic -M mta1_mkdf,fifo=chrid -bios firmware.elf \
       -chardev pty,id=chrid
```

It tells you what serial port it is using, for instance `/dev/pts/1`. This is
what you need to use as `--port` when running the host programs. Continue in
the section below, "Using runapp".

The the MTA1 machine running on QEMU (which in turn runs the firmware, and
then the app) can output some memory access (and other) logging. You can add
`-d guest_errors` to the qemu commandline To make QEMU send these to stderr.

## Using runapp

By now you should have learned which serial port to use from one of the
"Running on"-sections. If you're running on hardware, the LED on the device is
expected to be flashing white, indicating that firmware is ready to receive an
app to run.

The host program `runapp` performs a complete, verbose signing. To run the
program you need to specify both the serial port and the raw app binary that
should be run. The port used below is just an example.

```
$ ./runapp --port /dev/pts/1 --file signerapp/app.bin
```

If you're on hardware, the LED on the device is a steady green while the app
is receiving data to sign. The LED then flashes green, indicating that you're
required to touch the device for the signing to complete. The touch sensor is
located next to the flashing LED -- touch and release. If running on QEMU, the
virtual device is always touched automatically.

The program should eventually output a signature and say that it was verified.

When all is done, the hardware device will flash a nice blue, indicating that
it is ready to make (another) signature.

If `--file` is not passed, the app is assumed to be loaded and running
already, and signing is attempted right away.

That was fun, now let's try the ssh-agent!

## Using mkdf-ssh-agent

This host program for the signerapp is a complete, alternative ssh-agent with
practical use. The signerapp binary gets built into the mkdf-ssh-agent, which
will upload it to the device when started. If the serial port path is not the
default, you need to pass it as `--port`. An example:

```
$ ./mkdf-ssh-agent -a ./agent.sock --port /dev/pts/1
```

This will start the ssh-agent and tell it to listen on the specified socket
`./agent.sock`.

It will also output the ed25519 public key for this instance of the app on
this key device. If the app binary, or the physical key device changes, then
the private key will also change -- and thus also the public key displayed!

If you copy-paste the public key into your `~/.ssh/authorized_keys` you can
try to log onto your local computer (if sshd is running there). The socket
path set/output above is also needed by ssh in `SSH_AUTH_SOCK`:

```
$ SSH_AUTH_SOCK=/path/to/agent.sock ssh -F /dev/null localhost
```

`-F /dev/null` is used to ignore your ~/.ssh/config which could interfere with
this test.

The message `agent 27: ssh: parse error in message type 27` coming from
mkdf-ssh-agent is due to https://github.com/golang/go/issues/51689 and will
eventually be fixed by https://go-review.googlesource.com/c/crypto/+/412154/
(until then it's also not possible to implement the upcoming SSH agent
restrictions https://www.openssh.com/agent-restrict.html).

You can use `-k` (long option: `--show-pubkey`) to only output the pubkey (on
stdout, some message are still present on stderr), which can be useful:

```
$ ./mkdf-ssh-agent -k --port /dev/pts/1
```

# fooapp

In `fooapp/` there is also a very, very simple app written in assembler,
foo.bin (foo.S) that blinks the LED.

# Developing apps

## Memory

RAM starts at 0x4000\_0000 and ends at 0x4002\_0000. Your program will be
loaded by firmware at 0x4001\_0000 which means a maximum size including
`.data` and `.bss` of 64 kiB. In this app (see `crt0.S`) you have 64 kiB of
stack from 0x4000\_ffff down to where RAM starts.

There are no heap allocation functions, no `malloc()` and friends.

Special memory areas for memory mapped hardware functions are available at
base 0xc000\_0000 and an offset. See
[software.md](https://github.com/tillitis/tillitis-key1/blob/main/doc/system_description/software.md)
(in the tillitis-key1 repository), and the include file `mta1_mkdf_mem.h`.

### Debugging

If you're running the app on our qemu emulator we have added a debug port on
0xfe00\_1000 (MTA1_MKDF_MMIO_QEMU_DEBUG). Anything written there will be
printed as a character by qemu on the console.

`putchar()`, `puts()`, `putinthex()`, `hexdump()` and friends (see
`signerapp/lib.[ch]`) use this debug port to print stuff. If you compile with
`-DNODEBUG` all these are no-ops.

# Licensing

See [LICENSES](./LICENSES/README.md) for more information about the projects'
licenses.
