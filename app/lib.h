#ifndef LIB_H
#define LIB_H

#include "types.h"

int putchar(uint8_t ch);
void lf();
void puts(const char *s);
void puthex(uint8_t ch);
void hexdump(uint8_t *buf, int len);
void *memset(void *dest, int c, unsigned n);
void *memcpy(void *dest, const void *src, unsigned n);

#endif
