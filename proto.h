#ifndef PROTO_H
#define PROTO_H

enum endpoints {
	DST_HW_IFPGA,
	DST_HW_AFPGA,
	DST_FW,
	DST_SW
};

enum cmdlen {
	LEN_1,
	LEN_4,
	LEN_32,
	LEN_64
};

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

#endif

