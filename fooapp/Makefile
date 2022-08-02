foo.bin: foo.S
	clang -c -target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 -mcmodel=medany -mno-relax foo.S
	ld.lld -o foo.bin foo.o --oformat binary

.PHONY:clean
clean:
	$(RM) -f foo.bin foo foo.o
