.PHONY: all
all: signerapp runapp tk1sign mkdf-ssh-agent

.PHONY: signerapp
signerapp:
	$(MAKE) -C signerapp

runapp: ./cmd/runapp/*.go
	go build ./cmd/runapp

tk1sign: signerapp ./cmd/tk1sign/*.go
	go build ./cmd/tk1sign

mkdf-ssh-agent: signerapp ./cmd/mkdf-ssh-agent/*.go
	cp -af signerapp/app.bin cmd/mkdf-ssh-agent/app.bin
	go build ./cmd/mkdf-ssh-agent

.PHONY: clean
clean:
	rm -f runapp tk1sign mkdf-ssh-agent cmd/mkdf-ssh-agent/app.bin
	$(MAKE) -C signerapp clean

.PHONY: update-mem-include
update-mem-include:
	cp -af ../tillitis-key1/hw/application_fpga/fw/mta1_mkdf_mem.h common/mta1_mkdf_mem.h

.PHONY: lint
lint: golangci-lint
	./golangci-lint run

golangci-lint: go.mod go.sum
	go build github.com/golangci/golangci-lint/cmd/golangci-lint
