// Copyright (C) 2022 - Tillitis AB
// SPDX-License-Identifier: GPL-2.0-only
#include <tk1_mem.h>
#include <lib.h>

// clang-format off
volatile uint32_t *cpu_mon_ctrl  = (volatile uint32_t *)TK1_MMIO_TK1_CPU_MON_CTRL;
volatile uint32_t *cpu_mon_first = (volatile uint32_t *)TK1_MMIO_TK1_CPU_MON_FIRST;
volatile uint32_t *cpu_mon_last  = (volatile uint32_t *)TK1_MMIO_TK1_CPU_MON_LAST;
static volatile uint32_t *led    = (volatile uint32_t *)TK1_MMIO_TK1_LED;

#define LED_GREEN (1 << TK1_MMIO_TK1_LED_G_BIT)
// clang-format on

int main(void)
{
	int led_on = 0;

blink:
	// Blink green
	for (int i = 0; i < 10; i++) {
		*led = led_on ? LED_GREEN : 0;
		for (volatile int j = 0; j < 350000; j++) {
		}
		led_on = !led_on;
	}

	*cpu_mon_first = (uint32_t)&&blink;
	*cpu_mon_last = (uint32_t)(&&blink + 1024);
	*cpu_mon_ctrl = 1;

	// Should not blink anymore
	goto blink;
}
