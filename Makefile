.PHONY: build-linux

LDFLAGS=-ldflags "-w -s"

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ./build/go-admin .
