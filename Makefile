BINARY  = tb
REPO    = devjoaoGustavo/tb
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || ([ -f VERSION ] && echo "v$$(cat VERSION)") || echo "dev")
LDFLAGS = -ldflags "-X main.version=$(VERSION) -s -w"

.PHONY: build install uninstall test dist release

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/tb

install:
	go install $(LDFLAGS) ./cmd/tb

uninstall:
	rm -f $(shell go env GOPATH)/bin/$(BINARY)

test:
	go test ./...

dist:
	mkdir -p dist
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)_darwin_amd64   ./cmd/tb
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)_darwin_arm64   ./cmd/tb
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)_linux_amd64    ./cmd/tb
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)_linux_arm64    ./cmd/tb
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)_windows_amd64.exe ./cmd/tb

release: dist
	gh release create "$(VERSION)" dist/* \
		--title "$(VERSION)" \
		--generate-notes
