OBJCOPY ?= llvm-objcopy-14

CC=clang

INCLUDE=../../include
CFLAGS=-target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 -mcmodel=medany \
   -static -std=gnu99 -O2 -ffast-math -fno-common -fno-builtin-printf \
   -fno-builtin-putchar -static -nostdlib -mno-relax -Wall -flto -I $(INCLUDE)

LDFLAGS=-T ../app.lds -L ../../libcommon/ -lcommon -L ../../libcrt0/ -lcrt0

RM=/bin/rm

OBJS=app_proto.o main.o monocypher/monocypher.o monocypher/monocypher-ed25519.o

.PHONY:all
all: app.bin

app.bin: app
	$(OBJCOPY) --input-target=elf32-littleriscv --output-target=binary app app.bin

app: $(OBJS) ../app.lds
	$(CC) $(CFLAGS) $(OBJS) $(LDFLAGS) -o $@

$(OBJS): $(INCLUDE)/mta1_mkdf_mem.h \
	$(INCLUDE)/types.h \
	$(INCLUDE)/lib.h \
	$(INCLUDE)/proto.h \
	monocypher/monocypher.h \
	monocypher/monocypher-ed25519.h \
	app_proto.h

.PHONY:clean
clean:
	$(RM) -f app.bin app $(OBJS)

FMTFILES=main.c app_proto.h app_proto.c
.PHONY: fmt
fmt:
	make -C ../common fmt
	# Uses ../.clang-format
	clang-format --dry-run --ferror-limit=0 $(FMTFILES)
	clang-format --verbose -i $(FMTFILES)
