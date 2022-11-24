
# .PHONY to let go-build handle deps and rebuilds
.PHONY: golangci-lint
golangci-lint:
	go mod download github.com/golangci/golangci-lint
	go mod tidy
	go build github.com/golangci/golangci-lint/cmd/golangci-lint
