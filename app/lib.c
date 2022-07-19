#include "types.h"

volatile uint8_t *debugtx = (volatile uint8_t *)0x90001000;

int putchar(uint8_t ch)
{
	*debugtx = ch;

	return ch;
}

void lf()
{
	putchar('\n');
}

char hexnibble(uint8_t ch)
{
	switch (ch) {
	case 0x0:
		return '0';
	case 0x1:
		return '1';
	case 0x2:
		return '2';
	case 0x3:
		return '3';
	case 0x4:
		return '4';
	case 0x5:
		return '5';
	case 0x6:
		return '6';
	case 0x7:
		return '7';
	case 0x8:
		return '8';
	case 0x9:
		return '9';
	case 0xa:
		return 'a';
	case 0xb:
		return 'b';
	case 0xc:
		return 'c';
	case 0xd:
		return 'd';
	case 0xe:
		return 'e';
	case 0xf:
		return 'f';
	}

	return '0';
}

void puthex(uint8_t ch)
{
	putchar(hexnibble(ch >> 4 & 0x0f));
	putchar(hexnibble(ch & 0x0f));
}

void puts(const char *s)
{
	while (*s)
		putchar(*s++);
}

void hexdump(uint8_t *buf, int len)
{
	uint8_t *row;
	uint8_t *byte;
	uint8_t *max;

	row = buf;
	max = &buf[len];

	for (byte = 0; byte != max; row = byte) {
		// Offset
		// printf("%07x ", row - buf);

		for (byte = row; byte != max && byte != (row + 16); byte++) {
			puthex(*byte);
		}

		(void)putchar('\n');
	}
}

void *memset(void *dest, int c, unsigned n)
{
	uint8_t *s = dest;

	for (; n; n--, s++)
		*s = c;

	return dest;
}

__attribute__((used)) void *memcpy(void *dest, const void *src, unsigned n)
{
	uint8_t *src_byte = (uint8_t *)src;
	uint8_t *dest_byte = (uint8_t *)dest;

	for (int i = 0; i < n; i++) {
		dest_byte[i] = src_byte[i];
	}

	return dest;
}
