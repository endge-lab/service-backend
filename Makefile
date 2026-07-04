APP_ENV ?= development
ENV_FILE ?= .env.$(APP_ENV)
LOCAL_ENV_FILE ?= $(ENV_FILE).local
RUNTIME_ENV_FILE ?= $(ENV_FILE)

ifeq ($(APP_ENV),development)
ifneq ($(wildcard $(LOCAL_ENV_FILE)),)
RUNTIME_ENV_FILE := $(LOCAL_ENV_FILE)
endif
endif

-include $(ENV_FILE)
-include $(LOCAL_ENV_FILE)
export

APP_NAME ?= service-backend
MAIN := ./cmd
BIN := ./tmp/$(APP_NAME)
GOCACHE ?= /tmp/$(APP_NAME)-go-build
SQLC ?= $(shell command -v sqlc 2>/dev/null || command -v /tmp/bin/sqlc 2>/dev/null)

DOCKER_COMPOSE_DEV := docker-compose.dev.yml
COMPOSE_ENV_FILE_ARG := $(if $(wildcard $(RUNTIME_ENV_FILE)),--env-file $(RUNTIME_ENV_FILE),)

MIGRATIONS_DIR ?= ./migrations
DATABASE_URI ?= postgres://postgres:postgres@localhost:5432/service_backend?sslmode=disable

LDFLAGS := -s -w

.PHONY: all
all: mod lint test build

.PHONY: tools
tools:
	@echo "Installing tools..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/air-verse/air@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: mod
mod:
	@echo "Tidying dependencies..."
	go mod tidy -v

.PHONY: mod-update
mod-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy -v

.PHONY: docs
docs:
	swag init --parseDependency --parseInternal --parseDepth 5 -g ./cmd/main.go
	swagger2openapi ./docs/swagger.json -o ./docs/openapi3.yaml
	rm -f ./docs/swagger.json
	rm -f ./docs/swagger.yaml

.PHONY: docs-clean
docs-clean:
	rm -f docs/swagger.json docs/swagger.yaml docs/openapi3.yaml

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run ./...

.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

.PHONY: test-race
test-race:
	@echo "Running race tests..."
	go test -race -v ./...

.PHONY: test-cover
test-cover:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out

.PHONY: build
build:
	@echo "Building application..."
	mkdir -p ./tmp
	go build -ldflags="$(LDFLAGS)" -buildvcs=false -o $(BIN) $(MAIN)

.PHONY: run
run:
	@echo "Running application..."
	APP_ENV=$(APP_ENV) go run $(MAIN)

.PHONY: dev
dev:
	@echo "Running with Air..."
	APP_ENV=$(APP_ENV) air

.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -rf tmp coverage.out

.PHONY: sqlc
sqlc:
	@echo "Generating sqlc code..."
	@test -n "$(SQLC)" || (echo "sqlc is not installed; run make tools or set SQLC=/path/to/sqlc"; exit 1)
	$(SQLC) generate

.PHONY: create-migration
create-migration:
	@test -n "$(name)" || (echo "usage: make create-migration name=create_users"; exit 1)
	goose -dir $(MIGRATIONS_DIR) create $(name) sql

.PHONY: migrate-up
migrate-up:
	@echo "Applying migrations..."
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URI)" up

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back one migration..."
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URI)" down

.PHONY: docker-up
docker-up:
	APP_RUNTIME_ENV_FILE=$(RUNTIME_ENV_FILE) docker compose $(COMPOSE_ENV_FILE_ARG) -f $(DOCKER_COMPOSE_DEV) up --build --remove-orphans

.PHONY: docker-up-app
docker-up-app:
	APP_RUNTIME_ENV_FILE=$(RUNTIME_ENV_FILE) docker compose $(COMPOSE_ENV_FILE_ARG) -f $(DOCKER_COMPOSE_DEV) up --build --remove-orphans app

.PHONY: docker-down
docker-down:
	APP_RUNTIME_ENV_FILE=$(RUNTIME_ENV_FILE) docker compose $(COMPOSE_ENV_FILE_ARG) -f $(DOCKER_COMPOSE_DEV) down

.PHONY: docker-clean
docker-clean:
	APP_RUNTIME_ENV_FILE=$(RUNTIME_ENV_FILE) docker compose $(COMPOSE_ENV_FILE_ARG) -f $(DOCKER_COMPOSE_DEV) down --volumes --remove-orphans

.PHONY: docker-recreate
docker-recreate:
	APP_RUNTIME_ENV_FILE=$(RUNTIME_ENV_FILE) docker compose $(COMPOSE_ENV_FILE_ARG) -f $(DOCKER_COMPOSE_DEV) up --build --force-recreate --remove-orphans

.PHONY: db
db:
	docker compose $(COMPOSE_ENV_FILE_ARG) -f $(DOCKER_COMPOSE_DEV) exec postgres psql -U $${POSTGRES_USER:-postgres} -d $${POSTGRES_DB:-service_backend}
