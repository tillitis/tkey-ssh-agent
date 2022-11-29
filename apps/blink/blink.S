// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only

	.section ".text.init"
	.global _start

_start:
        li a0, 0xff000024 # LED MMIO
        li a1, 0x1
loop:
        sw a1, 0(a0)
        addi a1, a1, 1

        li a2, 0
delay:
        li a3, 100000
        addi a2, a2, 1
        blt a2, a3, delay
        j loop
