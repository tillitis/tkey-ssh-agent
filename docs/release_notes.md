# Release notes

## unreleased

- tkey-ssh-agent now connects to the TKey for each SSH agent operation
  (and disconnects afterwards with a delay). The serial port is thus
  left accessible to others.

## v0.0.3

- Update tk1_mem.h and timer app to the revised timer MMIO API

## v0.0.2

We forgot to add the release here when tagging for
https://github.com/tillitis/tillitis-key1-apps/releases/tag/v0.0.2

Notable changes:

- Make monocypher a library - breaks CDI!
- Remove GET_UDI from signer app, and the use of it - breaks CDI!

## v0.0.1

Since we haven't tagged any release before this we list some recent
significant and/or breaking changes.

### Revised SSH Agent

Introduces a revised Tillitis TKey SSH Agent, `tkey-ssh-agent`. The
new agent:

- runs as a daemon all the time (as `systemd` user unit, if you want).
- autodetects TKey removal and insertion with the help of `udev` rules
  (or just send it a `SIGHUP` yourself to make it look for a TKey
  again).
- spawns a graphical `pinentry` program to enter the User-Supplied
  Secret.

The first iteration of this revision of the SSH agent is focused on
Linux distributions and has a Ubuntu/Debian package available.

### Simplified firmware protocol

The firmware protocol for loading a TKey app has changed. We now
combine starting to load an app by setting size and loading USS into a
single request. The firmware automatically returns the app digest and
start the app when the last chunk of the binary has been received.

`GetNameVersion` also now expects an ASCII array for `NAME0` and
`NAME1` both from the firmware and from TKey apps. This also means the
`signerapp` has a new digest and hence a new identity.

### Division no longer available

We now build the TKey apps with the RV32 Zmmul extension since we
removed support for division on the PicoRV32 CPU.
