.DEFAULT_GOAL := build

COMMIT_HASH = `git rev-parse --short HEAD 2>/dev/null`
BUILD_DATE = `date +%FT%T%z`

GO = go
BINARY_DIR=bin

BUILD_DEPS:= github.com/alecthomas/gometalinter
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

.PHONY: vendor test build

help:
	@echo "vendor      - Install govendor and sync vendored dependencies"
	@echo "checkstyle  - executes bunch of checkstyle validators"
	@echo "fmt         - formats the project"
	@echo "test        - executes unit tests"
	@echo "build       - builds Linux binary"
	@echo "docker      - Builds docker image"
	@echo "clean       - Cleans build-related files from working directory"


vendor: ## Install govendor and sync Hugo's vendored dependencies
	go get github.com/kardianos/govendor
	govendor sync

get-build-deps: vendor # prepare stuff required for the build
	$(GO) get $(BUILD_DEPS)
	gometalinter --install

# executes unit-tests
test: vendor
	govendor test +local

# executes bunch of checkstyle validators
checkstyle: get-build-deps
	gometalinter --vendor ./... --fast --disable=gas --disable=errcheck --disable=gotype #--deadline 5m

# formats the project
fmt:
	gofmt -l -w -s ${GOFILES_NOVENDOR}

# Builds the project for linux-based OS
build: vendor
	$(GO) build -o ${BINARY_DIR}/rpLandingInfo ./landinginfo.go

# Builds docker image
docker: build
	docker build -t reportportal/landing-info .

# clean-ups stuff
clean:
	if [ -d ${BINARY_DIR} ] ; then rm -r ${BINARY_DIR} ; fi

release: vendor test
	script/release.sh $v
