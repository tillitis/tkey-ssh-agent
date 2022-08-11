
        /* Read a byte from special MTA1_MKDF_MMIO_QEMU_CDI 0xfe000400 TODO */
        lui a0,0xfe000
        lb a0,0x400(a0)
