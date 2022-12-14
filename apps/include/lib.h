// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

#ifndef LIB_H
#define LIB_H

#include <types.h>

int qemu_putchar(uint8_t ch);
void qemu_lf();
void qemu_putinthex(const uint32_t n);
void qemu_puts(const char *s);
void qemu_puthex(uint8_t ch);
void qemu_hexdump(uint8_t *buf, int len);
void *memset(void *dest, int c, unsigned n);
void *memcpy(void *dest, const void *src, unsigned n);
void *wordcpy(void *dest, const void *src, unsigned n);
int blake2s(void *out, unsigned long outlen, const void *key,
	    unsigned long keylen, const void *in, unsigned long inlen,
	    blake2s_ctx *ctx);

#endif
