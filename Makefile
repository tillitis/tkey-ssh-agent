.PHONY: all
all: signerapp runapp tk1sign mkdf-ssh-agent

.PHONY: signerapp
signerapp:
	$(MAKE) -C signerapp

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

# .PHONY to let go-build handle deps and rebuilds
.PHONY: golangci-lint
golangci-lint:
	go mod download github.com/golangci/golangci-lint
	go mod tidy
	go build github.com/golangci/golangci-lint/cmd/golangci-lint
