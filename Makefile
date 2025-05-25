BIN := "./bin/gomigrator"
DOCKER_IMG="gomigrator:develop"
CONFIG="configs/config.yml"
MIGRATION_DIR_PATH="migrations/"
TEST_DB_CONTAINER_NAME="db_test"

.PHONY: build test integration-test lint build-img up down redo status dbversion

build:
	go build -v -o $(BIN) ./cmd/gomigrator

test:
	go test -race -count 100 ./pkg/...

integration-test:
	docker run -d -p 5432:5432 --name $(TEST_DB_CONTAINER_NAME) \
 	-e POSTGRES_USER=otus \
 	-e POSTGRES_PASSWORD=123456 \
 	-e POSTGRES_DB=migrator \
 	postgres:15
	@bash -c 'until docker exec $(TEST_DB_CONTAINER_NAME) pg_isready -U otus > /dev/null 2>&1; do sleep 1; done'
	@bash -c '\
            TEST_DB_DSN="postgres://otus:123456@localhost:5432/migrator?sslmode=disable" \
            go test -count=1 -tags integration ./pkg/... ; \
            EXIT_CODE=$$?; \
            docker stop $(TEST_DB_CONTAINER_NAME) > /dev/null || true; \
            docker rm $(TEST_DB_CONTAINER_NAME) > /dev/null || true; \
            exit $$EXIT_CODE'

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.64.6

lint: install-lint-deps
	golangci-lint run ./...

build-img:
	docker build \
		--build-arg CONFIG=$(CONFIG) \
		--build-arg MIGRATION_DIR_PATH=$(MIGRATION_DIR_PATH) \
		-t $(DOCKER_IMG) \
		-f build/Dockerfile .

up: build-img
	docker run --rm $(DOCKER_IMG) up

down: build-img
	docker run --rm $(DOCKER_IMG) down

redo: build-img
	docker run --rm $(DOCKER_IMG) redo

status: build-img
	docker run --rm $(DOCKER_IMG) status

dbversion: build-img
	docker run --rm $(DOCKER_IMG) dbversion
