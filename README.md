# mta1signer

Simple app to run on MTA1-MKDF.

There are really two apps, one very, very simple assembler app,
foo.bin (foo.S), and one slightly larger C app: app.bin.

The larger C app is an ed25519 signer.

You need `riscv32-elf-binutils`.

To build everything:

```
make
```

Build our [qemu](https://github.com/mullvad/mta1-mkdf-qemu-priv). Use
the `mta1-regs` branch.

Build [the firmware](https://github.com/mullvad/mta1-mkdf-firmware-priv).

Then run the emulator:

```
% <path-to-qemu>/build/qemu-system-riscv32 -nographic -M mta1_mkdf,fifo=chrid -bios firmware \
	-chardev socket,host=127.0.0.1,port=4444,server=on,wait=off,id=chrid
```

Then run the host program:

```
%  ./runapp -file app/app.bin
```

which should give you a signature on the output.

If -file is not passed, the app is assumed to be loaded and running on the
device, and signing is attempted.
