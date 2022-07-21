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

.PHONY: clean
clean:
	rm -f runapp mkdf-ssh-agent cmd/mkdf-ssh-agent/app.bin
	$(MAKE) -C app clean

.PHONY: lint
lint: golangci-lint
	./golangci-lint run

golangci-lint: go.mod go.sum
	go build github.com/golangci/golangci-lint/cmd/golangci-lint
