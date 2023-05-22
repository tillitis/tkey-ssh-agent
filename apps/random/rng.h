// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

#ifndef RNG_H
#define RNG_H

// state context
typedef struct {
	uint32_t ctr;
	uint32_t state[16];
	uint32_t digest[32];
} rng_ctx;

void rng_init(rng_ctx *ctx);
int rng_get(uint32_t *output, rng_ctx *ctx, int bytes);

#endif
