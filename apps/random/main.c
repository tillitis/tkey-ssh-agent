// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

#include <tk1_mem.h>

#include "app_proto.h"

// clang-format off
static volatile uint32_t *led =          (volatile uint32_t *)TK1_MMIO_TK1_LED;
static volatile uint32_t *trng_status  = (volatile uint32_t *)TK1_MMIO_TRNG_STATUS;
static volatile uint32_t *trng_entropy = (volatile uint32_t *)TK1_MMIO_TRNG_ENTROPY;

#define LED_BLACK  0
#define LED_RED    (1 << TK1_MMIO_TK1_LED_R_BIT)
#define LED_GREEN  (1 << TK1_MMIO_TK1_LED_G_BIT)
#define LED_BLUE   (1 << TK1_MMIO_TK1_LED_B_BIT)
// clang-format on

const uint8_t app_name0[4] = "tk1 ";
const uint8_t app_name1[4] = "rand";
const uint32_t app_version = 0x00000001;

// RSP_GET_RANDOM cmdlen - (responsecode + status)
#define RANDOM_PAYLOAD_MAXBYTES 128 - (1 + 1)

void get_random(uint8_t *buf, int bytes)
{
	int left = bytes;
	for (;;) {
		while ((*trng_status & (1 << TK1_MMIO_TRNG_STATUS_READY_BIT)) ==
		       0) {
		}
		uint32_t rnd = *trng_entropy;
		if (left > 4) {
			memcpy(buf, &rnd, 4);
			buf += 4;
			left -= 4;
			continue;
		}
		memcpy(buf, &rnd, left);
		break;
	}
}

int main(void)
{
	uint32_t stack;
	struct frame_header hdr; // Used in both directions
	uint8_t cmd[CMDLEN_MAXBYTES];
	uint8_t rsp[CMDLEN_MAXBYTES];
	uint8_t in;

	puts("Hello, I'm randomapp! &stack is on: ");
	putinthex((uint32_t)&stack);
	lf();

	for (;;) {
		// blocking; flashing while waiting for cmd
		in = readbyte_ledflash(LED_RED | LED_BLUE, 900000);
		puts("Read byte: ");
		puthex(in);
		putchar('\n');

		if (parseframe(in, &hdr) == -1) {
			puts("Couldn't parse header\n");
			continue;
		}

		memset(cmd, 0, CMDLEN_MAXBYTES);
		// Read app command, blocking
		read(cmd, hdr.len);

		// Is it for us?
		if (hdr.endpoint != DST_SW) {
			puts("Message not meant for app. endpoint was 0x");
			puthex(hdr.endpoint);
			lf();
			continue;
		}

		// Reset response buffer
		memset(rsp, 0, CMDLEN_MAXBYTES);

		// Min length is 1 byte so this should always be here
		switch (cmd[0]) {
		case APP_CMD_GET_NAMEVERSION:
			puts("APP_CMD_GET_NAMEVERSION\n");
			// only zeroes if unexpected cmdlen bytelen
			if (hdr.len == 1) {
				memcpy(rsp, app_name0, 4);
				memcpy(rsp + 4, app_name1, 4);
				memcpy(rsp + 8, &app_version, 4);
			}
			appreply(hdr, APP_RSP_GET_NAMEVERSION, rsp);
			break;

		case APP_CMD_GET_RANDOM:
			puts("APP_CMD_GET_RANDOM\n");
			if (hdr.len != 4) {
				puts("APP_CMD_GET_RANDOM bad cmd length\n");
				break;
			}
			// cmd[1] is number of bytes requested
			int bytes = cmd[1];
			if (bytes < 1 || bytes > RANDOM_PAYLOAD_MAXBYTES) {
				puts("Requested bytes outside range\n");
				rsp[0] = STATUS_BAD;
				appreply(hdr, APP_RSP_GET_RANDOM, rsp);
				break;
			}
			*led = LED_RED | LED_BLUE;
			rsp[0] = STATUS_OK;
			get_random(rsp + 1, bytes);
			appreply(hdr, APP_RSP_GET_RANDOM, rsp);
			break;

		default:
			puts("Received unknown command: ");
			puthex(cmd[0]);
			lf();
			appreply(hdr, APP_RSP_UNKNOWN_CMD, rsp);
		}
	}
}
