.PHONY: build test vet-cross tidy demo clean

VERSION := $(shell cat VERSION 2>/dev/null || echo 1.0.0)

build:
	go build -ldflags "-X magic/internal/core.Version=$(VERSION)" -o magic ./cmd/magic

# vet-cross vets the two non-host targets. Path handling and syscall use differ
# per platform, and a mistake there compiles fine on the host, so both are vetted
# before the tests run.
vet-cross:
	GOOS=linux go vet ./...
	GOOS=windows go vet ./...

test: vet-cross
	go test ./...

tidy:
	go mod tidy

demo:
	./scripts/demo.sh

clean:
	rm -f magic
