VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
VERSION_NUM = $(shell echo $(VERSION) | sed 's/^v//')
LDFLAGS = -s -w -X main.version=$(VERSION_NUM)

.PHONY: build test lint clean install

build:
	go build -ldflags "$(LDFLAGS)" -o bin/ancc ./cmd/ancc

test:
	go test -race -count=1 ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

install: build
	cp bin/ancc $(GOPATH)/bin/ancc
