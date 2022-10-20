OBJCOPY ?= llvm-objcopy-14

CC = clang-14

INCLUDE=include

# If you want libcommon's puts() et cetera to output something on our
# QEMU debug port, remove -DNODEBUG below
CFLAGS = -target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 -mcmodel=medany \
   -static -std=gnu99 -O2 -ffast-math -fno-common -fno-builtin-printf \
   -fno-builtin-putchar -nostdlib -mno-relax -Wall -flto -I $(INCLUDE) -DNODEBUG

AS = clang-14
ASFLAGS = -target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 -mno-relax

LDFLAGS=-T app.lds -L libcommon/ -lcommon -L libcrt0/ -lcrt0

RM=/bin/rm


.PHONY: all
all: signerapp/app.bin timer/timer.bin random/random.bin

# C runtime library
libcrt0/libcrt0.a: libcrt0/crt0.o
	 ar -q $@ libcrt0/crt0.o

# Common C functions
LIBOBJS=libcommon/lib.o libcommon/proto.o

libcommon/libcommon.a: $(LIBOBJS)
	 ar -q $@ libcommon/lib.o libcommon/proto.o

$(LIBOBJS): include/types.h include/tk1_mem.h include/lib.h include/proto.h

# Turn elf into bin for device
%.bin: %.elf
	$(OBJCOPY) --input-target=elf32-littleriscv --output-target=binary $^ $@

# Simple ed25519 signer app
SIGNEROBJS=signerapp/app_proto.o \
    signerapp/main.o \
    signerapp/monocypher/monocypher.o \
    signerapp/monocypher/monocypher-ed25519.o

signerapp/app.elf: libcrt0/libcrt0.a libcommon/libcommon.a $(SIGNEROBJS) app.lds
	$(CC) $(CFLAGS) $(SIGNEROBJS) $(LDFLAGS) -o $@

$(SIGNEROBJS): $(INCLUDE)/tk1_mem.h signerapp/app_proto.h \
    signerapp/monocypher/monocypher.h \
    signerapp/monocypher/monocypher-ed25519.h

# A simpler timer app
TIMEROBJS = timer/main.o timer/app_proto.o

timer/timer.elf: libcrt0/libcrt0.a libcommon/libcommon.a $(TIMEROBJS) app.lds
	$(CC) $(CFLAGS) $(TIMEROBJS) $(LDFLAGS) -o $@

$(TIMEROBJS): $(INCLUDE)/tk1_mem.h timer/app_proto.h

# Random number generator app
RANDOMOBJS=random/main.o random/app_proto.o

random/random.bin: random/random.elf
	$(OBJCOPY) --input-target=elf32-littleriscv --output-target=binary $^ $@

random/random.elf: libcrt0/libcrt0.a libcommon/libcommon.a $(RANDOMOBJS) app.lds
	$(CC) $(CFLAGS) $(RANDOMOBJS) $(LDFLAGS) -o $@

$(RANDOMOBJS): $(INCLUDE)/tk1_mem.h random/app_proto.h


.PHONY: clean
clean:
	$(RM) -f signerapp/app.bin signerapp/app.elf $(SIGNEROBJS)
	$(RM) -f timer/timer.bin timer/timer.elf $(TIMEROBJS)
	$(RM) -f random/random.bin random/random.elf $(RANDOMOBJS)
	$(RM) -f libcommon/libcommon.a $(LIBOBJS) libcrt0/libcrt0.a libcrt0/crt0.o

FMTFILES=include/*.h libcommon/*.c signerapp/*.[ch] timer/*.[ch] random/*.[ch]
.PHONY: fmt
fmt:
	# Uses ../.clang-format
	clang-format --dry-run --ferror-limit=0 $(FMTFILES)
	clang-format --verbose -i $(FMTFILES)

.PHONY: update-mem-include
update-mem-include:
	cp -af ../../tillitis-key1/hw/application_fpga/fw/tk1_mem.h include/tk1_mem.h
