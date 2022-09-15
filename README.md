
# signerapp

An ed25519 signer app written in C to run on MTA1-MKDF.

To build you need the `clang`, `llvm` and `lld` packages installed. And they
need to have risc32 support, check this with `llc --version | grep riscv32`.
Build everything:

```
make
```

Build our [qemu](https://github.com/tillitis/qemu). Use
the `mta1` branch:

```
% git clone -b mta1 https://github.com/tillitis/qemu
% mkdir qemu/build
% cd qemu/build
% ../configure --target-list=riscv32-softmmu
% make -j $(nproc)
```

Build [the firmware](https://github.com/mullvad/mta1-mkdf-firmware-priv).

Then run the emulator:

```
% <path-to-qemu>/build/qemu-system-riscv32 -nographic -M mta1_mkdf,fifo=chrid -bios firmware \
       -chardev pty,id=chrid
```

It tells you what serial device it is using, for instance `/dev/pts/0`.

Then run the host program, specifying both the serial device from QEMU and the
raw binary app you want to run:

```
% ./runapp --port /dev/pts/0 --file signerapp/app.bin
```

which should give you a signature on the output.

If `--file` is not passed, the app is assumed to be loaded and running on the
emulated device, and signing is attempted.

The mta1 guest machine running in QEMU (which in turn runs the firmware and
then the app) outputs some memory access (and other) logging. To make QEMU send
these to stderr, add `-d guest_errors` to the qemu commandline.

# fooapp

In `fooapp/` there is also a very, very simple app written in assembler,
foo.bin (foo.S) that blinks the LED.

# Using mkdf-ssh-agent

The signer app gets build into mkdf-ssh-agent, which will upload it to the
device when started. You can start it like this:

```
% ./mkdf-ssh-agent -a ./agent.sock --port /dev/pts/0
```

This will start the ssh-agent, listening on the specified socket. It will also
output the ed25519 public key for this instance of the app on this device. If
the app binary, or the physical device changes, then the private key will also
change -- and thus also the public key displayed!

If you copy-paste the public key into your `~/.ssh/authorized_keys` you can try
to log onto your local machine (if sshd is running there). Also note the
listening socket path in the output above, which ssh needs in `SSH_AUTH_SOCK`:

```
% SSH_AUTH_SOCK=/path/to/agent.sock ssh -F /dev/null localhost
```

(`-F /dev/null` is to not have any of your ~/.ssh/config interfere)

The message `agent 27: ssh: parse error in message type 27` coming from
mkdf-ssh-agent is due to https://github.com/golang/go/issues/51689 and will
eventually be fixed by https://go-review.googlesource.com/c/crypto/+/412154/
(until then it's also not possible to implement the upcoming SSH agent
restrictions https://www.openssh.com/agent-restrict.html).

You can use `-k` (long option: `--show-pubkey`) to only output the pubkey (on
stdout, some message are still present on stderr), which can be practical.

% ./mkdf-ssh-agent -k --port /dev/pts/0

# Developing the app

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
