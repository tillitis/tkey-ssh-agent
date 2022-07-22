CC=clang -target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 -mcmodel=medany \
   -static -std=gnu99 -O2 -ffast-math -fno-common -fno-builtin-printf \
   -fno-builtin-putchar -static -nostdlib -mno-relax -Wall -flto

#CFLAGS=-DNODEBUG
LDFLAGS=-T app.ld

RM=/bin/rm

OBJS=crt0.o lib.o proto.o main.o monocypher.o monocypher-ed25519.o

.PHONY:all
all: app.bin foo.bin

foo.bin: foo.S
	clang -c -target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 -mcmodel=medany -mno-relax foo.S
	ld.lld -o foo.bin foo.o --oformat binary

app.bin: app
	llvm-objcopy --input-target=elf32-littleriscv --output-target=binary app app.bin

app: $(OBJS) types.h lib.h proto.h
	$(CC) $(CFLAGS) $(OBJS) $(LDFLAGS) -o $@

.PHONY:clean
clean:
	$(RM) -f app.bin foo.bin app foo foo.o $(OBJS)

.PHONY: fmt
fmt:
	clang-format -i main.c lib.h lib.c proto.h proto.c types.h
