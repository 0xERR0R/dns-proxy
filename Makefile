#!/usr/bin/env bash

.PHONY: all clean build swagger test lint run help
.DEFAULT_GOAL := help

VERSION := $(shell git describe --always --tags)
BUILD_TIME=$(shell date '+%Y%m%d-%H%M%S')
DOCKER_IMAGE_NAME="ghcr.io/0xerr0r/dns-proxy"
BINARY_NAME=dns-proxy
BIN_OUT_DIR=bin

export PATH=$(shell go env GOPATH)/bin:$(shell echo $$PATH)

all: test lint build ## Build binary (with tests)

clean: ## cleans output directory
	$(shell rm -rf $(BIN_OUT_DIR)/*)

build:  ## Build binary
	go build -v -o $(BIN_OUT_DIR)/$(BINARY_NAME)$(BINARY_SUFFIX)

test:  ## run tests
	go test -v -coverprofile=coverage.txt -covermode=atomic -cover ./...

lint: build ## run golangcli-lint checks
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.42.1
	$(shell go env GOPATH)/bin/golangci-lint run

run: build ## Build and run binary
	./$(BIN_OUT_DIR)/$(BINARY_NAME)

fmt: ## gofmt and goimports all go files
	find . -name '*.go' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done

docker-build:  ## Build docker image
	docker build --network=host --tag ${DOCKER_IMAGE_NAME} .

help:  ## Shows help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
