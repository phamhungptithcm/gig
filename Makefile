BINARY := gig
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell git log -1 --format=%cI HEAD 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -buildid= -X gig/internal/buildinfo.Version=$(VERSION) -X gig/internal/buildinfo.Commit=$(COMMIT) -X gig/internal/buildinfo.Date=$(BUILD_DATE)

.PHONY: build run test fmt tidy release-assets package-manager-files

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

package-manager-files:
	./scripts/generate-package-manager-files.sh $(VERSION) dist
