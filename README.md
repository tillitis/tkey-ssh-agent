# mkdf-app

Simple app to run on MTA1-MKDF.

There are really two apps, one very, very simple assembler app,
foo.bin (foo.S), and one slightly larger C app: app.bin.

They are supposed to be loaded by the
[mta1](https://github.com/mullvad/mta1-mkdf-host-priv) host program
like this:

```
% mta1 load-app app.bin
```
