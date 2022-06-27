REPO_ROOT ?= $(shell git rev-parse --show-toplevel)
# copy to dir & uncomment
#-include $(REPO_ROOT)/go.mk

# Module override
-include local.mk

COLOR 	:= "\e[1;36m%s\e[0m\n"
RED 	:= "\e[1;31m%s\e[0m\n"

ifdef GOROOT
PATH 	:= $(GOROOT)/bin:$(PATH)
endif

GOBIN 	:= $(if $(shell go env GOBIN),$(shell go env GOBIN),$(GOPATH)/bin)
PATH 	:= $(GOBIN):$(PATH)

GOOS 		?= $(shell go env GOOS)
GOARCH 		?= $(shell go env GOARCH)
GOPATH 		?= $(shell go env GOPATH)
CGO_ENABLED	?= 0

GOMODDIR	?= $(shell dirname $(shell go env GOMOD))
GOMODNAME 	?= $(shell basename $(GOMODDIR))

DOCKER_TAG 	:= $(or $(DOCKER_TAG),$(shell git rev-parse --abbrev-ref HEAD))
BUILD_TAG	:= $(or $(BUILD_TAG),$(shell git rev-parse --abbrev-ref HEAD))

GOFLAGS	?= -trimpath -ldflags "-s -w"

# Extra local ignored actions
-include ignored.mk

##### Golang #####

.PHONY: info
info:
	@echo "GOOS=$(GOOS)"
	@echo "GOARCH=$(GOARCH)"
	@echo "CGO_ENABLED=$(CGO_ENABLED)"
	@echo "GOPROXY=`go env GOPROXY`"
	@echo "DOCKER_REPO=$(DOCKER_REPO)"
	@echo "DOCKER_TAG=$(DOCKER_TAG)"

.PHONY: build
build: ## build binary
	@ls cmd | xargs -n1 -I {} sh -c 'set -x;echo Building {}; GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build $(GOFLAGS) -o build/{}/bin/{} ./cmd/{}'
	@! command -v upx > /dev/null || upx build/*/bin/*
	-@du -sh build/*/bin/*

ifneq ("$(wildcard build/*/Dockerfile)","")
image: GOOS:=linux
image: build # build image
	@ls cmd | xargs -n1 -I {} sh -c 'set -x;echo Building {}; docker build -t {}:$(BUILD_TAG) build/{}'
endif

.PHONY: lint
lint: ## lint
	@printf $(COLOR) "Linting..."
	@[ ! -e .golangci.yml ] || golangci-lint run
	@[ ! -e "$(REPO_ROOT)/.golangci.yml" ] || { printf $(COLOR) "Using root .golangci.yml" ; golangci-lint run -c "$(REPO_ROOT)/.golangci.yml"; }

.PHONY: fmt
fmt: tidy ## tidy,format and imports
	[ ! -e buf.gen.yaml ] || buf format -w
	gofumpt -w `find . -type f -name '*.go' -not -path "./vendor/*"`
	goimports -w `find . -type f -name '*.go' -not -path "./vendor/*"`

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

.PHONY: gen
gen: ## generate
	[ ! -e buf.gen.yaml ] || buf generate `ls -d proto/* | grep -v bundle`
	$(MAKE) fmt

.PHONY: go-test-cover
go-test-cover: ## run test & generate coverage
	@printf $(COLOR) "Running test with coverage..."
	@go test -race -coverprofile=cover.out -coverpkg=./... ./...
	@go tool cover -html=cover.out -o cover.html

.PHONY: go-mod-up
go-mod-up: ## update go dependencies
	@printf $(COLOR) "Update dependencies..."
	@go get -u -t $(PINNED_DEPENDENCIES) ./...
	@go mod tidy

.PHONY: go-list-cgo
go-list-cgo: ## List cgo modules
	@printf $(COLOR) "List cgo module..."
	@go list -f "{{if .CgoFiles}}{{.ImportPath}}{{end}}" `go list -f "{{.ImportPath}}{{range .Deps}} {{.}}{{end}}" ./...`

##### ecosystem #####

ifneq ("$(wildcard ent/schema)","")
go-ent-describe:
	go run entgo.io/ent/cmd/ent describe ./ent/schema
go-ent-gen:
	go generate ./ent
	$(MAKE) fmt
	git add ent
endif

ifneq ("$(wildcard atlas.hcl)","")
atlas-migrate-new:
	atlas migrate new
atlas-migrate-hash-force:
	atlas migrate hash --force
atlas-migrate-migrate:
	atlas migrate validate
endif

##### Generic #####

ensure-no-changes: ## ensure git doesn't have any changes
	@printf $(COLOR) "Check for local changes..."
	@printf $(COLOR) "========================================================================"
	@git diff --name-status --exit-code || (printf $(COLOR) "========================================================================"; printf $(RED) "Above files are not regenerated properly. Regenerate them and try again."; exit 1)

clean: ## cleanup build
	rm -rf build/*/bin/*
	rm -rf bazel-bin/*
	[ ! -e BUILD.bazel ] || bazel clean

.PHONY: help
.DEFAULT_GOAL := help
help: ## show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
