SHELL := /bin/bash

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# the go version must match the one set in the go.mod file.
GOVERSION ?= go1.26.1

GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.11.3

MOCKGEN = $(LOCALBIN)/mockgen
MOCKGEN_VERSION ?= v0.6.0

NILAWAY = $(LOCALBIN)/nilaway
# Pinned pseudo-version (go.uber.org/nilaway); bump when adopting newer nilaway fixes.
NILAWAY_VERSION ?= v0.0.0-20260318203545-ad240b12fb4c
NILAWAY_INCLUDE_PKGS ?= go.emeland.io/modelsrv


.PHONY: build
build: test ## Build the project binary.
	go build -ldflags "-s -w" -o bin/modelsrv ./cmd/modelsrv
	go build -ldflags "-s -w" -o bin/emelandctl ./cmd/emelandctl

.PHONY: generate
generate: 
	go generate ./...

.PHONY: test
test: generate fmt vet ## Run tests.
	go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...


.PHONY: lint
lint: golangci-lint $(NILAWAY) ## Run golangci-lint and nilaway
	$(GOLANGCI_LINT) run
	$(NILAWAY) -include-pkgs=$(NILAWAY_INCLUDE_PKGS) ./...

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	# v2.x releases use the /v2 module path (see https://golangci-lint.run/welcome/install/#install-from-sources)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: nilaway
nilaway: $(NILAWAY) ## Run Uber NilAway nilness checks on this module
	$(NILAWAY) -include-pkgs=$(NILAWAY_INCLUDE_PKGS) ./...

$(NILAWAY): $(LOCALBIN)
	$(call go-install-tool,$(NILAWAY),go.uber.org/nilaway/cmd/nilaway,$(NILAWAY_VERSION))

.PHONY: tools
tools: $(MOCKGEN) $(NILAWAY)
$(MOCKGEN): $(LOCALBIN)
	$(call go-install-tool,$(MOCKGEN),go.uber.org/mock/mockgen,$(MOCKGEN_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOTOOLCHAIN=$(GOVERSION) GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
