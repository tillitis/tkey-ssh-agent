OBJCOPY ?= llvm-objcopy-14
COMMON=../../common

CC=clang -target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 -mcmodel=medany \
   -static -std=gnu99 -O2 -ffast-math -fno-common -fno-builtin-printf \
   -fno-builtin-putchar -static -nostdlib -mno-relax -Wall -flto -I $(COMMON)

CFLAGS=-DNODEBUG
LDFLAGS=-T app.lds

RM=/bin/rm



OBJS=crt0.o $(COMMON)/lib.o $(COMMON)/proto.o app_proto.o main.o monocypher/monocypher.o monocypher/monocypher-ed25519.o

.PHONY:all
all: app.bin

app.bin: app
	$(OBJCOPY) --input-target=elf32-littleriscv --output-target=binary app app.bin

app: $(OBJS) app.lds
	$(CC) $(CFLAGS) $(OBJS) $(LDFLAGS) -o $@

$(OBJS): $(COMMON)/mta1_mkdf_mem.h \
         $(COMMON)/types.h $(COMMON)/lib.h $(COMMON)/proto.h app_proto.h monocypher/monocypher.h monocypher/monocypher-ed25519.h

.PHONY:clean
clean:
	$(RM) -f app.bin app $(OBJS)

FMTFILES=main.c app_proto.h app_proto.c
.PHONY: fmt
fmt:
	make -C $(COMMON) fmt
	# Uses ../.clang-format
	clang-format --dry-run --ferror-limit=0 $(FMTFILES)
	clang-format --verbose -i $(FMTFILES)
