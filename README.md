# mkdfsigner

Simple app to run on MTA1-MKDF.

There are really two apps, one very, very simple assembler app,
foo.bin (foo.S), and one slightly larger C app: app.bin.

The larger C app is an ed25519 signer.

To build you need the `clang`, `llvm` and `lld` packages installed. And they
need to have risc32 support, check this with `llc --version | grep riscv32`.
Build everything:

```
make
```

Build our [qemu](https://github.com/mullvad/mta1-mkdf-qemu-priv). Use
the `mta1-regs` branch:

```
% mkdir build
% cd build
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
%  ./runapp -port /dev/pts/0 -file app/app.bin
```

which should give you a signature on the output.

If -file is not passed, the app is assumed to be loaded and running on the
emulated device, and signing is attempted.

# Using mkdf-ssh-agent

The mkdf-ssh-agent should be able to upload the app itself. You can start it
like this:

```
% ./mkdf-ssh-agent -a ./agent.sock -port /dev/pts/0
```

This will output the unique public key for the instance of the app on this
device. If you copy-paste this into your `~/.ssh/authorized_keys` you can try
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

## Developing the app

### Memory

RAM starts at 0x8000\_0000 and ends at 0x8002\_0000. Your program
will be loaded by firmware at 0x8001\_0000 which means a maximum size
including `.data` and `.bss` of 64 kiB. In this app (see `crt0.S`) you
have 64 kiB of stack from 0x8000\_ffff down to where RAM starts.

There are no heap allocation functions, no `malloc()` and friends.

Special memory areas for memory mapped hardware functions are
available at base 0x9000\_0000 and an offset. See [MTA1-MKDF
software](https://github.com/mullvad/mta1_mkdf/blob/main/doc/system_description/software.md).

### Debugging

If you're running the app on our qemu emulator we have added a debug
port on 0x9000\_1000. Anything written there will be printed as a
character by qemu on the console.

`putchar()`, `puts()`, `putinthex()`, `hexdump()` and friends (see
`app/lib.[ch]`) use this debug port to print stuff. If you compile
with `-DNODEBUG` all these are no-ops.
