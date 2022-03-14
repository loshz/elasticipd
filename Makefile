# Configurable env vars
BUILD_NUMBER ?= dev
REGISTRY ?= ghcr.io/loshz/elasticipd
DOCKER ?= sudo docker
GO_TEST_FLAGS ?= -v -failfast

.PHONY: docker-build
docker-build:
	$(DOCKER) build \
	  --build-arg BUILD_NUMBER=$(BUILD_NUMBER) \
	  --tag $(REGISTRY):$(BUILD_NUMBER) .

.PHONY: docker-push
docker-push:
	$(DOCKER) push $(REGISTRY):$(BUILD_NUMBER)

.PHONY: docker-build-push
docker-build-push: docker-build docker-push

.PHONY: test
test:
	go test $(GO_TEST_FLAGS) ./cmd/...
