RM=/bin/rm

CC = clang-14

CFLAGS = -target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 \
   -static -std=gnu99 -O2 -ffast-math -fno-common -fno-builtin-printf \
   -fno-builtin-putchar -nostdlib -mno-relax -Wall -flto -I include #CFLAGS=-DNODEBUG

AS = clang-14
ASFLAGS = -target riscv32-unknown-none-elf -march=rv32imc -mabi=ilp32 -mno-relax

.PHONY: all
all: signerapp runapp tk1sign mkdf-ssh-agent

.PHONY: signerapp
signerapp: libcrt0/libcrt0.a libcommon/libcommon.a
	$(MAKE) -C apps/signerapp

# C runtime library
libcrt0/libcrt0.a: libcrt0/crt0.o
	 ar -q $@ libcrt0/crt0.o

# Common C functions
LIBOBJS=libcommon/lib.o libcommon/proto.o

libcommon/libcommon.a: $(LIBOBJS)
	 ar -q $@ libcommon/lib.o libcommon/proto.o

$(LIBOBJS): include/types.h include/mta1_mkdf_mem.h include/lib.h include/proto.h

# .PHONY to let go-build handle deps and rebuilds
.PHONY: runapp
runapp:
	go build ./cmd/runapp

# .PHONY to let go-build handle deps and rebuilds
.PHONY: tk1sign
tk1sign:
	go build ./cmd/tk1sign

# .PHONY to let go-build handle deps and rebuilds
.PHONY: mkdf-ssh-agent
mkdf-ssh-agent:
	cp -af apps/signerapp/app.bin cmd/mkdf-ssh-agent/app.bin
	go build ./cmd/mkdf-ssh-agent

.PHONY: clean
clean:
	rm -f runapp tk1sign mkdf-ssh-agent cmd/mkdf-ssh-agent/app.bin
	$(MAKE) -C apps/signerapp clean
	$(RM) -f libcommon/libcommon.a $(LIBOBJS)

.PHONY: update-mem-include
update-mem-include:
	cp -af ../tillitis-key1/hw/application_fpga/fw/mta1_mkdf_mem.h common/mta1_mkdf_mem.h

.PHONY: lint
lint: golangci-lint
	./golangci-lint run

# .PHONY to let go-build handle deps and rebuilds
.PHONY: golangci-lint
golangci-lint:
	go mod download github.com/golangci/golangci-lint
	go mod tidy
	go build github.com/golangci/golangci-lint/cmd/golangci-lint
