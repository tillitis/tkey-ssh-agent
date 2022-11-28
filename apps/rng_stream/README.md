
# rng_stream

## Introduction
This app will continiously generate random numbers and send to the
host as a stream of bytes. The generator is using a Hash_DRBG built
around the BLAKE2s hash function. The Hash_DRBG is periodically seeded
by the TK1 TRNG entropyy source.

The purpose of the app is to provide a hardware based source of high
quality random numbers to the host. This can be useful as the primary
source of  randomness in an embedded system. This can also be used
as a way to improve the availability of randomness in a system,
by feeding the existing random source (/dev/random) with data from
 the generator.

## Usage
Load the app into the TK1 device using 'runapp'. When the
application has been loaded it will start sending random
bytes to the host.

To collect random data simply do:

	'cat /dev/ttyACM0 > rng_data.bin'


## Implementation details

The Hash_DRBG is built around the BLAKE2s hash function.

The internal 512 bit RNG state contain the last 256 bit digest and
256 bit of entropy extracted from the TK1 TRNG. A new complete hash
operation is calculated for each 128 bit block of random number
data generated. 128 bits of the resultinf digest is delivered to the host
and the whole digest is used to update the internal state. Additionally
the reseed counter is mixed into the 256 bits of entropy. This means
that the internal state is updated with at least 128 bits between each
each block delivered to the host.

Currently the generator will reseed after 1000 blocks of data,
that is after 16000 bytes.

The RNG stream application use a version of the BLAKE2s reference
as specified in [RFC 7693](https://www.rfc-editor.org/rfc/rfc7693.html)
by Markku-Juhani O. Saarinen.
