PROJECT_NAME := "cloud-agent"
PKG := "github.com/containership/$(PROJECT_NAME)"
PKG_LIST := $(shell glide novendor)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)

.PHONY: all dep build clean test coverage coverhtml lint

all: build

lint: ##Lint the files
	@golint -set_exit_status ${PKG_LIST}

test: ## Run unittests
	@go test -short ${PKG_LIST}

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

build-agent:
	@eval $$(minikube docker-env) ;\
	docker image build -t agent:debug -f Dockerfile.agent .

deploy-agent:
	kubectl apply -f deploy/development/agent.yaml

agent: build-agent deploy-agent

build-coordinator:
	@eval $$(minikube docker-env) ;\
	docker image build -t coordinator:debug -f Dockerfile.coordinator .

deploy-coordinator:
	kubectl apply -f deploy/development/coordinator.yaml

coordinator: build-coordinator deploy-coordinator
