// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

#ifndef APP_PROTO_H
#define APP_PROTO_H

#include <lib.h>
#include <proto.h>

// clang-format off
enum appcmd {
	APP_CMD_SET_TIMER = 0x01,
	APP_RSP_SET_TIMER = 0x02,
	APP_CMD_SET_PRESCALER = 0x03,
	APP_RSP_SET_PRESCALER = 0x04,
	APP_CMD_START_TIMER = 0x05,
	APP_RSP_START_TIMER = 0x06,

	APP_CMD_GET_NAMEVERSION = 0x09,
	APP_RSP_GET_NAMEVERSION = 0x0a,

	APP_RSP_UNKNOWN_CMD     = 0xff,
};
// clang-format on

void appreply(struct frame_header hdr, enum appcmd rspcode, void *buf);

#endif
