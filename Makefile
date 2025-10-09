GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

ifndef VERBOSE
.SILENT:
endif

all: help

help: ## Show this help.
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)
.PHONY: help

test: ## Run tests
	bash -c 'diff -u <(echo -n) <(gofmt -s -d .)'
	go vet ./...
	go test ./... -v -covermode=atomic -coverprofile=coverage.out
.PHONY: test

lint: ## golangci-lint linting
	 golangci-lint run --config golangci.yml ./...
.PHONY: lint
