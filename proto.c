#include "types.h"
#include "lib.h"
#include "proto.h"

volatile uint8_t *can_rx = (volatile uint8_t *)0x90000214;
volatile uint8_t *rx = (volatile uint8_t *)0x90000215;
volatile uint8_t *can_tx = (volatile uint8_t *)0x90000216;
volatile uint8_t *tx = (volatile uint8_t *)0x90000217;

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
	hdr->endpoint = (b & 0x10) >> 3;

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
	case LEN_64:
		hdr->len = 64;
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
	puts("Sending: \n");
	hexdump(buf, nbytes);
	for (int i = 0; i < nbytes; i ++) {
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

void read(uint8_t *buf, size_t nbytes)
{
	for (int n = 0; n < nbytes; n ++) {
		buf[n] = readbyte();
	}
}

void appreply(struct frame_header hdr, enum appcmd rspcode, void *buf)
{
	size_t nbytes;
	enum cmdlen len;

	switch (rspcode) {
	case APP_RSP_GET_PUBKEY:
		len = LEN_32;
		nbytes = 32;
		break;

	case APP_CMD_SET_SIZE:
		len = LEN_1;
		nbytes = 1;
		break;

	case APP_CMD_SIGN_DATA:
		len = LEN_1;
		nbytes = 1;
		break;

	case APP_CMD_GET_SIG:
		len = LEN_64;
		nbytes = 64;
		break;

	default:
		puts("appreply(): Unknown response code: ");
		puthex(rspcode);
		lf();

		return;
	}

	// Frame Protocol Header
	//printf("Sending: (0x%x)\n  id:%d, endpoint:%d, len: %d\n  rspcode: %d\n", genhdr(hdr.id, hdr.endpoint, 0x0, len), hdr.id, hdr.endpoint, len, rspcode);
	writebyte(genhdr(hdr.id, hdr.endpoint, 0x0, len));

	// App protocol header
	//writebyte(rspcode);
	//nbytes --;

	write(buf, nbytes);
}
