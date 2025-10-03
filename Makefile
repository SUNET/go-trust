MODULE = $(shell go list -m)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || echo "1.0.0")
# Extract Go version from go.mod, removing the 'go ' prefix and any potential trailing spaces
GO_VERSION := $(shell grep -E '^go [0-9]+\.[0-9]+\.[0-9]+' go.mod | sed 's/go //g' | tr -d ' ')
# Extract major.minor without patch version for Docker image tags
GO_VERSION_MINOR := $(shell echo $(GO_VERSION) | sed -E 's/^([0-9]+\.[0-9]+).*/\1/')
PACKAGES := $(shell go list ./... | grep -v /vendor/)
LDFLAGS := -ldflags "-X main.Version=${VERSION}"
GOBIN ?= $$(go env GOPATH)/bin

.PHONY: install-go-test-coverage
install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

.PHONY: check-coverage ## check test coverage and generate report
check-coverage: check-go-version install-go-test-coverage ## generate coverage report
	go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
	${GOBIN}/go-test-coverage --config=./.testcoverage.yml

.PHONY: default
default: check-go-version build

.PHONY: check-go-version
check-go-version: ## Check if the current Go version matches the one required by go.mod
	@go version | grep -q "go$(GO_VERSION)" || (echo "Error: Go version mismatch. Required: $(GO_VERSION), Current: $$(go version | awk '{print $$3}' | sed 's/go//')" && exit 1)
	@echo "Using Go version: $(GO_VERSION)"

.PHONY: install
install: ## Install the binary to GOPATH/bin
	CGO_ENABLED=1 go install ${LDFLAGS} -trimpath ./cmd/main.go

.PHONY: run
run: check-go-version build ## Run the application (requires pipeline.yaml argument)
	./gt $(ARGS)

# generate help info from comments: thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help: ## help information about make commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: check-go-version ## run tests with coverage, race detection, and timeout
	go test -v -race -timeout 10m -count=1 -p 4 -coverprofile=cover.out -covermode=atomic ./... && \
	go tool cover -func=cover.out | tail -n 1 | awk '{ print "Total coverage: " $$3 }'

.PHONY: build
build: check-go-version ## build the library
	CGO_ENABLED=1 go build ${LDFLAGS} -trimpath -o gt -a ./cmd/main.go

.PHONY: clean
clean: ## remove temporary files
	go clean

.PHONY: deps
deps: ## Update dependencies
	go get -u ./...
	@echo "Don't forget to run 'make tidy' to clean up the go.mod file"

.PHONY: tidy
tidy: ## Clean up dependencies
	go mod tidy

.PHONY: gosec
gosec: ## Run security checks with gosec
	$(info Run gosec)
	# G107 is excluded because where http.Get(url) is used the url can't be a constant.
	gosec -exclude=G107 -color -nosec -tests ./...

.PHONY: staticcheck
staticcheck: ## Run static analysis with staticcheck
	$(info Run staticcheck)
	staticcheck ./...

.PHONY: lint
lint: ## Run linters (golangci-lint, gosec, staticcheck)
	$(info Run golangci-lint)
	golangci-lint run ./...
	$(MAKE) gosec
	$(MAKE) staticcheck

vscode: ## Install dependencies for VSCode development
	$(info Install APT packages)
	sudo apt-get update && sudo apt-get install -y \
		protobuf-compiler \
		netcat-openbsd
	$(info Install go packages)
	go install golang.org/x/tools/cmd/deadcode@latest && \
	go install github.com/securego/gosec/v2/cmd/gosec@latest && \
	go install honnef.co/go/tools/cmd/staticcheck@latest && \
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
	go install github.com/xuri/xgen/cmd/xgen@latest

.PHONY: docker
docker: check-go-version build ## Build a minimal Docker image
	docker build -t go-trust-status-lists:${VERSION} -t go-trust-status-lists:latest .

# Dockerfile for minimal image
Dockerfile:
	echo 'FROM golang:$(GO_VERSION_MINOR)-alpine AS builder' > Dockerfile
	echo 'WORKDIR /src' >> Dockerfile
	echo 'COPY . .' >> Dockerfile
	echo 'RUN apk add --no-cache build-base' >> Dockerfile
	echo 'RUN CGO_ENABLED=1 go build -ldflags "-X main.Version=${VERSION} -s -w" -trimpath -o app ./cmd/main.go' >> Dockerfile
	echo 'FROM alpine:latest' >> Dockerfile
	echo 'RUN apk add --no-cache libc6-compat ca-certificates' >> Dockerfile
	echo 'COPY --from=builder /src/app /app' >> Dockerfile
	echo 'ENTRYPOINT ["/app"]' >> Dockerfile
