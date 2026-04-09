BINARY := gig

.PHONY: build run test fmt tidy

build:
	mkdir -p bin
	go build -o bin/$(BINARY) ./cmd/gig

run:
	go run ./cmd/gig

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy
