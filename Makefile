APP_ENV ?= development
ENV_FILE ?= .env.$(APP_ENV)
LOCAL_ENV_FILE ?= $(ENV_FILE).local
RUNTIME_ENV_FILE ?= $(ENV_FILE)

ifeq ($(APP_ENV),development)
ifneq ($(wildcard $(LOCAL_ENV_FILE)),)
RUNTIME_ENV_FILE := $(LOCAL_ENV_FILE)
else ifeq ($(wildcard $(RUNTIME_ENV_FILE)),)
RUNTIME_ENV_FILE := .env.development.example
endif
endif

-include $(ENV_FILE)
-include $(LOCAL_ENV_FILE)
export

APP_NAME ?= service-backend
MAIN := ./cmd
BIN := ./tmp/$(APP_NAME)
APP_VERSION := $(strip $(shell sed -n 's/^APP_VERSION=//p' VERSION))
WORKSPACE_SCHEMA_VERSION := $(strip $(shell sed -n 's/^WORKSPACE_SCHEMA_VERSION=//p' VERSION))
BUILDINFO_PACKAGE := github.com/endge-lab/service-backend/internal/buildinfo
GOCACHE ?= /tmp/$(APP_NAME)-go-build
SQLC_VERSION ?= v1.31.1
SQLC ?= go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
SWAG_VERSION ?= v1.16.6
SWAGGER2OPENAPI_VERSION ?= 7.0.8
FUZZ_TIME ?= 30s
OPENAPI_TMP_DIR := ./tmp/openapi
OPENAPI_SPEC := ./docs/openapi3.yaml
OPENAPI_GENERATED_GO := ./internal/api/http/openapi/openapi.gen.go

DOCKER_COMPOSE_BASE := docker-compose.yml
DOCKER_COMPOSE_DEV := docker-compose.dev.yml
DOCKER_COMPOSE_OBSERVABILITY := docker-compose.observability.yml
DOCKER_COMPOSE_APP := -f $(DOCKER_COMPOSE_BASE) -f $(DOCKER_COMPOSE_DEV)
COMPOSE_ENV_FILE_ARG := $(if $(wildcard $(RUNTIME_ENV_FILE)),--env-file $(RUNTIME_ENV_FILE),)
APP_RUNTIME_ENV_FILE_ARG := APP_RUNTIME_ENV_FILE=$(RUNTIME_ENV_FILE)

MIGRATIONS_DIR ?= ./migrations
POSTGRES_HOST ?= localhost
POSTGRES_PORT ?= 5432
POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
POSTGRES_DATABASE ?= $(if $(POSTGRES_DB),$(POSTGRES_DB),service_backend)
POSTGRES_SCHEMA ?= public
POSTGRES_SSLMODE ?= disable
POSTGRES_DSN := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DATABASE)?sslmode=$(POSTGRES_SSLMODE)&search_path=$(POSTGRES_SCHEMA)

LDFLAGS := -s -w -X $(BUILDINFO_PACKAGE).Version=$(APP_VERSION) -X $(BUILDINFO_PACKAGE).WorkspaceSchemaVersion=$(WORKSPACE_SCHEMA_VERSION)

.PHONY: validate-version
validate-version:
	@test -f VERSION
	@test "$$(grep -Ec '^(APP_VERSION|WORKSPACE_SCHEMA_VERSION)=' VERSION)" -eq 2
	@test "$$(grep -Evc '^(APP_VERSION=[0-9]+\.[0-9]+\.[0-9]+|WORKSPACE_SCHEMA_VERSION=[1-9][0-9]*)$$' VERSION)" -eq 0
	@grep -Eq '^APP_VERSION=[0-9]+\.[0-9]+\.[0-9]+$$' VERSION
	@grep -Eq '^WORKSPACE_SCHEMA_VERSION=[1-9][0-9]*$$' VERSION

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
	rm -rf $(OPENAPI_TMP_DIR)
	mkdir -p $(OPENAPI_TMP_DIR)
	go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init --dir ./cmd,./internal/api/http,./internal/domain/entities --generalInfo main.go --parseInternal --parseDepth 5 --outputTypes json --output $(OPENAPI_TMP_DIR)
	npx --yes swagger2openapi@$(SWAGGER2OPENAPI_VERSION) $(OPENAPI_TMP_DIR)/swagger.json -o $(OPENAPI_SPEC)
	go run ./internal/tools/openapiembed -input $(OPENAPI_SPEC) -output $(OPENAPI_GENERATED_GO)
	rm -rf $(OPENAPI_TMP_DIR)

.PHONY: docs-clean
docs-clean:
	rm -rf $(OPENAPI_TMP_DIR)
	rm -f $(OPENAPI_SPEC) $(OPENAPI_GENERATED_GO)

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

.PHONY: test-unit
test-unit:
	@echo "Running unit, architecture, contract and fuzz seed tests without Docker..."
	go test ./...

.PHONY: test-integration
test-integration:
	@echo "Running isolated PostgreSQL integration tests..."
	go test -tags=integration -count=1 ./test/integration ./test/support

.PHONY: test-e2e
test-e2e:
	@echo "Running full HTTP pipeline against isolated PostgreSQL..."
	go test -tags=e2e -count=1 ./test/e2e

.PHONY: test-critical
test-critical: test-unit test-integration test-e2e

.PHONY: fuzz-transport
fuzz-transport:
	go test ./internal/api/http/v1/shared -run='^$$' -fuzz='^FuzzDecodeAndValidate$$' -fuzztime=$(FUZZ_TIME)
	go test ./internal/api/http/v1/shared -run='^$$' -fuzz='^FuzzIfMatch$$' -fuzztime=$(FUZZ_TIME)

.PHONY: fuzz-documents
fuzz-documents:
	go test ./internal/usecase/documents -run='^$$' -fuzz='^FuzzDocumentInputs$$' -fuzztime=$(FUZZ_TIME)
	go test ./internal/usecase/documents -run='^$$' -fuzz='^FuzzSecretValidation$$' -fuzztime=$(FUZZ_TIME)

.PHONY: fuzz-import
fuzz-import:
	go test ./internal/usecase/workspace_state -run='^$$' -fuzz='^FuzzPortableBundleStructure$$' -fuzztime=$(FUZZ_TIME)

.PHONY: fuzz-auth
fuzz-auth:
	go test ./internal/auth -run='^$$' -fuzz='^FuzzOIDCResolverMalformedToken$$' -fuzztime=$(FUZZ_TIME)

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
build: validate-version
	@echo "Building application..."
	mkdir -p ./tmp
	go build -ldflags="$(LDFLAGS)" -buildvcs=false -o $(BIN) $(MAIN)

.PHONY: run
run: validate-version
	@echo "Running application..."
	APP_ENV=$(APP_ENV) go run -ldflags="$(LDFLAGS)" $(MAIN)

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
	goose -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" up

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back one migration..."
	goose -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" down

.PHONY: compose-observability-up
compose-observability-up:
	docker compose $(COMPOSE_ENV_FILE_ARG) -f $(DOCKER_COMPOSE_OBSERVABILITY) up -d
	# docker compose --env-file .env.development.example -f docker-compose.observability.yml up -d

.PHONY: compose-observability-down
compose-observability-down:
	docker compose $(COMPOSE_ENV_FILE_ARG) -f $(DOCKER_COMPOSE_OBSERVABILITY) down
	# docker compose --env-file .env.development.example -f docker-compose.observability.yml down

.PHONY: compose-observability-clean
compose-observability-clean:
	docker compose $(COMPOSE_ENV_FILE_ARG) -f $(DOCKER_COMPOSE_OBSERVABILITY) down --volumes --remove-orphans
	# docker compose --env-file .env.development.example -f docker-compose.observability.yml down --volumes --remove-orphans

.PHONY: compose-observability-logs
compose-observability-logs:
	docker compose $(COMPOSE_ENV_FILE_ARG) -f $(DOCKER_COMPOSE_OBSERVABILITY) logs -f
	# docker compose --env-file .env.development.example -f docker-compose.observability.yml logs -f

.PHONY: compose-observability-ps
compose-observability-ps:
	docker compose $(COMPOSE_ENV_FILE_ARG) -f $(DOCKER_COMPOSE_OBSERVABILITY) ps
	# docker compose --env-file .env.development.example -f docker-compose.observability.yml ps

.PHONY: compose-up
compose-up:
	$(APP_RUNTIME_ENV_FILE_ARG) docker compose $(COMPOSE_ENV_FILE_ARG) $(DOCKER_COMPOSE_APP) up --build --remove-orphans
	# APP_RUNTIME_ENV_FILE=.env.development.example docker compose --env-file .env.development.example -f docker-compose.yml -f docker-compose.dev.yml up --build --remove-orphans

.PHONY: compose-app
compose-app:
	$(APP_RUNTIME_ENV_FILE_ARG) docker compose $(COMPOSE_ENV_FILE_ARG) $(DOCKER_COMPOSE_APP) up --build --remove-orphans service-backend
	# APP_RUNTIME_ENV_FILE=.env.development.example docker compose --env-file .env.development.example -f docker-compose.yml -f docker-compose.dev.yml up --build --remove-orphans service-backend

.PHONY: compose-down
compose-down:
	$(APP_RUNTIME_ENV_FILE_ARG) docker compose $(COMPOSE_ENV_FILE_ARG) $(DOCKER_COMPOSE_APP) down
	# APP_RUNTIME_ENV_FILE=.env.development.example docker compose --env-file .env.development.example -f docker-compose.yml -f docker-compose.dev.yml down

.PHONY: compose-clean
compose-clean:
	$(APP_RUNTIME_ENV_FILE_ARG) docker compose $(COMPOSE_ENV_FILE_ARG) $(DOCKER_COMPOSE_APP) down --volumes --remove-orphans
	# APP_RUNTIME_ENV_FILE=.env.development.example docker compose --env-file .env.development.example -f docker-compose.yml -f docker-compose.dev.yml down --volumes --remove-orphans

.PHONY: compose-logs
compose-logs:
	$(APP_RUNTIME_ENV_FILE_ARG) docker compose $(COMPOSE_ENV_FILE_ARG) $(DOCKER_COMPOSE_APP) logs -f
	# APP_RUNTIME_ENV_FILE=.env.development.example docker compose --env-file .env.development.example -f docker-compose.yml -f docker-compose.dev.yml logs -f

.PHONY: compose-ps
compose-ps:
	$(APP_RUNTIME_ENV_FILE_ARG) docker compose $(COMPOSE_ENV_FILE_ARG) $(DOCKER_COMPOSE_APP) ps
	# APP_RUNTIME_ENV_FILE=.env.development.example docker compose --env-file .env.development.example -f docker-compose.yml -f docker-compose.dev.yml ps
.PHONY: db
db:
	$(APP_RUNTIME_ENV_FILE_ARG) docker compose $(COMPOSE_ENV_FILE_ARG) $(DOCKER_COMPOSE_APP) exec postgres psql -U $${POSTGRES_USER:-postgres} -d $${POSTGRES_DB:-service_backend}
	# APP_RUNTIME_ENV_FILE=.env.development.example docker compose --env-file .env.development.example -f docker-compose.yml -f docker-compose.dev.yml exec postgres psql -U $${POSTGRES_USER:-postgres} -d $${POSTGRES_DB:-service_backend}
