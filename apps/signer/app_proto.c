// Copyright (C) 2022, 2023 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

#include "app_proto.h"

// Send app reply with frame header, response code, and LEN_X-1 bytes from buf
void appreply(struct frame_header hdr, enum appcmd rspcode, void *buf)
{
	size_t nbytes;
	enum cmdlen len;

	switch (rspcode) {
	case APP_RSP_GET_PUBKEY:
		len = LEN_512;
		nbytes = 512;
		break;

	case APP_RSP_SET_SIZE:
		len = LEN_4;
		nbytes = 4;
		break;

	case APP_RSP_SIGN_DATA:
		len = LEN_4;
		nbytes = 4;
		break;

	case APP_RSP_GET_SIG:
		len = LEN_512;
		nbytes = 512;
		break;

	case APP_RSP_GET_NAMEVERSION:
		len = LEN_32;
		nbytes = 32;
		break;

	case APP_RSP_UNKNOWN_CMD:
		len = LEN_1;
		nbytes = 1;
		break;

	default:
		qemu_puts("appreply(): Unknown response code: ");
		qemu_puthex(rspcode);
		qemu_lf();

		return;
	}

	// Frame Protocol Header
	writebyte(genhdr(hdr.id, hdr.endpoint, 0x0, len));

	// app protocol header is 1 byte response code
	writebyte(rspcode);
	nbytes--;

	write(buf, nbytes);
}
