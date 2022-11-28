
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
