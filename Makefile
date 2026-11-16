VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "N/A")
DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "N/A")

LDFLAGS := -X main.buildVersion=$(VERSION) \
           -X main.buildDate=$(DATE) \
           -X main.buildCommit=$(COMMIT)

.PHONY: build run test

build:
	go build -ldflags "$(LDFLAGS)" -o shortener ./cmd/shortener

run:
	go run -ldflags "$(LDFLAGS)" ./cmd/shortener

test:
	go test ./...

	statictests:
	go vet -vettool=statictest ./...
