CC=clang -target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 -mcmodel=medany \
   -static -std=gnu99 -O2 -ffast-math -fno-common -fno-builtin-printf \
   -fno-builtin-putchar -static -nostdlib -mno-relax -Wall -flto

#CFLAGS=-DNODEBUG
LDFLAGS=-T app.ld

RM=/bin/rm

OBJS=crt0.o lib.o proto.o main.o monocypher.o monocypher-ed25519.o

.PHONY:all
all: app.bin

app.bin: app
	llvm-objcopy --input-target=elf32-littleriscv --output-target=binary app app.bin

app: $(OBJS)
	$(CC) $(CFLAGS) $(OBJS) $(LDFLAGS) -o $@

$(OBJS): types.h lib.h proto.h monocypher.h monocypher-ed25519.h

.PHONY:clean
clean:
	$(RM) -f app.bin app $(OBJS)

.PHONY: fmt
fmt:
	clang-format -i main.c lib.h lib.c proto.h proto.c types.h
