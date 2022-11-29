RM=/bin/rm

.PHONY: all
all: apps runapp tk-sign runsign.sh tk-ssh-agent runtimer runrandom

DESTDIR=/
PREFIX=/usr/local
SYSTEMDDIR=/etc/systemd
UDEVDIR=/etc/udev
destbin=$(DESTDIR)/$(PREFIX)/bin
destman1=$(DESTDIR)/$(PREFIX)/share/man/man1
destunit=$(DESTDIR)/$(SYSTEMDDIR)/user
destrules=$(DESTDIR)/$(UDEVDIR)/rules.d
.PHONY: install
install:
	install -Dm755 tk-ssh-agent $(destbin)/tk-ssh-agent
	strip $(destbin)/tk-ssh-agent
	install -Dm644 system/tk-ssh-agent.1 $(destman1)/tk-ssh-agent.1
	gzip -n9f $(destman1)/tk-ssh-agent.1
	install -Dm644 system/tk-ssh-agent.service.tmpl $(destunit)/tk-ssh-agent.service
	sed -i -e "s,##BINDIR##,$(PREFIX)/bin," $(destunit)/tk-ssh-agent.service
	install -Dm644 system/60-tillitis-key.rules $(destrules)/60-tillitis-key.rules
	install -Dm644 system/90-tk-ssh-agent.rules $(destrules)/90-tk-ssh-agent.rules
.PHONY: uninstall
uninstall:
	rm -f \
	$(destbin)/tk-ssh-agent \
	$(destunit)/tk-ssh-agent.service \
	$(destrules)/60-tillitis-key.rules \
	$(destrules)/90-tk-ssh-agent.rules \
	$(destman1)/tk-ssh-agent.1.gz
.PHONY: reload-rules
reload-rules:
	udevadm control --reload
	udevadm trigger

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
runrandom: apps
	cp -af apps/random/random.bin cmd/runrandom/app.bin
	go build ./cmd/runrandom

# .PHONY to let go-build handle deps and rebuilds
.PHONY: tk-ssh-agent
tk-ssh-agent: apps
	cp -af apps/signerapp/app.bin cmd/tk-ssh-agent/app.bin
	CGO_ENABLED=0 go build -trimpath ./cmd/tk-ssh-agent

.PHONY: clean
clean:
	$(RM) -f runapp tk-sign runsign.sh tk-ssh-agent cmd/tk-ssh-agent/app.bin runtimer runrandom cmd/runrandom/app.bin
	$(MAKE) -C apps clean

.PHONY: lint
lint:
	$(MAKE) -C gotools
	./gotools/golangci-lint run
