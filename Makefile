# Package configuration
PROJECT = ofelia
COMMANDS = ofelia
DEPENDENCIES = golang.org/x/tools/cmd/cover
PACKAGES = github.com/mcuadros/ofelia/core

# Environment
BASE_PATH := $(shell pwd)
BUILD_PATH := $(BASE_PATH)/build
SHA1 := $(shell git log --format='%H' -n 1 | cut -c1-10)
BUILD := $(shell date)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

# Packages content
PKG_OS = darwin linux
PKG_ARCH = amd64
PKG_CONTENT =
PKG_TAG = latest

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOGET = $(GOCMD) get -v
GOTEST = $(GOCMD) test -v
GHRELEASE = github-release

# Coverage
COVERAGE_REPORT = coverage.txt
COVERAGE_MODE = atomic

ifneq ($(origin TRAVIS_TAG), undefined)
	BRANCH := $(TRAVIS_TAG)
endif

# Rules
all: clean upload

dependencies:
	$(GOGET) -t ./...
	for i in $(DEPENDENCIES); do $(GOGET) $$i; done

test: dependencies
	for p in $(PACKAGES); do \
		$(GOTEST) $${p}; \
	done;

test-coverage: dependencies
	for p in $(PACKAGES); do \
		$(GOTEST) $${p} -coverprofile=$(COVERAGE_REPORT) -covermode=$(COVERAGE_MODE); \
	done;

packages: dependencies
	for os in $(PKG_OS); do \
		for arch in $(PKG_ARCH); do \
			cd $(BASE_PATH); \
			mkdir -p $(BUILD_PATH)/$(PROJECT)_$${os}_$${arch}; \
			for cmd in $(COMMANDS); do \
				GOOS=$${os} GOARCH=$${arch} $(GOCMD) build -ldflags "-X main.version $(BRANCH) -X main.build \"$(BUILD)\"" -o $(BUILD_PATH)/$(PROJECT)_$${os}_$${arch}/$${cmd} $${cmd}.go ; \
			done; \
			for content in $(PKG_CONTENT); do \
				cp -rf $${content} $(BUILD_PATH)/$(PROJECT)_$${os}_$${arch}/; \
			done; \
			cd  $(BUILD_PATH) && tar -cvzf $(BUILD_PATH)/$(PROJECT)_$(BRANCH)_$${os}_$${arch}.tar.gz $(PROJECT)_$${os}_$${arch}/; \
		done; \
	done;

clean:
	rm -rf $(BUILD_PATH)
	$(GOCLEAN) .
