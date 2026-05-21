BINARY=famcal-server
MODULE=github.com/brycesharrits/fam-cal-insta

# Load .env for targets that need DATABASE_URL etc. (only if present)
ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: build run dev tidy migrate-up migrate-down docker-up docker-down

build:
	go build -o bin/$(BINARY) ./cmd/server

run: build
	./bin/$(BINARY)

dev:
	go run ./cmd/server

tidy:
	go mod tidy

# Database migrations (requires golang-migrate CLI: brew install golang-migrate)
migrate-up:
	migrate -path internal/repository/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path internal/repository/migrations -database "$(DATABASE_URL)" down 1

# Local development dependencies (PostgreSQL + MinIO)
docker-up:
	docker compose up -d

docker-down:
	docker compose down

test:
	go test ./...

lint:
	golangci-lint run
