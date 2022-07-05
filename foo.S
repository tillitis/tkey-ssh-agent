
        /* Read a byte from special MMIO 0x90000400 */
        lui a0,0x90000
        lb a0,1024(a0) # 0x90000400
