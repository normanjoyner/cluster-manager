PROJECT_NAME := "cloud-agent"
PKG := "github.com/containership/$(PROJECT_NAME)"
PKG_LIST := $(shell glide novendor)
GO_FILES := $(shell find . -type f -not -path './vendor/*' -name '*.go' | grep -v _test.go)

# TODO golint has no way to ignore specific files or directories, so we have to
# manually build a lint list. This workaround can go away and we can use
# $PKG_LIST when the k8s code generator is updated to follow golang conventions
# for generated files. See https://github.com/kubernetes/code-generator/issues/30.
LINT_LIST := $(shell go list ./... | grep -v '/pkg/client')

.PHONY: all fmt-check lint test vet dep build clean coverage coverhtml

all: build

fmt-check: ## Check the file format
	@gofmt -e -d ${GO_FILES}

lint: ## Lint the files
	@golint -set_exit_status ${LINT_LIST}

test: ## Run unittests
	@go test -short ${PKG_LIST}

vet: ## Vet the files
	@go vet ${PKG_LIST}

## Read about data race https://golang.org/doc/articles/race_detector.html
## to not test file for race use `// +build !race` at top
race: dep ## Run data race detector
	@go test -race -short ${PKG_LIST}

msan: dep ## Run memory sanitizer (only works on linux/amd64)
	@go test -msan -short ${PKG_LIST}

# coverage: ## Generate global code coverage report
#	./tools/coverage.sh;

# coverhtml: ## Generate global code coverage report in HTML
#	./tools/coverage.sh html;

dep: ## Get the dependencies
	@go get -v -d ./...

build: dep ## Build the binary file
	@go build -i -v $(PKG)

clean: ## Remove previous build
	@rm -f $(PROJECT_NAME)

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

### Comands for local development
mount:
	@echo "Setting up mount as symlink in /Users/.minikube-mounts folder"
	$(shell sudo mkdir /Users/.minikube-mounts)
	$(shell sudo ln -s ${PWD}/internal/ /Users/.minikube-mounts/internal)
	@echo $(shell ls /Users/.minikube-mounts)

deploy-crds:
	kubectl apply -f deploy/common/containership-users-crd.yaml
	kubectl apply -f deploy/common/containership-registries-crd.yaml

deploy-common: deploy-crds
	kubectl apply -f deploy/common/containership-core-namespace.yaml
	kubectl apply -f deploy/common/containership-env-configmap.yaml

deploy-agent: deploy-common
	kubectl apply -f deploy/development/agent.yaml

deploy-coordinator: deploy-common
	kubectl apply -f deploy/development/coordinator.yaml

deploy: deploy-agent deploy-coordinator

build-agent:
	@eval $$(minikube docker-env) ;\
	docker image build -t containership/cloud-agent -f Dockerfile.agent .

agent: build-agent deploy-agent

build-coordinator:
	@eval $$(minikube docker-env) ;\
	docker image build -t containership/cloud-agent-coordinator -f Dockerfile.coordinator .

coordinator: build-coordinator deploy-coordinator
