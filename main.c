#include "types.h"
#include "lib.h"
#include "proto.h"

volatile uint8_t *cdi = (volatile uint8_t *)0x90000400;
volatile uint32_t *name0 = (volatile uint32_t *)0x90000208;
volatile uint32_t *name1 = (volatile uint32_t *)0x9000020c;
volatile uint32_t *ver = (volatile uint32_t *)0x90000210;

int main(void)
{
	uint8_t buf[64];

	uint8_t cdi0 = *cdi;

	memset(buf, 0x0, 64);
	memcpy(buf, "hello from app!", 15);

	writebyte(genhdr(2, DST_FW, 0x0, LEN_32));
	write(buf, 64);
}
