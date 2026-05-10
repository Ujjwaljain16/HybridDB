.PHONY: all build test test-integration bench lint fmt clean setup

all: lint build test

build:
	go build ./...

test:
	go test -v -short ./...

test-integration:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./benchmarks/...

lint:
	golangci-lint run

fmt:
	go fmt ./...

clean:
	rm -rf *.hdb
	go clean

setup:
	./scripts/setup.sh
