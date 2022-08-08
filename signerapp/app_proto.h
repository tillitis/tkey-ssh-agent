#ifndef APP_PROTO_H
#define APP_PROTO_H

#include "../common/lib.h"
#include "../common/proto.h"

enum appcmd {
	APP_CMD_GET_PUBKEY = 0x01,
	APP_CMD_SET_SIZE = 0x02,
	APP_CMD_SIGN_DATA = 0x03,
	APP_CMD_GET_SIG = 0x04,
	APP_CMD_GET_NAMEVERSION = 0x05,

	APP_RSP_UNKNOWN_CMD = 0xff
};

void appreply(struct frame_header hdr, enum appcmd rspcode, void *buf);

#endif
