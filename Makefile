.PHONY: api worker smtp migrate tidy test console sdk-js sdk-go cli mcp smoke seed compose-up compose-apps compose-down helm-sync helm-sync-check

DATABASE_URL ?= postgres://raisin:raisin@localhost:5433/raisin?sslmode=disable
HELM_MIG_DIR := deploy/helm/raisin/files/migrations
CONSOLE_PORT ?= 3001

api:
	go run ./apps/api/cmd

worker:
	go run ./apps/worker/cmd

smtp:
	go run ./apps/smtp/cmd

cli:
	go run ./apps/cli

mcp:
	pnpm --filter @raisin-run/mcp-server start

migrate:
	chmod +x scripts/migrate.sh
	./scripts/migrate.sh migrations

seed:
	chmod +x scripts/seed-dev.sh
	./scripts/seed-dev.sh

helm-sync:
	mkdir -p $(HELM_MIG_DIR)
	rsync -a --delete --include='*.sql' --exclude='*' migrations/ $(HELM_MIG_DIR)/
	cp scripts/migrate.sh $(HELM_MIG_DIR)/migrate.sh
	chmod +x $(HELM_MIG_DIR)/migrate.sh

helm-sync-check: helm-sync
	@git diff --exit-code -- $(HELM_MIG_DIR) || \
	  (echo "Helm migrations out of sync — run make helm-sync and commit"; exit 1)

tidy:
	go mod tidy

test:
	go test ./...
	cd packages/sdk-go && go test ./...

smoke:
	chmod +x scripts/smoke.sh
	./scripts/smoke.sh

console:
	cd apps/console && pnpm install && PORT=$(CONSOLE_PORT) BETTER_AUTH_URL=http://localhost:$(CONSOLE_PORT) pnpm dev

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
