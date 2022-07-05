#ifndef LIB_H
#define LIB_H

#include "types.h"

int puts(const char *s);
void printf(const char *format, ...);
void hexdump(uint8_t *buf, int len);
void *memset(void *dest, int c, unsigned n);
void *memcpy(void *dest, const void *src, unsigned n);

#endif
