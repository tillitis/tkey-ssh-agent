OBJCOPY ?= llvm-objcopy

CC = clang

INCLUDE=include

# If you want libcommon's qemu_puts() et cetera to output something on our QEMU
# debug port, remove -DNODEBUG below
CFLAGS = -target riscv32-unknown-none-elf -march=rv32iczmmul -mabi=ilp32 -mcmodel=medany \
   -static -std=gnu99 -O2 -ffast-math -fno-common -fno-builtin-printf \
   -fno-builtin-putchar -nostdlib -mno-relax -flto \
   -Wall -Werror=implicit-function-declaration \
   -I $(INCLUDE) -I . \
   -DNODEBUG
ifneq ($(TKEY_SIGNER_APP_NO_TOUCH),)
CFLAGS := $(CFLAGS) -DTKEY_SIGNER_APP_NO_TOUCH
endif

AS = clang
ASFLAGS = -target riscv32-unknown-none-elf -march=rv32iczmmul -mabi=ilp32 -mcmodel=medany -mno-relax

LDFLAGS=-T app.lds -L libcommon/ -lcommon -L libcrt0/ -lcrt0

RM=/bin/rm


.PHONY: all
all: signer/app.bin timer/app.bin random/app.bin rng_stream/app.bin blink/app.bin nx/app.bin

# C runtime library
libcrt0/libcrt0.a: libcrt0/crt0.o
	llvm-ar -qc $@ libcrt0/crt0.o

# Common C functions
LIBOBJS=libcommon/lib.o libcommon/proto.o
libcommon/libcommon.a: $(LIBOBJS)
	llvm-ar -qc $@ libcommon/lib.o libcommon/proto.o
$(LIBOBJS): include/types.h include/tk1_mem.h include/lib.h include/proto.h

# Monocypher
MONOOBJS=monocypher/monocypher.o monocypher/monocypher-ed25519.o
monocypher/libmonocypher.a: $(MONOOBJS)
	llvm-ar -qc $@ $(MONOOBJS)
$MONOOBJS: monocypher/monocypher-ed25519.h monocypher/monocypher.h

LIBS=libcrt0/libcrt0.a libcommon/libcommon.a

# Turn elf into bin for device
%.bin: %.elf
	$(OBJCOPY) --input-target=elf32-littleriscv --output-target=binary $^ $@
	chmod a-x $@

show-%-hash: %/app.bin
	cd $$(dirname $^) && sha512sum app.bin

check-signer-hash: signer/app.bin
	cd signer && sha512sum -c app.bin.sha512

# Simple ed25519 signer app
SIGNEROBJS=signer/main.o signer/app_proto.o
signer/app.elf: $(LIBS) monocypher/libmonocypher.a $(SIGNEROBJS) app.lds
	$(CC) $(CFLAGS) $(SIGNEROBJS) $(LDFLAGS) -L monocypher -lmonocypher -I monocypher -o $@
$(SIGNEROBJS): $(INCLUDE)/tk1_mem.h signer/app_proto.h

# A simpler timer app
TIMEROBJS = timer/main.o timer/app_proto.o
timer/app.elf: $(LIBS) $(TIMEROBJS) app.lds
	$(CC) $(CFLAGS) $(TIMEROBJS) $(LDFLAGS) -o $@
$(TIMEROBJS): $(INCLUDE)/tk1_mem.h timer/app_proto.h

# Random number generator app
RANDOMOBJS=random/main.o random/app_proto.o
random/app.elf: $(LIBS) $(RANDOMOBJS) app.lds
	$(CC) $(CFLAGS) $(RANDOMOBJS) $(LDFLAGS) -o $@
$(RANDOMOBJS): $(INCLUDE)/tk1_mem.h random/app_proto.h

# Streaming RNG random number app
RNG_STREAM_OBJS=rng_stream/main.o
rng_stream/app.elf: $(LIBS) $(RNG_STREAM_OBJS) app.lds
	$(CC) $(CFLAGS) $(RNG_STREAM_OBJS) $(LDFLAGS) -o $@
$(RNG_STREAM_OBJS): $(INCLUDE)/tk1_mem.h

NX_OBJS=nx/main.o
nx/app.elf: $(LIBS) $(NX_OBJS) app.lds
	$(CC) $(CFLAGS) $(NX_OBJS) $(LDFLAGS) -o $@
$(NX_OBJS): $(INCLUDE)/tk1_mem.h

# Blinkenlights assembler app
blink/app.elf: blink/blink.o
	ld.lld -o $@ blink/blink.o --oformat elf

.PHONY: clean
clean:
	$(RM) -f signer/app.bin signer/app.elf $(SIGNEROBJS)
	$(RM) -f timer/app.bin timer/app.elf $(TIMEROBJS)
	$(RM) -f random/app.bin random/app.elf $(RANDOMOBJS)
	$(RM) -f rng_stream/app.bin rng_stream/app.elf $(RNG_STREAM_OBJS)
	$(RM) -f $(LIBS) $(LIBOBJS) libcrt0/crt0.o
	$(RM) -f monocypher/libmonocypher.a $(MONOOBJS)
	$(RM) -f blink/app.bin blink/app.elf blink/blink.o
	$(RM) -f nx/app.bin nx/app.elf nx/main.o

# Uses ../.clang-format
FMTFILES=include/*.h libcommon/*.c signer/*.[ch] timer/*.[ch] random/*.[ch] rng_stream/*.[ch] nx/main.c
.PHONY: fmt
fmt:
	clang-format --dry-run --ferror-limit=0 $(FMTFILES)
	clang-format --verbose -i $(FMTFILES)
.PHONY: checkfmt
checkfmt:
	clang-format --dry-run --ferror-limit=0 --Werror $(FMTFILES)

.PHONY: update-mem-include
update-mem-include:
	cp -af ../../tillitis-key1/hw/application_fpga/fw/tk1_mem.h include/tk1_mem.h
