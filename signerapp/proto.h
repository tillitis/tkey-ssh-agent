#include "types.h"

#ifndef PROTO_H
#define PROTO_H

enum appcmd {
	APP_CMD_GET_PUBKEY = 0x01,
	APP_CMD_SET_SIZE = 0x02,
	APP_CMD_SIGN_DATA = 0x03,
	APP_CMD_GET_SIG = 0x04,
	APP_CMD_GET_NAMEVERSION = 0x05,

	APP_RSP_UNKNOWN_CMD = 0xff
};

enum endpoints {
	DST_HW_IFPGA = 0x00,
	DST_HW_AFPGA = 0x01,
	DST_FW = 0x02,
	DST_SW = 0x03
};

enum cmdlen {
	LEN_1,
	LEN_4,
	LEN_32,
	LEN_128
};

#define CMDLEN_MAXBYTES 128

enum status {
	STATUS_OK,
	STATUS_BAD
};

struct frame_header {
	uint8_t id;
	enum endpoints endpoint;
	enum cmdlen len;
};

uint8_t genhdr(uint8_t id, uint8_t endpoint, uint8_t status, enum cmdlen len);
int parseframe(uint8_t b, struct frame_header *hdr);
void writebyte(uint8_t b);
void write(uint8_t *buf, size_t nbytes);
uint8_t readbyte();
void read(uint8_t *buf, size_t nbytes);
void appreply(struct frame_header hdr, enum appcmd rspcode, void *buf);

#endif
