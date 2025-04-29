###########################################################################
# Variables
###########################################################################

IMAGE       ?= queue-broker
TAG         ?= latest
BINARY      ?= queue-broker
PORT        ?= 8080
COVER       ?= coverage.out

DOCKERFILE  ?= deployments/Dockerfile

.DEFAULT_GOAL := build

###########################################################################
# Go targets
###########################################################################
.PHONY: build
build: ## Build local binary
	go build -tags=integration -o $(BINARY) ./cmd/server

.PHONY: test
test: ## Run unit tests with coverage summary
	go test ./...

.PHONY: cover
cover: ## Generate coverage.out and print summary
	go test ./... -coverprofile=$(COVER)
	go tool cover -func=$(COVER)

.PHONY: cover-html
cover-html: ## Create HTML coverage report
	go test ./... -coverprofile=$(COVER)
	go tool cover -html=$(COVER) -o $(COVER).html
	@echo "HTML report written to $(COVER).html"

.PHONY: run
run: build ## Run locally (PORT env overrides)
	./$(BINARY) -port $(PORT)

.PHONY: lint
lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || { echo >&2 "install golangci-lint first"; exit 1; }
	golangci-lint run ./...

.PHONY: clean
clean: ## Remove built artifacts & coverage files
	rm -f $(BINARY) $(COVER) $(COVER).html

###########################################################################
# Docker targets
###########################################################################
.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -f $(DOCKERFILE) -t $(IMAGE):$(TAG) .

.PHONY: docker-run
docker-run: docker-build ## Run container exposing PORT
	docker run --rm -p $(PORT):$(PORT) $(IMAGE):$(TAG) -port $(PORT)

###########################################################################
# Help
###########################################################################
.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?##' $(firstword $(MAKEFILE_LIST)) | awk 'BEGIN {FS=":"}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
