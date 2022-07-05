CC=clang -target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 -mcmodel=medany \
   -static -std=gnu99 -O2 -ffast-math -fno-common -fno-builtin-printf \
   -fno-builtin-putchar -static -nostdlib -mno-relax -Wall 

LDFLAGS=-T app.ld

RM=/bin/rm

OBJS=crt0.o lib.o proto.o main.o

all: app.bin foo.bin

foo.bin: foo.S
	clang -c -target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 -mcmodel=medany -mno-relax foo.S
	ld.lld -o foo.bin foo.o --oformat binary

app.bin: app
	riscv32-elf-objcopy -O binary app app.bin

app: $(OBJS) types.h lib.h proto.h
	$(CC) $(CFLAGS) $(OBJS) $(LDFLAGS) -o $@

clean:
	$(RM) -f app.bin foo.bin app foo foo.o $(OBJS)
