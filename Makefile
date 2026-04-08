.PHONY: clean lint test build install

export GO111MODULE=on

LDFLAGS_PREFIX := github.com/traefik/assimilis/pkg/version

TAG_NAME := $(shell git tag -l --contains HEAD)
SHA := $(shell git rev-parse --short HEAD)
VERSION := $(if $(TAG_NAME),$(TAG_NAME),$(SHA))
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')

BIN_OUTPUT := $(if $(filter $(shell go env GOOS), windows), assimilis.exe, assimilis)

LDFLAGS := -s -w \
	-X "$(LDFLAGS_PREFIX).Version=$(VERSION)" \
	-X "$(LDFLAGS_PREFIX).Commit=$(SHA)" \
	-X "$(LDFLAGS_PREFIX).Date=$(BUILD_DATE)"

default: clean lint test build

test: clean
	go test -v -cover ./...

clean:
	rm -rf dist/ cover.out

lint:
	golangci-lint run

build: clean
	@echo Version: $(VERSION) $(BUILD_DATE)
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_OUTPUT) ./cmd/assimilis/

install:
	@echo Version: $(VERSION) $(BUILD_DATE)
	CGO_ENABLED=0 go install -trimpath -ldflags '$(LDFLAGS)' ./cmd/assimilis/
