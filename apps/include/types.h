// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

#ifndef TYPES_H
#define TYPES_H

typedef unsigned int uintptr_t;
typedef unsigned long long uint64_t;
typedef unsigned int uint32_t;
typedef int int32_t;
typedef long long int64_t;
typedef unsigned char uint8_t;
typedef char int8_t;
typedef signed short int int16_t;
typedef unsigned long size_t;

#define NULL ((char *)0)

// blake2s state context
typedef struct {
	uint8_t b[64]; // input buffer
	uint32_t h[8]; // chained state
	uint32_t t[2]; // total number of bytes
	size_t c;      // pointer for b[]
	size_t outlen; // digest size
} blake2s_ctx;

#endif
