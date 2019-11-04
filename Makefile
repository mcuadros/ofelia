# Package configuration
PROJECT = ofelia
COMMANDS = ofelia

# Environment
BASE_PATH := $(shell pwd)
BUILD_PATH := $(BASE_PATH)/bin
SHA1 := $(shell git log --format='%H' -n 1 | cut -c1-10)
BUILD := $(shell date +"%m-%d-%Y_%H_%M_%S")
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

# Packages content
PKG_OS = darwin linux
PKG_ARCH = amd64
PKG_CONTENT =
PKG_TAG = latest

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GHRELEASE = github-release
LDFLAGS = -ldflags "-X main.version=$(BRANCH) -X main.build=$(BUILD)" 

# Coverage
COVERAGE_REPORT = coverage.txt
COVERAGE_MODE = atomic

ifneq ($(origin TRAVIS_TAG), undefined)
	BRANCH := $(TRAVIS_TAG)
endif

# Rules
all: clean packages

.PHONY: test
test: 
	@go test -v ./...

.PHONY: test-coverage
test-coverage: 
	@echo "mode: $(COVERAGE_MODE)" > $(COVERAGE_REPORT);
	@go test -v ./... $${p} -coverprofile=tmp_$(COVERAGE_REPORT) -covermode=$(COVERAGE_MODE); 
	cat tmp_$(COVERAGE_REPORT) | grep -v "mode: $(COVERAGE_MODE)" >> $(COVERAGE_REPORT); 
	rm tmp_$(COVERAGE_REPORT); 

build:
	go build -o $(BUILD_PATH)/$(PROJECT) $${cmd}.go;

packages:
	@for os in $(PKG_OS); do \
		for arch in $(PKG_ARCH); do \
			cd $(BASE_PATH); \
			FINAL_PATH=$(BUILD_PATH)/$(PROJECT)_$${os}_$${arch}; \
			mkdir -p $${FINAL_PATH}; \
			for cmd in $(COMMANDS); do \
				BINARY=$(BUILD_PATH)/$(PROJECT)_$${os}_$${arch}/$${cmd};\
				GOOS=$${os} GOARCH=$${arch} $(GOCMD) build -ldflags "-X main.version=$(BRANCH) -X main.build=$(BUILD)" -o $${BINARY} $${cmd}.go;\
				du -h $${BINARY};\
			done; \
			for content in $(PKG_CONTENT); do \
				cp -rfv $${content} $(BUILD_PATH)/$(PROJECT)_$${os}_$${arch}/; \
			done; \
			TAR_PATH=$(BUILD_PATH)/$(PROJECT)_$(BRANCH)_$${os}_$${arch}.tar.gz;\
			cd  $(BUILD_PATH) && tar -cvzf $${TAR_PATH} $(PROJECT)_$${os}_$${arch}/; \
			du -h $${TAR_PATH};\
		done; \
	done;

clean:
	@rm -rf $(BUILD_PATH)
