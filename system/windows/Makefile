
IMAGE=ghcr.io/tillitis/msi-builder:1

.PHONY: msi
msi:
	@[ -n "$$SEMVER" ] || { printf "Run like: make SEMVER=0.0.6 msi\n"; false; }
	podman run --arch=amd64 --rm -it --mount type=bind,source=$$(pwd),target=/build -w /build \
	  $(IMAGE) \
	  build-msi.sh "$$SEMVER" tkey-ssh-agent.wxs
	ls -l *.msi

.PHONY: exes
exes:
	make -C ../../ windows
	cp -af ../../tkey-ssh-agent{,-tray}.exe .

.PHONY: build-msi-builder
build-msi-builder:
	podman build -f Dockerfile -t msi-builder

.PHONY: clean
clean:
	rm -f tkey-ssh-agent{,-tray}.exe \
	  tkey-ssh-agent-*.{msi,wixpdb} tkey-ssh-agent.wixobj
