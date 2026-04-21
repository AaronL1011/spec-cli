BINARY := spec
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X github.com/nexl/spec-cli/cmd.Version=$(VERSION)"
GOFLAGS := -trimpath

.PHONY: build test lint clean install fmt vet

build:
	go build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY) .

install:
	go build $(GOFLAGS) $(LDFLAGS) -o $$(go env GOPATH)/bin/spec .

test:
	go test ./... -race -count=1

test-cover:
	go test ./... -race -count=1 -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	go vet ./...
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping"

fmt:
	gofmt -s -w .

vet:
	go vet ./...

clean:
	rm -rf bin/ coverage.out coverage.html

all: lint test build
