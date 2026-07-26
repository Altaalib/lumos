.PHONY: build test migrate-up migrate-down

build:
	go build -o bin/reader ./cmd/reader
	go build -o bin/analyzer ./cmd/analyzer
	go build -o bin/notifier ./cmd/notifier

test:
	go test ./... -race -cover

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1
