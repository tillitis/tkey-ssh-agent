# signerapp

An ed25519 signer app written in C to run on MTA1-MKDF.

To build you need the `clang`, `llvm` and `lld` packages installed. And they
need to have risc32 support, check this with `llc --version | grep riscv32`.
Build everything:

```
$ make
```

If your available `objcopy` the default `llvm-objcopy-14`, then define
`OBJCOPY` to whatever they're called on your system.

## Real hardware or QEMU platform

The signerapp can be run both on the hardware Tillitis Key1, and on a QEMU
machine that emulates the platform. In both cases, the host program (`runapp`
or `mkdf-ssh-agent` that is running on your computer) will talk to the app over
a serial port, virtual or real. Please continue below in the hardware or QEMU
section.

### Running on hardware, the Tillitis Key1

Plug the Key1 into your computer. `lsusb` should list it as `1207:8887 Tillitis
MTA1-USB-V1`. On Linux, the Key1's serial port path is typically
`/dev/ttyACM0`. This is also the path that the host programs use by default.
You can list the possible paths using `mkdf-ssh-agent --list-ports`.

You also need to be able to access the serial port path as your regular user.
One way is by becoming a member of the group that owns the serial port. You can
do this using something like:

```
$ id -un
exampleuser
$ ls -l /dev/ttyACM0
crw-rw---- 1 root dialout 166, 0 Sep 16 08:20 /dev/ttyACM0
$ sudo usermod -a -G dialout exampleuser
```

Then logout from your system and log in again, for the change to take effect.
You can also run `newgrp dialout` in the terminal that you're working in.

Now you're ready to build the FPGA bitstream (including the firmware) and
program it into the flash of the device.

- TODO refer to https://github.com/tillitis/tillitis-key1 README.md which
  should explain how to plug in for programming, and getting all built and the
  flash programmed. Note, we also talk above about plugging in the device ^

Your Key1 device should now be running the firmware and its LED should flash
white. You should also have learned what serial port path to use for accessing
it. You may need to pass this as `--port` when running the host programs.
Continue in the section below, "Using runapp".

### Running on QEMU

Build our [qemu](https://github.com/tillitis/qemu). Use the `mta1` branch:

```
$ git clone -b mta1 https://github.com/tillitis/qemu
$ mkdir qemu/build
$ cd qemu/build
$ ../configure --target-list=riscv32-softmmu
$ make -j $(nproc)
```

Build the [firmware](https://github.com/mullvad/mta1-mkdf-firmware-priv).

Then run the emulator:

```
$ <path-to-qemu>/build/qemu-system-riscv32 -nographic -M mta1_mkdf,fifo=chrid -bios firmware \
       -chardev pty,id=chrid
```

It tells you what serial port it is using, for instance `/dev/pts/1`. This is
what you need to use as `--port` when running the host programs. Continue in
the section below, "Using runapp".

The the MTA1 machine running on QEMU (which in turn runs the firmware, and then
the app) can output some memory access (and other) logging. You can add `-d
guest_errors` to the qemu commandline To make QEMU send these to stderr.

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

If you're on hardware, the LED on the device is a steady green while the app is
receiving data to sign. The LED then flashes green, indicating that you're
required to touch the device for the signing to complete. If running on QEMU,
the virtual device is always touched automatically.

The program should eventually output a signature and say that it was verified.

When all is done, the hardware device will flash a nice blue, indicating that
it is ready to make (another) signature.

If `--file` is not passed, the app is assumed to be loaded and running already,
and signing is attempted right away.

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

It will also output the ed25519 public key for this instance of the app on this
key device. If the app binary, or the physical key device changes, then the
private key will also change -- and thus also the public key displayed!

If you copy-paste the public key into your `~/.ssh/authorized_keys` you can try
to log onto your local computer (if sshd is running there). The socket path
set/output above is also needed by ssh in `SSH_AUTH_SOCK`:

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

RAM starts at 0x4000\_0000 and ends at 0x4002\_0000. Your program
will be loaded by firmware at 0x4001\_0000 which means a maximum size
including `.data` and `.bss` of 64 kiB. In this app (see `crt0.S`) you
have 64 kiB of stack from 0x4000\_ffff down to where RAM starts.

There are no heap allocation functions, no `malloc()` and friends.

Special memory areas for memory mapped hardware functions are
available at base 0xc000\_0000 and an offset. See [MTA1-MKDF
software](https://github.com/mullvad/mta1_mkdf/blob/main/doc/system_description/software.md)
and the include file `mta1_mkdf_mem.h`.

### Debugging

If you're running the app on our qemu emulator we have added a debug
port on 0xfe00\_1000 (MTA1_MKDF_MMIO_QEMU_DEBUG). Anything written
there will be printed as a character by qemu on the console.

`putchar()`, `puts()`, `putinthex()`, `hexdump()` and friends (see
`signerapp/lib.[ch]`) use this debug port to print stuff. If you compile
with `-DNODEBUG` all these are no-ops.
