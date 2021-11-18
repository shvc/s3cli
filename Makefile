APP?=s3cli
BUILDDATE=$(shell date +'%Y-%m-%dT%H:%M:%SZ')
VERSION=2.2.9
LONGVER=${VERSION}@${BUILDDATE}@$(shell git rev-parse --short HEAD)

LDFLAGS=-ldflags "-X main.version=${LONGVER}"

.DEFAULT_GOAL:=default

## pkg: build and package the app
.PHONY: pkg
pkg:
	@echo "Building Linux amd64 ${APP}-${VERSION}"
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS}
	zip -m ${APP}-${VERSION}-linux-amd64.zip ${APP}

	@echo "Building Linux arm64 ${APP}-${VERSION}"
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS}
	zip -m ${APP}-${VERSION}-linux-arm64.zip ${APP}
	
	@echo "Building Macos amd64 ${APP}-${VERSION}"
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS}
	zip -m ${APP}-${VERSION}-macos-amd64.zip ${APP}

	@echo "Building Macos arm64 ${APP}-${VERSION}"
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS}
	zip -m ${APP}-${VERSION}-macos-arm64.zip ${APP}
	
	@echo "Building Windows amd64 ${APP}-${VERSION}"
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS}
	zip -m ${APP}-${VERSION}-win-amd64.zip ${APP}.exe

## test: runs go test with default values
.PHONY: test
test:
	go test ./...

## vet: runs go vet
.PHONY: vet
vet:
	go vet ./...

## default: build the app
.PHONY: default
default:
	@echo "Building ${APP}-${VERSION}"
	go build ${LDFLAGS}

## clean: cleans the build results
.PHONY: clean
clean:
	rm -rf *zip ${APP}

## help: prints this help message
.PHONY: help
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

