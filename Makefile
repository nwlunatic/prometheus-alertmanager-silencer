DOCKERFLAGS := --rm -v $(CURDIR):/app:rw -w /app
BUILD_DEV_IMAGE_PATH := golang:1.16.3-buster
IMAGE_PATH := nwlunatic/prometheus-alertmanager-silencer
REF_NAME ?= $(shell git rev-parse --abbrev-ref HEAD)
export IMAGE_VERSION ?= ${REF_NAME}-$(shell git rev-parse HEAD)

TESTS_PKGS := $$(go list ./... | egrep -e '(/tests/)')

.PHONY: clean
clean:
	rm -rf ./bin/*

.PHONY: build
build: clean
	GOOS=linux go build -a -o ./bin/silencer ./src/cmd/silencer

.PHONY: build-image
build-image:
	docker run $(DOCKERFLAGS) $(BUILD_DEV_IMAGE_PATH) make build
	docker build -t $(IMAGE_PATH):$(IMAGE_VERSION) .

.PHONY: tag-image
tag-image:
	docker tag $(IMAGE_PATH):$(IMAGE_VERSION) $(IMAGE_PATH):$(IMAGE_VERSION)

.PHONY: push-image
push-image:
	docker push $(IMAGE_PATH):$(IMAGE_VERSION)

.PHONY: deploy
deploy:
	export IMAGE_VERSION=$(IMAGE_VERSION)
	envsubst < deploy/deploy.yml | kubectl apply -f -

.PHONY: test
test:
	go test -v -race ./src/...

.PHONY: tests-with-infrastructure
tests-with-infrastructure:
	for pkg in $(TESTS_PKGS); do \
		go test -v -race $$pkg || exit 1 ; \
	done;

.PHONY: docker-lint
docker-lint:
	docker run $(DOCKERFLAGS) golangci/golangci-lint:v1.39.0 golangci-lint run -v

.PHONY: docker-test
docker-test:
	docker run $(DOCKERFLAGS) $(BUILD_DEV_IMAGE_PATH) make test

.PHONY: docker-tests-with-infrastructure
docker-tests-with-infrastructure:
	docker-compose run $(DOCKERFLAGS) tests make tests-with-infrastructure

.PHONY: up-infrastructure
up-infrastructure: down
	docker-compose up --build --force-recreate -d alertmanager
	docker-compose run --rm dockerize \
		-wait tcp://alertmanager:9093 \
		-timeout 60s true

.PHONY: down
down:
	@-docker-compose down --remove-orphans --volumes

.PHONY: dev-init
dev-init: export COMPOSE_FILE=docker-compose.dev.yml

.PHONY: dev-up
dev-up: export COMPOSE_FILE=docker-compose.dev.yml
dev-up: dev-down dev-init up-infrastructure
	docker-compose logs -f

.PHONY: dev-down
dev-down: export COMPOSE_FILE=docker-compose.dev.yml
dev-down: down
