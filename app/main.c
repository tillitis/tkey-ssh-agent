#include "monocypher-ed25519.h"
#include "types.h"
#include "lib.h"
#include "proto.h"

volatile uint8_t *cdi = (volatile uint8_t *)0x90000400;
volatile uint32_t *name0 = (volatile uint32_t *)0x90000208;
volatile uint32_t *name1 = (volatile uint32_t *)0x9000020c;
volatile uint32_t *ver = (volatile uint32_t *)0x90000210;

#define MAX_SIGN_SIZE 32768

int main(void)
{
	uint8_t pubkey[32];
	struct frame_header hdr; // Used in both directions
	uint8_t cmd[64];
	uint8_t rsp[64];
	uint32_t message_size = 0;
	uint8_t message[MAX_SIGN_SIZE];
	uint8_t msg_idx; // Where we are currently loading the data to sign
	uint8_t signature[64];
	int left = 0; // Bytes left to read
	int nbytes = 0; // Bytes to write to memory
	uint8_t in;

	puts("Hello!\n");

	// Generate a public key from CDI
	crypto_ed25519_public_key(pubkey, (const uint8_t *)cdi);

	for (;;) {
		in = readbyte(); // blocks
		puts("Read byte: ");
		puthex(in);
		putchar('\n');
		//printf("Read byte 0x%x\n", in);

		if (parseframe(in, &hdr) == -1) {
			// Couldn't parse header
			puts("Couldn't parse header\n");
			continue;
		}

		//printf("id: %d, endpoint: %d, len: %d\n", hdr.id, hdr.endpoint, hdr.len);

		//Is it for us?
		if (hdr.endpoint != DST_SW) {
			puts("Message not meant for us, endpoint: ");
			puthex(hdr.endpoint);
			puthex(DST_SW);
			lf();

			// TODO eat the rest of the length?
			continue;
		}

		memset(cmd, 0, 64);
		// Read firmware command: Blocks!
		read(cmd, hdr.len);

		// Reset response buffer
		memset(rsp, 0, 64);

		// Min length is 1 byte so this should always be here
		//printf("command: %d\n", cmd[0]);
		switch (cmd[0]) {
		case APP_CMD_GET_PUBKEY:
			puts("APP_CMD_GET_PUBKEY\n");
			memcpy(rsp, pubkey, 32);
			appreply(hdr, APP_RSP_GET_PUBKEY, rsp);
			break;

		case APP_CMD_SET_SIZE:
			puts("APP_CMD_SET_SIZE\n");
			if (hdr.len != 32) {
				// Bad length
				puts("APP_CMD_SET_SIZE bad length\n");
				continue;
			}

			// cmd[1..4] contains the size.
			message_size = cmd[1] + (cmd[2] << 8) + (cmd[3] << 16) + (cmd[4] << 24);

			if (message_size > MAX_SIGN_SIZE) {
				puts("Message to big!\n");
				rsp[0] = STATUS_BAD;
				appreply(hdr, APP_CMD_SET_SIZE, rsp);
			}

			// Reset where we load the data
			left = message_size;
			msg_idx = 0;

			puts("Reply OK\n");
			rsp[0] = STATUS_OK;
			appreply(hdr, APP_CMD_SET_SIZE, rsp);
			break;

		case APP_CMD_SIGN_DATA:
			if (left > 63) {
				nbytes = 63;
			} else {
				nbytes = left;
			}

			memcpy(&message[msg_idx], cmd + 1, nbytes);
			msg_idx += nbytes;
			left -= nbytes;

			rsp[0] = STATUS_OK;
			appreply(hdr, APP_CMD_SET_SIZE, rsp);

			if (left == 0) {
				// All loaded, sign the message
				crypto_ed25519_sign(signature, (void *)cdi, pubkey, message, message_size);
			}

			break;

		case APP_CMD_GET_SIG:
			memcpy(rsp, signature, 64);
			appreply(hdr, APP_CMD_GET_SIG, rsp);
			break;
		}
	}
}