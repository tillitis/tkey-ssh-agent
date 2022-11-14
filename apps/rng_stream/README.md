
# rng_stream

## Introduction

This app will continiously generate random numbers and send to the
host as a stream of bytes. The generator is using a Hash_DRBG built
around the BLAKE2s hash function. The Hash_DRBG is periodically seeded
by the TK1 TRNG entropyy source.

The purpose of the app is to provide a hardware based source of high
quality random numbers to the host.
