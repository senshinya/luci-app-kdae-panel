.PHONY: dev build build-go test vuln fmt web-install web-build clean

BINARY := bin/kdae-panel
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

dev:
	go run ./cmd/kdae-panel

build: web-build build-go

# build-go 跳过前端构建，供只改了 Go 代码时的快速循环使用；
# 嵌入的仍是 internal/webui/dist 里已有的那份产物。
build-go:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/kdae-panel

test:
	go test ./...
	go vet ./...
	cd web && npm run typecheck && npm test

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

fmt:
	gofmt -w cmd internal

web-install:
	cd web && npm ci

web-build:
	cd web && npm run build

clean:
	go clean
	rm -rf bin web/dist internal/webui/dist/assets
