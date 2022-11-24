RM=/bin/rm

.PHONY: all
all: apps runapp tk-sign runsign.sh tk-ssh-agent runtimer runrandom

.PHONY: apps
apps:
	$(MAKE) -C apps

# .PHONY to let go-build handle deps and rebuilds
.PHONY: runapp
runapp:
	go build ./cmd/runapp

# .PHONY to let go-build handle deps and rebuilds
.PHONY: tk-sign
tk-sign:
	go build ./cmd/tk-sign

runsign.sh: apps/signerapp/runsign.sh
	cp -af $< $@

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
.PHONY: tk-ssh-agent
tk-ssh-agent:
	$(MAKE) -C apps signerapp/app.bin
	cp -af apps/signerapp/app.bin cmd/tk-ssh-agent/app.bin
	go build ./cmd/tk-ssh-agent

.PHONY: clean
clean:
	$(RM) -f runapp tk-sign runsign.sh tk-ssh-agent cmd/tk-ssh-agent/app.bin runtimer runrandom cmd/runrandom/app.bin
	$(MAKE) -C apps clean

.PHONY: lint
lint:
	$(MAKE) -C gotools
	./gotools/golangci-lint run
