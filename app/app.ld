OUTPUT_ARCH( "riscv" )
ENTRY(_start)

MEMORY
{
	ROM (rwx)   : ORIGIN = 0x1000, LENGTH = 0x20000 /* 128 KB */
	RAM (rwx)   : ORIGIN = 0x80001000, LENGTH = 0x20000 /* 128 KB */
}

SECTIONS
{
	.text.init :
	{
		*(.text.init)
	} >RAM

	/* .htif : */
	/* { */
	/* 	. = ALIGN(0x1000); */
	/* 	*(.htif) */
	/* } >ROM */

	.text :
	{
		. = ALIGN(4);
		*(.text)           /* .text sections (code) */
		*(.text*)          /* .text* sections (code) */
		*(.rodata)         /* .rodata sections (constants, strings, etc.) */
		*(.rodata*)        /* .rodata* sections (constants, strings, etc.) */
		*(.srodata)        /* .rodata sections (constants, strings, etc.) */
		*(.srodata*)       /* .rodata* sections (constants, strings, etc.) */
		. = ALIGN(4);
		_etext = .;
		_sidata = _etext;
	} >RAM

	.data : AT (_etext)
	{
		. = ALIGN(4);
		_sdata = .;
		. = ALIGN(4);
		*(.data)           /* .data sections */
		*(.data*)          /* .data* sections */
		*(.sdata)           /* .sdata sections */
		*(.sdata*)          /* .sdata* sections */
		. = ALIGN(4);
		_edata = .;
	} >RAM

	/* Uninitialized data section */
	.bss :
	{
		. = ALIGN(4);
		_sbss = .;
		*(.bss)
		*(.bss*)
		*(.sbss)
		*(.sbss*)
		*(COMMON)

		. = ALIGN(4);
		_ebss = .;
	} >RAM
	/* Init stack to _ebss + size */
}
