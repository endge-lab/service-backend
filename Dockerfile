# syntax=docker/dockerfile:1.7

ARG BASE_BUILDER_IMAGE=golang:1.26.1-alpine
ARG BASE_RUNTIME_IMAGE=alpine:3.21
FROM ${BASE_BUILDER_IMAGE} AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN app_version="$(sed -n 's/^APP_VERSION=//p' VERSION)" \
  && workspace_schema_version="$(sed -n 's/^WORKSPACE_SCHEMA_VERSION=//p' VERSION)" \
  && test "$(grep -Ec '^(APP_VERSION|WORKSPACE_SCHEMA_VERSION)=' VERSION)" -eq 2 \
  && test "$(grep -Evc '^(APP_VERSION=[0-9]+\.[0-9]+\.[0-9]+|WORKSPACE_SCHEMA_VERSION=[1-9][0-9]*)$' VERSION)" -eq 0 \
  && echo "$app_version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$' \
  && echo "$workspace_schema_version" | grep -Eq '^[1-9][0-9]*$' \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath \
  -ldflags="-s -w -X github.com/endge-lab/service-backend/internal/buildinfo.Version=$app_version -X github.com/endge-lab/service-backend/internal/buildinfo.WorkspaceSchemaVersion=$workspace_schema_version" \
  -buildvcs=false -o /out/service-backend ./cmd

FROM ${BASE_RUNTIME_IMAGE}

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app \
  && apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/service-backend /app/service-backend
COPY migrations /app/migrations
COPY production.yaml /app/production.yaml

USER app

EXPOSE 8080

ENTRYPOINT ["/app/service-backend"]
