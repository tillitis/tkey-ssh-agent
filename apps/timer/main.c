// Copyright (C) 2022, 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

#include <lib.h>
#include <proto.h>
#include <tk1_mem.h>
#include <types.h>

#include "app_proto.h"

// clang-format off
volatile uint32_t *led =             (volatile uint32_t *)TK1_MMIO_TK1_LED;
volatile uint32_t *timer =           (volatile uint32_t *)TK1_MMIO_TIMER_TIMER;
volatile uint32_t *timer_prescaler = (volatile uint32_t *)TK1_MMIO_TIMER_PRESCALER;
volatile uint32_t *timer_status =    (volatile uint32_t *)TK1_MMIO_TIMER_STATUS;
volatile uint32_t *timer_ctrl =      (volatile uint32_t *)TK1_MMIO_TIMER_CTRL;

#define LED_BLACK  0
#define LED_RED    (1 << TK1_MMIO_TK1_LED_R_BIT)
#define LED_GREEN  (1 << TK1_MMIO_TK1_LED_G_BIT)
#define LED_BLUE   (1 << TK1_MMIO_TK1_LED_B_BIT)
// clang-format on

int main(void)
{
	struct frame_header hdr; // Used in both directions
	uint8_t cmd[CMDLEN_MAXBYTES];
	uint8_t rsp[CMDLEN_MAXBYTES];

	*led = LED_RED | LED_GREEN;
	for (;;) {
		uint8_t in = readbyte();

		if (parseframe(in, &hdr) == -1) {
			qemu_puts("Couldn't parse header\n");
			continue;
		}

		memset(cmd, 0, CMDLEN_MAXBYTES);
		// Read app command, blocking
		read(cmd, hdr.len);

		// Is it for us?
		if (hdr.endpoint != DST_SW) {
			qemu_puts("Message not meant for app. endpoint was 0x");
			qemu_puthex(hdr.endpoint);
			qemu_lf();
			continue;
		}

		// Reset response buffer
		memset(rsp, 0, CMDLEN_MAXBYTES);

		switch (cmd[0]) {
		case APP_CMD_SET_TIMER:
			qemu_puts("APP_CMD_SET_TIMER\n");
			if (hdr.len != 32) {
				// Bad length
				qemu_puts("APP_CMD_SET_TIMER bad length\n");
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
			qemu_puts("APP_CMD_SET_PRESCALER\n");
			if (hdr.len != 32) {
				// Bad length
				qemu_puts("APP_CMD_SET_TIMER bad length\n");
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
			*timer_ctrl = (1 << TK1_MMIO_TIMER_CTRL_START_BIT);

			// Wait for the timer to expire
			while (*timer_status &
			       (1 << TK1_MMIO_TIMER_STATUS_RUNNING_BIT)) {
			}

			rsp[0] = STATUS_OK;
			appreply(hdr, APP_RSP_START_TIMER, rsp);
			break;

		default:
			qemu_puts("Received unknown command: ");
			qemu_puthex(cmd[0]);
			qemu_lf();
			appreply(hdr, APP_RSP_UNKNOWN_CMD, rsp);
			break;
		}
	}
}
