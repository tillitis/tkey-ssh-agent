.PHONY: all
all: runapp mta1-ssh-agent

.PHONY: runapp
runapp:
	go build ./cmd/runapp

.PHONY: mta1-ssh-agent
mta1-ssh-agent:
	go build ./cmd/mta1-ssh-agent

.PHONY: lint
lint:
	docker run --rm -it -v $$(pwd):/src -w /src golangci/golangci-lint:v1.46-alpine golangci-lint run
