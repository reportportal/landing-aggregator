.DEFAULT_GOAL := build

COMMIT_HASH = `git rev-parse --short HEAD 2>/dev/null`
BUILD_DATE = `date +%FT%T%z`

GO = go
BINARY_DIR=bin

BUILD_DEPS:= github.com/alecthomas/gometalinter
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

.PHONY: test build

help:
	@echo "checkstyle  - executes bunch of checkstyle validators"
	@echo "fmt         - formats the project"
	@echo "test        - executes unit tests"
	@echo "build       - builds Linux binary"
	@echo "docker      - Builds docker image"
	@echo "clean       - Cleans build-related files from working directory"


get-build-deps: # prepare stuff required for the build
	$(GO) get $(BUILD_DEPS)
	gometalinter --install

# executes unit-tests
test:
	$(GO) test ./...

# executes bunch of checkstyle validators
checkstyle:
	gometalinter --vendor ./... --fast --disable=gas --disable=errcheck --disable=gotype --deadline 5m

# formats the project
fmt:
	gofmt -l -w -s ${GOFILES_NOVENDOR}

# Builds the project for linux-based OS
build:
	$(GO) build -o ${BINARY_DIR}/landinginfo ./landinginfo.go

# Builds the project for linux-based OS
build_docker:
	CGO_ENABLED=0 GOOS=linux $(GO) build -o ${BINARY_DIR}/landinginfo ./landinginfo.go

# Builds docker image
docker: build_docker
	docker build -t reportportal/landing-aggregator .

# clean-ups stuff
clean:
	if [ -d ${BINARY_DIR} ] ; then rm -r ${BINARY_DIR} ; fi

release: test
	scripts/release.sh $v
