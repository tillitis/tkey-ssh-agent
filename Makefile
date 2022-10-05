RM=/bin/rm

.PHONY: all
all: apps runapp tk1sign mkdf-ssh-agent runtimer runrandom

.PHONY: apps
apps:
	$(MAKE) -C apps

# .PHONY to let go-build handle deps and rebuilds
.PHONY: runapp
runapp:
	go build ./cmd/runapp

# .PHONY to let go-build handle deps and rebuilds
.PHONY: tk1sign
tk1sign:
	go build ./cmd/tk1sign

.PHONY: runtimer
runtimer:
	go build ./cmd/runtimer

# .PHONY to let go-build handle deps and rebuilds
.PHONY: runrandom
runrandom:
	$(MAKE) -C apps random/random.bin
	cp -af apps/random/random.bin cmd/runrandom/app.bin
	go build ./cmd/runrandom

# .PHONY to let go-build handle deps and rebuilds
.PHONY: mkdf-ssh-agent
mkdf-ssh-agent:
	$(MAKE) -C apps signerapp/app.bin
	cp -af apps/signerapp/app.bin cmd/mkdf-ssh-agent/app.bin
	go build ./cmd/mkdf-ssh-agent

.PHONY: clean
clean:
	rm -f runapp tk1sign mkdf-ssh-agent cmd/mkdf-ssh-agent/app.bin runtimer runrandom cmd/runrnadom/app.bin
	$(MAKE) -C apps clean

.PHONY: lint
lint: golangci-lint
	./golangci-lint run

# .PHONY to let go-build handle deps and rebuilds
.PHONY: golangci-lint
golangci-lint:
	go mod download github.com/golangci/golangci-lint
	go mod tidy
	go build github.com/golangci/golangci-lint/cmd/golangci-lint
