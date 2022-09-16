OBJCOPY ?= llvm-objcopy-14

CC=clang -target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 -mcmodel=medany \
   -static -std=gnu99 -O2 -ffast-math -fno-common -fno-builtin-printf \
   -fno-builtin-putchar -static -nostdlib -mno-relax -Wall -flto

CFLAGS=-DNODEBUG
LDFLAGS=-T app.lds

RM=/bin/rm

OBJS=crt0.o ../common/lib.o ../common/proto.o app_proto.o main.o monocypher.o monocypher-ed25519.o

.PHONY:all
all: app.bin

app.bin: app
	$(OBJCOPY) --input-target=elf32-littleriscv --output-target=binary app app.bin

app: $(OBJS) app.lds
	$(CC) $(CFLAGS) $(OBJS) $(LDFLAGS) -o $@

$(OBJS): ../common/mta1_mkdf_mem.h \
         ../common/types.h ../common/lib.h ../common/proto.h app_proto.h monocypher.h monocypher-ed25519.h

.PHONY:clean
clean:
	$(RM) -f app.bin app $(OBJS)

.PHONY: fmt
fmt:
	make -C ../common fmt
	clang-format -i main.c app_proto.h app_proto.c
