BINARY_NAME := otel-collector-mcp
DOCKER_IMAGE := ghcr.io/hrexed/otel-collector-mcp
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build test lint docker-build clean

build:
	CGO_ENABLED=0 GOOS=linux go build -o bin/$(BINARY_NAME) ./cmd/server

test:
	go test ./... -v -race -coverprofile=coverage.out

lint:
	golangci-lint run ./...

docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

clean:
	rm -rf bin/ coverage.out
