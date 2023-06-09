OBJCOPY ?= llvm-objcopy

P := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
LIBDIR ?= $(P)/../../tkey-libs

CC = clang

INCLUDE=$(LIBDIR)/include

# If you want libcommon's qemu_puts() et cetera to output something on our QEMU
# debug port, remove -DNODEBUG below
CFLAGS = -target riscv32-unknown-none-elf -march=rv32iczmmul -mabi=ilp32 -mcmodel=medany \
   -static -std=gnu99 -O2 -ffast-math -fno-common -fno-builtin-printf \
   -fno-builtin-putchar -nostdlib -mno-relax -flto -g \
   -Wall -Werror=implicit-function-declaration \
   -I $(INCLUDE) -I $(LIBDIR)  \
   -DNODEBUG

AS = clang
ASFLAGS = -target riscv32-unknown-none-elf -march=rv32iczmmul -mabi=ilp32 -mcmodel=medany -mno-relax

LDFLAGS=-T $(LIBDIR)/app.lds -L $(LIBDIR)/libcommon/ -lcommon -L $(LIBDIR)/libcrt0/ -lcrt0

.PHONY: all
all: timer/app.bin rng_stream/app.bin blink/app.bin nx/app.bin touch/app.bin

# Turn elf into bin for device
%.bin: %.elf
	$(OBJCOPY) --input-target=elf32-littleriscv --output-target=binary $^ $@
	chmod a-x $@

show-%-hash: %/app.bin
	cd $$(dirname $^) && sha512sum app.bin

# A simpler timer app
TIMEROBJS = timer/main.o timer/app_proto.o
timer/app.elf: $(TIMEROBJS)
	$(CC) $(CFLAGS) $(TIMEROBJS) $(LDFLAGS) -o $@
$(TIMEROBJS): $(INCLUDE)/tk1_mem.h timer/app_proto.h

# Streaming RNG random number app
RNG_STREAM_OBJS=rng_stream/main.o
rng_stream/app.elf: $(RNG_STREAM_OBJS)
	$(CC) $(CFLAGS) $(RNG_STREAM_OBJS) $(LDFLAGS) -o $@
$(RNG_STREAM_OBJS): $(INCLUDE)/tk1_mem.h

NX_OBJS=nx/main.o
nx/app.elf: $(NX_OBJS)
	$(CC) $(CFLAGS) $(NX_OBJS) $(LDFLAGS) -o $@
$(NX_OBJS): $(INCLUDE)/tk1_mem.h

# Blinkenlights assembler app
blink/app.elf: blink/blink.o
	ld.lld -o $@ blink/blink.o --oformat elf

TOUCHOBJS = touch/main.o
touch/app.elf: $(LIBS) $(TOUCHOBJS)
	$(CC) $(CFLAGS) $(TOUCHOBJS) $(LDFLAGS) -o $@
$(TOUCHOBJS): $(INCLUDE)/tk1_mem.h

.PHONY: clean
clean:
	rm -f timer/app.bin timer/app.elf $(TIMEROBJS)
	rm -f rng_stream/app.bin rng_stream/app.elf $(RNG_STREAM_OBJS)
	rm -f blink/app.bin blink/app.elf blink/blink.o
	rm -f nx/app.bin nx/app.elf nx/main.o
	rm -f touch/app.bin touch/app.elf touch/main.o

# Uses ../.clang-format
FMTFILES=timer/*.[ch] rng_stream/*.[ch] nx/main.c touch/*.[ch]

.PHONY: fmt
fmt:
	clang-format --dry-run --ferror-limit=0 $(FMTFILES)
	clang-format --verbose -i $(FMTFILES)
.PHONY: checkfmt
checkfmt:
	clang-format --dry-run --ferror-limit=0 --Werror $(FMTFILES)
