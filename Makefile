.PHONY: all build test race cover vet fmt fmtcheck tidy lint examples integration

all: fmtcheck vet test

build:
	go build ./...

test:
	go test ./...

race:
	go test -race -count=1 ./...

# Live system tests. Requires IPREGISTRY_API_KEY; consumes credits.
integration:
	go test -tags integration -run Integration ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

vet:
	go vet ./...

fmt:
	gofmt -w .

fmtcheck:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed on:"; gofmt -l .; exit 1)

tidy:
	go mod tidy

# Requires: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
lint:
	golangci-lint run ./...

examples:
	go build ./examples/...
