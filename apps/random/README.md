# Random generator

The TKey `Random generator` device application is an hardware based
source of high quality random numbers. The generator is using a
Hash_DRBG built around the BLAKE2s hash function. The application can
also sign the random data in order to provide proof of its origin.

## Client Go application

The corresponding client application can be found in `cmd/runrandom`.

## Application protocol

`Random generator` has a simple protocol on top of the [TKey Framing
Protocol](https://dev.tillitis.se/protocol/#framing-protocol) with the
following requests:

| *command*             | *FP length* | *code* | *data*                              | *response*            |
|-----------------------|-------------|--------|-------------------------------------|-----------------------|
| `CMD_GET_NAMEVERSION` | 1 B         | 0x01   | none                                | `RSP_GET_NAMEVERSION` |
| `CMD_GET_RANDOM`      | 4 B         | 0x03   | Number of bytes, 1 < x < 126        | `RSP_GET_RANDOM`      |
| `CMD_GET_PUBKEY`      | 1 B         | 0x05   | none                                | `RSP_GET_PUBKEY`      |
| `CMD_GET_SIG`         | 1 B         | 0x07   | none                                | `RSP_GET_SIG`         |
| `CMD_GET_HASH`        | 1 B         | 0x09   | none                                | `RSP_GET_HASH`        |


| *response*            | *FP length* | *code* | *data*                             |
|-----------------------|-------------|--------|------------------------------------|
| `RSP_GET_NAMEVERSION` | 32 B        | 0x02   | 2 * 4 bytes name, version 32 bit LE|
| `RSP_GET_RANDOM`      | 128 B       | 0x04   | Up to 126 byte of random data      |
| `RSP_GET_PUBKEY`      | 128 B       | 0x06   | 32 bytes ed25519 public key        |
| `RSP_GET_SIG`         | 128 B       | 0x08   | 64 bytes ed25519 signature         |
| `RSP_GET_HASH`        | 128 B       | 0x0a   | 32 bytes hash                      |

| `RSP_UNKNOWN_CMD`     | 1 B         | 0xff   | none                               |

| *status replies* | *code* |
|------------------|--------|
| OK               | 0      |
| BAD              | 1      |

It identifies itself with:

- `name0`: "tk1  "
- `name1`: "rand"

Please note that `random` also replies with a `NOK` Framing Protocol
response status if the endpoint field in the FP header is meant for
the firmware (endpoint = `DST_FW`). This is recommended for
well-behaved device applications so the client side can probe for the
firmware.

Typical use by a client application:

1. Probe for firmware by sending firmware's `GET_NAME_VERSION` with
   FPheader endpoint = `DST_FW`.
2. If firmware is found, load `random`.
3. Upon receiving the device app digest back from firmware, switch to
   start talking the `random` protocol above.
4. Send `CMD_GET_RANDOM` to recieve generated random data.
5. Repeat step 4 until wanted amount of random data is recieved.
6. Send `CMD_GET_SIG` to calculate and get the signature.
7. Send `CMD_GET_HASH` to receive the generated hash of the random
   data.
8. Send `CMD_GET_PUBKEY` to receive the `random`'s public key. If the
   public key is already stored, check against it so it's the
   expected key.

**Please note**: `CMD_GET_SIG` must be sent before `CMD_GET_HASH` in
order to retrevie the correct hash. The application will return with
and status `BAD`,

**Please note**: The firmware detection mechanism is not by any means
secure. If in doubt a user should always remove the TKey and insert it
again before doing any operation.

## Licenses and SPDX tags

Unless otherwise noted, the project sources are licensed under the
terms and conditions of the "GNU General Public License v2.0 only":

> Copyright Tillitis AB.
>
> These programs are free software: you can redistribute it and/or
> modify it under the terms of the GNU General Public License as
> published by the Free Software Foundation, version 2 only.
>
> These programs are distributed in the hope that they will be useful,
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


## Running

Please see the [Developer
Handbook](https://dev.tillitis.se/tools/#qemu) for [how to run with
QEMU](https://dev.tillitis.se/tools/#qemu) or [how to run apps on a
TKey](https://dev.tillitis.se/devapp/#running-tkey-apps) but generally
to run `random` you either use our
[runrandom](https://github.com/tillitis/tillitis-key1-apps/cmd/runrandom) or
you use our development tool
[tkey-runapp](https://github.com/tillitis/tillitis-key1-apps).

```
$ ./tkey-runapp apps/random/app.bin
$ ./runrandom -b 256 -s
```

Use `--port` if the device port is not automatically detected.

