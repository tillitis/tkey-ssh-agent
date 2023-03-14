// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

#ifndef APP_PROTO_H
#define APP_PROTO_H

#include <lib.h>
#include <proto.h>

// clang-format off
enum appcmd {
	APP_CMD_GET_PUBKEY      = 0x01,
	APP_RSP_GET_PUBKEY      = 0x02,
	APP_CMD_SET_SIZE        = 0x03,
	APP_RSP_SET_SIZE        = 0x04,
	APP_CMD_SIGN_DATA       = 0x05,
	APP_RSP_SIGN_DATA       = 0x06,
	APP_CMD_GET_SIG         = 0x07,
	APP_RSP_GET_SIG         = 0x08,
	APP_CMD_GET_NAMEVERSION = 0x09,
	APP_RSP_GET_NAMEVERSION = 0x0a,

	APP_RSP_UNKNOWN_CMD     = 0xff,
};
// clang-format on

void appreply_nok(struct frame_header hdr);
void appreply(struct frame_header hdr, enum appcmd rspcode, void *buf);

#endif
