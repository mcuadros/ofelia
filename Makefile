# Package configuration
PROJECT = ofelia
COMMANDS = ofelia
DEPENDENCIES = github.com/aktau/github-release

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

# Rules
all: clean upload

dependencies:
	$(GOGET) -t ./...
	for i in $(DEPENDENCIES); do $(GOGET) $$i; done

test: dependencies
	$(GOTEST) ./...

packages: dependencies
	for os in $(PKG_OS); do \
		for arch in $(PKG_ARCH); do \
			cd $(BASE_PATH); \
			mkdir -p $(BUILD_PATH)/$(PROJECT)_$${os}_$${arch}; \
			for cmd in $(COMMANDS); do \
				GOOS=$${os} GOARCH=$${arch} $(GOCMD) build -ldflags "-X main.version $(SHA1) -X main.build \"$(BUILD)\"" -o $(BUILD_PATH)/$(PROJECT)_$${os}_$${arch}/$${cmd} $${cmd}.go ; \
			done; \
			for content in $(PKG_CONTENT); do \
				cp -rf $${content} $(BUILD_PATH)/$(PROJECT)_$${os}_$${arch}/; \
			done; \
			cd  $(BUILD_PATH) && tar -cvzf $(BUILD_PATH)/$(PROJECT)_$${os}_$${arch}.tar.gz $(PROJECT)_$${os}_$${arch}/; \
		done; \
	done;

upload: packages
	cd $(BASE_PATH); \
	$(GHRELEASE) delete --tag $(PKG_TAG); \
	$(GHRELEASE) release --tag $(PKG_TAG) --name "$(PKG_TAG) ($(SHA1))"; \
	for os in $(PKG_OS); do \
		for arch in $(PKG_ARCH); do \
			$(GHRELEASE) upload \
		    --tag $(PKG_TAG) \
				--name "$(PROJECT)_$${os}_$${arch}.tar.gz" \
				--file $(BUILD_PATH)/$(PROJECT)_$${os}_$${arch}.tar.gz; \
		done; \
	done;

clean:
	rm -rf $(BUILD_PATH)
	$(GOCLEAN) .
