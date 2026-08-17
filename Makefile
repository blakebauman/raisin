.PHONY: api worker smtp migrate tidy test console sdk-js sdk-go smoke compose-up compose-apps compose-down

DATABASE_URL ?= postgres://raisin:raisin@localhost:5433/raisin?sslmode=disable

api:
	go run ./apps/api/cmd

worker:
	go run ./apps/worker/cmd

smtp:
	go run ./apps/smtp/cmd

migrate:
	psql "$(DATABASE_URL)" -f migrations/001_init.sql
	psql "$(DATABASE_URL)" -f migrations/002_better_auth.sql

tidy:
	go mod tidy

test:
	go test ./...
	cd packages/sdk-go && go test ./...

smoke:
	chmod +x scripts/smoke.sh
	./scripts/smoke.sh

console:
	cd apps/console && pnpm install && pnpm dev

sdk-js:
	cd packages/sdk-js && pnpm install && pnpm build

sdk-go:
	cd packages/sdk-go && go test ./...

compose-up:
	docker compose up -d

compose-apps:
	docker compose --profile apps up -d --build

compose-down:
	docker compose --profile apps down -v
