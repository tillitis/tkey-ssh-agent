#include "app_proto.h"

// Send app reply with frame header, response code, and LEN_X-1 bytes from buf
void appreply(struct frame_header hdr, enum appcmd rspcode, void *buf)
{
	size_t nbytes;
	enum cmdlen len;

	switch (rspcode) {
	case APP_CMD_GET_PUBKEY:
		len = LEN_128;
		nbytes = 128;
		break;

	case APP_CMD_SET_SIZE:
		len = LEN_4;
		nbytes = 4;
		break;

	case APP_CMD_SIGN_DATA:
		len = LEN_4;
		nbytes = 4;
		break;

	case APP_CMD_GET_SIG:
		len = LEN_128;
		nbytes = 128;
		break;

	case APP_CMD_GET_NAMEVERSION:
		len = LEN_32;
		nbytes = 32;
		break;

	case APP_RSP_UNKNOWN_CMD:
		len = LEN_1;
		nbytes = 1;
		break;

	default:
		puts("appreply(): Unknown response code: ");
		puthex(rspcode);
		lf();

		return;
	}

	// Frame Protocol Header
	writebyte(genhdr(hdr.id, hdr.endpoint, 0x0, len));

	// app protocol header is 1 byte response code
	writebyte(rspcode);
	nbytes--;

	write(buf, nbytes);
}
