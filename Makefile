# Make does not offer a recursive wildcard function, so here's one:
rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

SHELL := /bin/bash
NAME := jx-semanticcheck
BINARY_NAME := jx-semanticcheck
BUILD_TARGET = build
MAIN_SRC_FILE=cmd/main.go
GO := GO111MODULE=on go
GO_NOMOD :=GO111MODULE=off go
REV := $(shell git rev-parse --short HEAD 2> /dev/null || echo 'unknown')
ORG := jenkins-x-plugins
ORG_REPO := $(ORG)/$(NAME)
RELEASE_ORG_REPO := $(ORG_REPO)
ROOT_PACKAGE := github.com/$(ORG_REPO)
GO_VERSION := 1.23.3
GO_DEPENDENCIES := $(call rwildcard,pkg/,*.go) $(call rwildcard,cmd/,*.go)

BRANCH     := $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null  || echo 'unknown')
BUILD_DATE := $(shell date +%Y%m%d-%H:%M:%S)
CGO_ENABLED = 0

REPORTS_DIR=$(BUILD_TARGET)/reports

GOTEST := $(GO) test

# set dev version unless VERSION is explicitly set via environment
VERSION ?= $(shell echo "$$(git for-each-ref refs/tags/ --count=1 --sort=-version:refname --format='%(refname:short)' 2>/dev/null)-dev+$(REV)" | sed 's/^v//')

# Build flags for setting build-specific configuration at build time - defaults to empty
#BUILD_TIME_CONFIG_FLAGS ?= ""

# Full build flags used when building binaries. Not used for test compilation/execution.
BUILDFLAGS :=  -ldflags \
  " -X $(ROOT_PACKAGE)/pkg/cmd/version.Version=$(VERSION)\
		-X github.com/jenkins-x-plugins/jx-semanticcheck/pkg/cmd/version.Version=$(VERSION)\
		-X $(ROOT_PACKAGE)/pkg/cmd/version.Revision='$(REV)'\
		-X $(ROOT_PACKAGE)/pkg/cmd/version.Branch='$(BRANCH)'\
		-X $(ROOT_PACKAGE)/pkg/cmd/version.BuildDate='$(BUILD_DATE)'\
		-X $(ROOT_PACKAGE)/pkg/cmd/version.GoVersion='$(GO_VERSION)'\
		$(BUILD_TIME_CONFIG_FLAGS)"

# Some tests expect default values for version.*, so just use the config package values there.
TEST_BUILDFLAGS :=  -ldflags "$(BUILD_TIME_CONFIG_FLAGS)"

ifdef DEBUG
BUILDFLAGS := -gcflags "all=-N -l" $(BUILDFLAGS)
endif

ifdef PARALLEL_BUILDS
BUILDFLAGS += -p $(PARALLEL_BUILDS)
GOTEST += -p $(PARALLEL_BUILDS)
else
# -p 4 seems to work well for people
GOTEST += -p 4
endif

ifdef DISABLE_TEST_CACHING
GOTEST += -count=1
endif

TEST_PACKAGE ?= ./...
COVER_OUT:=$(REPORTS_DIR)/cover.out
COVERFLAGS=-coverprofile=$(COVER_OUT) --covermode=count --coverpkg=./pkg/...

.PHONY: list
list: ## List all make targets
	@$(MAKE) -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

full: check ## Build and run the tests
check: build test ## Build and run the tests

print-version: ## Print version
	@echo $(VERSION)

build: $(GO_DEPENDENCIES) clean ## Build jx-semanticcheck binary for current OS
	CGO_ENABLED=$(CGO_ENABLED) $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/$(BINARY_NAME) $(MAIN_SRC_FILE)

build-all: $(GO_DEPENDENCIES) build make-reports-dir ## Build all files - runtime, all tests etc.
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -run=nope -tags=integration -failfast -short ./... $(BUILDFLAGS)

tidy-deps: ## Cleans up dependencies
	$(GO) mod tidy
	# mod tidy only takes compile dependencies into account, let's make sure we capture tooling dependencies as well
	@$(MAKE) install-generate-deps

.PHONY: make-reports-dir
make-reports-dir:
	mkdir -p $(REPORTS_DIR)

test: ## Run tests with the "unit" build tag
	KUBECONFIG=/cluster/connections/not/allowed CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) --tags=unit -failfast -short ./... $(TEST_BUILDFLAGS)

test-coverage : make-reports-dir ## Run tests and coverage for all tests with the "unit" build tag
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) --tags=unit $(COVERFLAGS) -failfast -short ./... $(TEST_BUILDFLAGS)

test-report-html: make-reports-dir test-coverage
	$(GO) tool cover -html=$(COVER_OUT)

install: $(GO_DEPENDENCIES) ## Install the binary
	GOBIN=${GOPATH}/bin $(GO) install $(BUILDFLAGS) $(MAIN_SRC_FILE)

linux: ## Build for Linux
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/linux/$(BINARY_NAME) $(MAIN_SRC_FILE)
	chmod +x build/linux/$(BINARY_NAME)

arm: ## Build for ARM
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/arm/$(BINARY_NAME) $(MAIN_SRC_FILE)
	chmod +x build/arm/$(BINARY_NAME)

win: ## Build for Windows
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/win/$(BINARY_NAME)-windows-amd64.exe $(MAIN_SRC_FILE)

darwin: ## Build for OSX
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/darwin/$(BINARY_NAME) $(MAIN_SRC_FILE)
	chmod +x build/darwin/$(BINARY_NAME)

.PHONY: release
release: clean linux test

release-all: release linux win darwin

.PHONY: goreleaser
goreleaser:
	step-go-releaser --organisation=$(ORG) --revision=$(REV) --branch=$(BRANCH) --build-date=$(BUILD_DATE) --go-version=$(GO_VERSION) --root-package=$(ROOT_PACKAGE) --version=$(VERSION)

.PHONY: clean
clean: ## Clean the generated artifacts
	rm -rf build release dist

get-fmt-deps: ## Install test dependencies
	$(GO_NOMOD) get golang.org/x/tools/cmd/goimports

.PHONY: fmt
fmt: importfmt ## Format the code
	$(eval FORMATTED = $(shell $(GO) fmt ./...))
	@if [ "$(FORMATTED)" == "" ]; \
      	then \
      	    echo "All Go files properly formatted"; \
      	else \
      		echo "Fixed formatting for: $(FORMATTED)"; \
      	fi

.PHONY: importfmt
importfmt: get-fmt-deps
	@echo "Formatting the imports..."
	goimports -w $(GO_DEPENDENCIES)

.PHONY: lint
lint: ## Lint the code
	./hack/gofmt.sh
	./hack/linter.sh
	./hack/generate.sh

.PHONY: all
all: fmt build test lint

verify-code-unchanged: ## Verify the generated/formatting of code is up to date
	$(eval CHANGED = $(shell git ls-files --modified --others --exclude-standard))
	@if [ "$(CHANGED)" == "" ]; \
      	then \
      	    echo "All generated and formatted files up to date"; \
      	else \
      		echo "Code generation and/or formatting is out of date"; \
      		echo "$(CHANGED)"; \
			git diff; \
      		exit 1; \
      	fi

crd-manifests: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) crd:maxDescLen=0 paths="./pkg/apis/preview/v1alpha1/..." output:crd:artifacts:config=crds

.PHONY: docs
docs: cli-docs crds-docs

DOCS_GEN := bin/gen-docs
$(DOCS_GEN):
	$(GO) build -o bin/gen-docs ./hack/struct-docs.go

	pushd /tmp; $(GO) get -u sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.0; popd

.PHONY: crds-docs
crds-docs: $(DOCS_GEN)
	rm -rf ./docs/crds
	$(DOCS_GEN) --input=./pkg/apis/preview/v1alpha1/... --root=Preview --output=./docs/crds

bin/docs:
	go build $(LDFLAGS) -v -o bin/docs cmd/docs/*.go

.PHONY: cli-docs
cli-docs: bin/docs ## update docs
	@echo "Generating docs"
	@./bin/docs --target=./docs/cmd
	@./bin/docs --target=./docs/man/man1 --kind=man
	@rm -f ./bin/docs


.PHONY: code-generate
code-generate:
	./hack/generate.sh

.PHONY: gen-schema
gen-schema:
	mkdir -p schema
	go run cmd/schemagen/main.go
