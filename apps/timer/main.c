// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

#include <lib.h>
#include <proto.h>
#include <tk1_mem.h>
#include <types.h>

#include "app_proto.h"

// clang-format off
volatile uint32_t *timer =           (volatile uint32_t *)TK1_MMIO_TIMER_TIMER;
volatile uint32_t *timer_prescaler = (volatile uint32_t *)TK1_MMIO_TIMER_PRESCALER;
volatile uint32_t *timer_status =    (volatile uint32_t *)TK1_MMIO_TIMER_STATUS;
volatile uint32_t *timer_ctrl =      (volatile uint32_t *)TK1_MMIO_TIMER_CTRL;
// clang-format on

int main(void)
{
	struct frame_header hdr; // Used in both directions
	uint8_t cmd[CMDLEN_MAXBYTES];
	uint8_t rsp[CMDLEN_MAXBYTES];

	for (;;) {
		uint8_t in = readbyte();

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

		switch (cmd[0]) {
		case APP_CMD_SET_TIMER:
			puts("APP_CMD_SET_TIMER\n");
			if (hdr.len != 32) {
				// Bad length
				puts("APP_CMD_SET_TIMER bad length\n");
				rsp[0] = STATUS_BAD;
				appreply(hdr, APP_RSP_SET_TIMER, rsp);
				break;
			}

			*timer = cmd[1] + (cmd[2] << 8) + (cmd[3] << 16) +
				 (cmd[4] << 24);

			rsp[0] = STATUS_OK;
			appreply(hdr, APP_RSP_SET_TIMER, rsp);
			break;

		case APP_CMD_SET_PRESCALER:
			puts("APP_CMD_SET_PRESCALER\n");
			if (hdr.len != 32) {
				// Bad length
				puts("APP_CMD_SET_TIMER bad length\n");
				rsp[0] = STATUS_BAD;
				appreply(hdr, APP_RSP_SET_PRESCALER, rsp);
				break;
			}

			*timer_prescaler = cmd[1] + (cmd[2] << 8) +
					   (cmd[3] << 16) + (cmd[4] << 24);

			rsp[0] = STATUS_OK;
			appreply(hdr, APP_RSP_SET_PRESCALER, rsp);
			break;

		case APP_CMD_START_TIMER:
			*timer_ctrl = 0x01;

			// Wait for the timer to expire
			for (;;) {
				if (*timer_status == 1) {
					puts("Timer expired\n");
					break;
				}
			}

			rsp[0] = STATUS_OK;
			appreply(hdr, APP_RSP_START_TIMER, rsp);
			break;

		default:
			puts("Received unknown command: ");
			puthex(cmd[0]);
			lf();
			appreply(hdr, APP_RSP_UNKNOWN_CMD, rsp);
			break;
		}
	}
}
