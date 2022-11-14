// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

// rng stream extraction app.
//
// When loaded and started, this app will continiously generate random data
// words and send them to the host as a stream of bytes.

#include "../blake2s/blake2s.h"
#include "../include/tk1_mem.h"

#define RESEED_TIME 1000

// clang-format off
static volatile uint32_t *led =            (volatile uint32_t *)TK1_MMIO_TK1_LED;
static volatile uint32_t *cdi =            (volatile uint32_t *)TK1_MMIO_TK1_CDI_FIRST;
static volatile uint32_t *trng_status =    (volatile uint32_t *)TK1_MMIO_TRNG_STATUS;
static volatile uint32_t *trng_entropy =   (volatile uint32_t *)TK1_MMIO_TRNG_ENTROPY;
static volatile uint32_t *uart_tx_status = (volatile uint32_t *)TK1_MMIO_UART_TX_STATUS;
static volatile uint32_t *uart_tx_data =   (volatile uint32_t *)TK1_MMIO_UART_TX_DATA;
// clang-format on

// state context
typedef struct {
	uint32_t ctr;
	uint32_t state[16];
} rng_ctx;

void transmit_w32(uint32_t w)
{
	while (!*uart_tx_status) {
	}
	*uart_tx_data = w >> 24;

	while (!*uart_tx_status) {
	}
	*uart_tx_data = w >> 16 & 0xff;

	while (!*uart_tx_status) {
	}
	*uart_tx_data = w >> 8 & 0xff;

	while (!*uart_tx_status) {
	}
	*uart_tx_data = w & 0xff;
}

void output_rnd(uint32_t *random_data)
{
	for (int i = 0; i < 4; i++) {
		transmit_w32(random_data[i]);
	}
}

uint32_t get_w32_entropy()
{
	while (!*trng_status) {
	}
	return *trng_entropy;
}

void init_rng_state(rng_ctx *ctx)
{
	for (int i = 0; i < 8; i++) {
		ctx->state[i] = cdi[i];
		ctx->state[i + 8] = get_w32_entropy();
	}

	ctx->ctr = 0;
}

void update_rng_state(rng_ctx *ctx, uint32_t *digest)
{
	for (int i = 0; i < 8; i++) {
		ctx->state[i] = digest[i];
	}

	ctx->ctr += 1;
	ctx->state[15] += ctx->ctr;

	if (ctx->ctr == RESEED_TIME) {
		for (int i = 0; i < 8; i++) {
			ctx->state[i + 8] = get_w32_entropy();
		}
		*led += +1;
		ctx->ctr = 0;
	}
}

int main(void)
{
	uint32_t digest[8];
	rng_ctx ctx;

	init_rng_state(&ctx);

	for (;;) {
		blake2s(&digest[0], 32, NULL, 0, &ctx.state[0], 64);
		output_rnd(&digest[0]);
		update_rng_state(&ctx, &digest[0]);
	}
}
