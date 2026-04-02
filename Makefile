# Nome do binário
BINARY_NAME=llm-pack

# Metadados de Build
VERSION=1.0.0
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION}"

.PHONY: all build clean install help

all: build-all

## build-linux: Compila para Linux (amd64)
build-linux:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-linux-amd64 main.go

## build-mac: Compila para macOS (Intel e Apple Silicon)
build-mac:
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-darwin-arm64 main.go

## build-windows: Compila para Windows (amd64)
build-windows:
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o bin/${BINARY_NAME}-windows-amd64.exe main.go

## build-all: Compila para todas as plataformas de uma vez
build-all: clean build-linux build-mac build-windows

## clean: Remove a pasta de binários e arquivos gerados
clean:
	rm -rf bin/
	rm -f llm_bundle_*.md project_map.md

## install: Compila e instala no seu /usr/local/bin (apenas Unix)
install:
	go build ${LDFLAGS} -o ${BINARY_NAME} main.go
	sudo mv ${BINARY_NAME} /usr/local/bin/
	@echo "🚀 llm-pack instalado com sucesso! Tente rodar 'llm-pack --help'"

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'