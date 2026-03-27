BINARY     := mysqlpulse
MODULE     := github.com/ppiankov/mysqlpulse
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
VERSION_NUM = $(subst v,,$(VERSION))
REVISION   := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS    := -s -w -X main.version=$(VERSION_NUM) -X main.revision=$(REVISION)

.PHONY: build test lint clean docker

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/mysqlpulse

test:
	go test -race -count=1 ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

docker:
	docker build -t $(BINARY):$(VERSION_NUM) .
