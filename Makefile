.PHONY: all
all: app runapp mkdf-ssh-agent

.PHONY: app
app:
	$(MAKE) -C app

.PHONY: runapp
runapp:
	go build ./cmd/runapp

.PHONY: mkdf-ssh-agent
mkdf-ssh-agent: app
	cp -af app/app.bin cmd/mkdf-ssh-agent/app.bin
	go build ./cmd/mkdf-ssh-agent

.PHONE: clean
clean:
	rm -f runapp mkdf-ssh-agent
	$(MAKE) -C app clean

.PHONY: lint
lint:
	docker run --rm -it --env GOFLAGS=-buildvcs=false -v $$(pwd):/src -w /src golangci/golangci-lint:v1.46-alpine golangci-lint run
