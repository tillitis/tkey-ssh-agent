.PHONY: all
all: signerapp runapp mkdf-ssh-agent

.PHONY: signerapp
signerapp:
	$(MAKE) -C signerapp

.PHONY: runapp
runapp:
	go build ./cmd/runapp

.PHONY: mkdf-ssh-agent
mkdf-ssh-agent: signerapp
	cp -af signerapp/app.bin cmd/mkdf-ssh-agent/app.bin
	go build ./cmd/mkdf-ssh-agent

.PHONY: clean
clean:
	rm -f runapp mkdf-ssh-agent cmd/mkdf-ssh-agent/app.bin
	$(MAKE) -C signerapp clean

.PHONY: update-mem-include
update-mem-include:
	cp -af ../mta1-mkdf-qemu-priv/include/hw/riscv/mta1_mkdf_mem.h common/mta1_mkdf_mem.h

.PHONY: lint
lint: golangci-lint
	./golangci-lint run

golangci-lint: go.mod go.sum
	go build github.com/golangci/golangci-lint/cmd/golangci-lint
