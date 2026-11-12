VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint fmt snapshot clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/howto ./cmd/howto

test:
	go test ./...

lint:
	go vet ./...
	golangci-lint run

fmt:
	gofmt -w .

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin dist
