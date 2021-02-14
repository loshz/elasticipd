# Configurable env vars
BUILD_NUMBER ?= dev
CONTAINER_IMG ?= quay.io/syscll/elasticipd
DOCKER ?= sudo docker
GO_TEST_FLAGS ?= -v -failfast

define HELP_TXT
Use `make <target>` where <target> is one of:
  docker-build      build docker image
  docker-push       push a given docker image to AWS ECR
  test              run all go unit tests
endef

# This should be the first target in order for `make`
# to use it as the default
.PHONY: help
help:
	@: $(info $(HELP_TXT))

.PHONY: docker-build
docker-build:
	$(DOCKER) build --tag $(CONTAINER_IMG):$(BUILD_NUMBER) .

.PHONY: docker-push
docker-push:
	$(DOCKER) push $(CONTAINER_IMG):$(BUILD_NUMBER)

.PHONY: test
test:
	go test $(GO_TEST_FLAGS) ./...
