.PHONY: run build test lint fmt vet docker-up docker-down migrate-up migrate-down

# Application
run:
	go run ./cmd/api

build:
	go build -o todo-api ./cmd/api

test:
	go test -race ./...

test-cover:
	go test -race -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

vet:
	go vet ./...

# Docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

# Migration (requires golang-migrate CLI)
migrate-up:
	migrate -path migrations -database "postgres://todo:todo@localhost:5432/todo?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://todo:todo@localhost:5432/todo?sslmode=disable" down 1
