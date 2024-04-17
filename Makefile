# Check for OS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	shasum = shasum -a 512
	BUILD_CGO_ENABLED ?= 1
else
	shasum = sha512sum
	BUILD_CGO_ENABLED ?= 0
endif

.PHONY: all
all: tkey-ssh-agent

.PHONY: windows
windows: tkey-ssh-agent.exe tkey-ssh-agent-tray.exe

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

.PHONY: podman
podman:
	podman run --rm --mount type=bind,source=$(CURDIR),target=/src -w /src -it ghcr.io/tillitis/tkey-builder:4 make -j

.PHONY: check-signer-hash
check-signer-hash:
	cd cmd/tkey-ssh-agent && $(shasum) -c signer.bin.sha512

TKEY_SSH_AGENT_VERSION ?=
# .PHONY to let go-build handle deps and rebuilds
.PHONY: tkey-ssh-agent
tkey-ssh-agent: check-signer-hash
	CGO_ENABLED=$(BUILD_CGO_ENABLED) go build -ldflags "-X main.version=$(TKEY_SSH_AGENT_VERSION) -X main.signerAppNoTouch=$(TKEY_SIGNER_APP_NO_TOUCH)" -trimpath -buildvcs=false ./cmd/tkey-ssh-agent

.PHONY: tkey-ssh-agent.exe
tkey-ssh-agent.exe:
	$(MAKE) -C gotools go-winres
	cd ./cmd/tkey-ssh-agent && ../../gotools/go-winres make --arch amd64
	$(MAKE) GOOS=windows GOARCH=amd64 tkey-ssh-agent

# .PHONY to let go-build handle deps and rebuilds
.PHONY: tkey-ssh-agent-tray.exe
tkey-ssh-agent-tray.exe:
	$(MAKE) -C gotools go-winres
	cd ./cmd/tkey-ssh-agent-tray && ../../gotools/go-winres make --arch amd64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-H windowsgui -X main.version=$(TKEY_SSH_AGENT_VERSION)" -trimpath -buildvcs=false ./cmd/tkey-ssh-agent-tray

.PHONY: clean
clean:
	rm -f \
	tkey-ssh-agent \
	tkey-ssh-agent.exe cmd/tkey-ssh-agent/rsrc_windows_amd64.syso \
	tkey-ssh-agent-tray.exe cmd/tkey-ssh-agent-tray/rsrc_windows_amd64.syso

.PHONY: lint
lint:
	GOOS=linux   golangci-lint run
	GOOS=windows golangci-lint run
