BINARY := gig
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell git log -1 --format=%cI HEAD 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -buildid= -X gig/internal/buildinfo.Version=$(VERSION) -X gig/internal/buildinfo.Commit=$(COMMIT) -X gig/internal/buildinfo.Date=$(BUILD_DATE)

.PHONY: build run test fmt tidy release-assets npm-package npm-release-prepare npm-release-bootstrap npm-release-trust npm-release-verify

build:
	mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/gig

run:
	go run ./cmd/gig

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

release-assets:
	./scripts/build-release-assets.sh $(VERSION) dist

npm-package:
	npm pack --dry-run

npm-release-prepare:
	./scripts/npm-release.sh prepare $(TAG)

npm-release-bootstrap:
	./scripts/npm-release.sh bootstrap $(TAG)

npm-release-trust:
	./scripts/npm-release.sh trust

npm-release-verify:
	./scripts/npm-release.sh verify
