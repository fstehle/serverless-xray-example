BUILD_DIR            := bin
SERVERLESS           := node_modules/.bin/serverless
GOLANGCI_LINT        := tools/golangci-lint
BINARIES             := $(shell find . -name 'main.go' | grep -v -e node_modules -e vendor |awk -F/ '{print "bin/" $$2}')
DEPENDENCIES         := $(shell find . -type f -name '*.go')
LINT_TARGETS         := $(shell go list -f '{{.Dir}}' ./... | sed -e"s|${CURDIR}/\(.*\)\$$|\1/...|g" | grep -v ^node_modules/ )

all: lint build

node_modules: package.json
	npm install
	touch node_modules

$(SERVERLESS): node_modules

$(BUILD_DIR)/%: %/*.go $(DEPENDENCIES)
	env GOOS=linux go build -ldflags="-s -w" -o $@	./$(notdir $@)

.PHONY: build
build: $(BINARIES)

.PHONY: deploy
deploy: $(SERVERLESS) $(BINARIES)
	$(SERVERLESS) deploy

$(GOLANGCI_LINT):
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b tools v1.15.0

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run $(LINT_TARGETS)

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)