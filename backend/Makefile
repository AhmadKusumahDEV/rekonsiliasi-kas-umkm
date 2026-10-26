ifeq ($(OS),Windows_NT)
SHELL := cmd.exe
endif

APP_MODULE := github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend
# Override saat build: make build VERSION=$(git rev-parse --short HEAD)
VERSION ?= dev
LDFLAGS := -s -w -X $(APP_MODULE)/internal/service.Version=$(VERSION)
BIN := bin/api$(shell go env GOEXE)
PG_MIGRATIONS := scripts/postgres/migrations

.DEFAULT_GOAL := help

.PHONY: help run build test fmt tidy lint check sqlc-install migrate-install gen.pg.models gen.pg.migration pg.migrate.up pg.migrate.down db.up db.down

help: ## Tampilkan daftar target
	@echo Available targets:
	@echo   make run
	@echo   make build
	@echo   make test
	@echo   make tidy
	@echo   make gen.pg.models
	@echo   make gen.pg.migration name=create_users
	@echo   make pg.migrate.up DATABASE_URL=postgres://...
	@echo   make db.up

run: ## Jalankan API lokal
	go run ./cmd/api

build: ## Build binary API
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/api

test: ## Jalankan semua test
	go test ./...

fmt: ## Format kode Go
	gofmt -w cmd internal pkg

tidy: ## Sinkronkan go.mod/go.sum tanpa vendor
	go mod tidy

lint: ## Jalankan golangci-lint
	golangci-lint run ./...

check: build test ## Build dan test

sqlc-install: ## Pasang sqlc CLI
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1

migrate-install: ## Pasang golang-migrate CLI
	go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest

gen.pg.models: ## Generate query type-safe dari scripts/postgres
	cd scripts/postgres && sqlc generate -f sqlc.yaml

gen.pg.migration: ## Buat migration. Contoh: make gen.pg.migration name=add_users
	migrate create -ext sql -dir $(PG_MIGRATIONS) -seq $(name)

pg.migrate.up: ## Jalankan migration. Isi DATABASE_URL
	migrate -path $(PG_MIGRATIONS) -database "$(DATABASE_URL)" up

pg.migrate.down: ## Rollback satu migration
	migrate -path $(PG_MIGRATIONS) -database "$(DATABASE_URL)" down 1

db.up: ## Nyalakan PostgreSQL lokal lewat Docker Compose
	docker compose up -d db

db.down: ## Matikan PostgreSQL lokal
	docker compose down
