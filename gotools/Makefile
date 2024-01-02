# .PHONY to let go-build handle deps and rebuilds
.PHONY: go-winres
go-winres:
	go mod download github.com/tc-hib/go-winres
	go mod tidy
	go build github.com/tc-hib/go-winres
