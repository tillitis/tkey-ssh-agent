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
the `mta1-regs` branch.

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

If you have followed the above, the signer app has now been loaded and is
running on the emulated device in QEMU. You can now start up our mkdf-ssh-agent
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
