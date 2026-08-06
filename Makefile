.PHONY: build fmt test test-race vet

build:
	go build ./...

fmt:
	go fmt ./...

test:
	go test ./...

test-race:
	go test ./... -race

vet:
	go vet ./...
