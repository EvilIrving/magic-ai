.PHONY: build test tidy demo clean

VERSION := $(shell cat VERSION 2>/dev/null || echo 1.0.0)

build:
	go build -ldflags "-X magic/internal/core.Version=$(VERSION)" -o magic ./cmd/magic

test:
	go test ./...

tidy:
	go mod tidy

demo:
	./scripts/demo.sh

clean:
	rm -f magic
