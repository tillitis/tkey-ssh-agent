.PHONY: all
all: app runapp mta1-ssh-agent

.PHONY: app
app:
	$(MAKE) -C app

.PHONY: runapp
runapp:
	go build ./cmd/runapp

.PHONY: mta1-ssh-agent
mta1-ssh-agent:
	go build ./cmd/mta1-ssh-agent

.PHONY: lint
lint:
	docker run --rm -it --env GOFLAGS=-buildvcs=false -v $$(pwd):/src -w /src golangci/golangci-lint:v1.46-alpine golangci-lint run
