// Copyright (C) 2022, 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

#include <lib.h>
#include <proto.h>
#include <tk1_mem.h>

// clang-format off
static volatile uint32_t *can_rx = (volatile uint32_t *)TK1_MMIO_UART_RX_STATUS;
static volatile uint32_t *rx =     (volatile uint32_t *)TK1_MMIO_UART_RX_DATA;
static volatile uint32_t *can_tx = (volatile uint32_t *)TK1_MMIO_UART_TX_STATUS;
static volatile uint32_t *tx =     (volatile uint32_t *)TK1_MMIO_UART_TX_DATA;
static volatile uint32_t *led =    (volatile uint32_t *)TK1_MMIO_TK1_LED;
// clang-format on

uint8_t genhdr(uint8_t id, uint8_t endpoint, uint8_t status, enum cmdlen len)
{
	return (id << 5) | (endpoint << 3) | (status << 2) | len;
}

int parseframe(uint8_t b, struct frame_header *hdr)
{
	if ((b & 0x80) != 0) {
		// Bad version
		return -1;
	}

	if ((b & 0x4) != 0) {
		// Must be 0
		return -1;
	}

	hdr->id = (b & 0x60) >> 5;
	hdr->endpoint = (b & 0x18) >> 3;

	// Length
	switch (b & 0x3) {
	case LEN_1:
		hdr->len = 1;
		break;
	case LEN_4:
		hdr->len = 4;
		break;
	case LEN_32:
		hdr->len = 32;
		break;
	case LEN_512:
		hdr->len = 512;
		break;
	default:
		// Unknown length
		return -1;
	}

	return 0;
}

void writebyte(uint8_t b)
{
	for (;;) {
		if (*can_tx) {
			*tx = b;
			return;
		}
	}
}

void write(uint8_t *buf, size_t nbytes)
{
	qemu_puts("Sending: \n");
	qemu_hexdump(buf, nbytes);
	for (int i = 0; i < nbytes; i++) {
		writebyte(buf[i]);
	}
}

uint8_t readbyte()
{
	for (;;) {
		if (*can_rx) {
			return *rx;
		}
	}
}

uint8_t readbyte_ledflash(int ledvalue, int loopcount)
{
	int led_on = 0;
	for (;;) {
		*led = led_on ? ledvalue : 0;
		for (int i = 0; i < loopcount; i++) {
			if (*can_rx) {
				return *rx;
			}
		}
		led_on = !led_on;
	}
}

void read(uint8_t *buf, size_t nbytes)
{
	for (int n = 0; n < nbytes; n++) {
		buf[n] = readbyte();
	}
}
