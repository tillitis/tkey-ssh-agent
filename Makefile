RM=/bin/rm

.PHONY: all
all: apps tkey-runapp tkey-sign runsign.sh tkey-ssh-agent runtimer runrandom

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
	install -Dm755 tkey-ssh-agent $(destbin)/tkey-ssh-agent
	strip $(destbin)/tkey-ssh-agent
	install -Dm644 system/tkey-ssh-agent.1 $(destman1)/tkey-ssh-agent.1
	gzip -n9f $(destman1)/tkey-ssh-agent.1
	install -Dm644 system/tkey-ssh-agent.service.tmpl $(destunit)/tkey-ssh-agent.service
	sed -i -e "s,##BINDIR##,$(PREFIX)/bin," $(destunit)/tkey-ssh-agent.service
	install -Dm644 system/60-tkey.rules $(destrules)/60-tkey.rules
.PHONY: uninstall
uninstall:
	rm -f \
	$(destbin)/tkey-ssh-agent \
	$(destunit)/tkey-ssh-agent.service \
	$(destrules)/60-tkey.rules \
	$(destman1)/tkey-ssh-agent.1.gz
.PHONY: reload-rules
reload-rules:
	udevadm control --reload
	udevadm trigger

.PHONY: apps
apps:
	$(MAKE) -C apps

# .PHONY to let go-build handle deps and rebuilds
.PHONY: tkey-runapp
tkey-runapp:
	go build ./cmd/tkey-runapp

# .PHONY to let go-build handle deps and rebuilds
.PHONY: tkey-sign
tkey-sign:
	go build -ldflags "-X main.signerAppNoTouch=$(TKEY_SIGNER_APP_NO_TOUCH)" ./cmd/tkey-sign

runsign.sh: apps/signer/runsign.sh
	cp -af $< $@

.PHONY: runtimer
runtimer:
	go build ./cmd/runtimer

# .PHONY to let go-build handle deps and rebuilds
.PHONY: runrandom
runrandom: apps
	cp -af apps/random/app.bin cmd/runrandom/app.bin
	go build ./cmd/runrandom

TKEY_SSH_AGENT_VERSION ?=
# .PHONY to let go-build handle deps and rebuilds
.PHONY: tkey-ssh-agent
tkey-ssh-agent: apps
	cp -af apps/signer/app.bin cmd/tkey-ssh-agent/app.bin
	CGO_ENABLED=0 go build -ldflags "-X main.version=$(TKEY_SSH_AGENT_VERSION) -X main.signerAppNoTouch=$(TKEY_SIGNER_APP_NO_TOUCH)" -trimpath ./cmd/tkey-ssh-agent

# TODO add windows manifest/program icon etc for both binaries
# $(MAKE) -C gotools go-winres

# .PHONY to let go-build handle deps and rebuilds
.PHONY: tkey-ssh-agent-tray.exe
tkey-ssh-agent-tray.exe:
	GOOS=windows CGO_ENABLED=0 go build -ldflags "-H windowsgui" -trimpath ./cmd/tkey-ssh-agent-tray

.PHONY: windows
windows:
	$(MAKE) GOOS=windows tkey-ssh-agent
	$(MAKE) tkey-ssh-agent-tray.exe

.PHONY: clean
clean:
	$(RM) -f tkey-runapp tkey-sign runsign.sh tkey-ssh-agent tkey-ssh-agent-tray.exe cmd/tkey-ssh-agent/app.bin runtimer runrandom cmd/runrandom/app.bin
	$(MAKE) -C apps clean

.PHONY: lint
lint:
	$(MAKE) -C gotools
	GOOS=linux   ./gotools/golangci-lint run
	GOOS=windows ./gotools/golangci-lint run
