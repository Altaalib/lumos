.PHONY: build test migrate-up migrate-down up down run-reader run-analyzer run-notifier

build:
	go build -o bin/reader ./cmd/reader
	go build -o bin/analyzer ./cmd/analyzer
	go build -o bin/notifier ./cmd/notifier

test:
	go test ./... -race -cover

# Локальный Postgres для разработки.
up:
	docker compose up -d

down:
	docker compose down

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

run-reader:
	go run ./cmd/reader

run-analyzer:
	go run ./cmd/analyzer

run-notifier:
	go run ./cmd/notifier
