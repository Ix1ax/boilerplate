SHELL := /bin/sh
.DEFAULT_GOAL := help
.PHONY: help setup install infra up down logs api web migrate migrate-status test check fmt smoke integration backup
help:
	@echo "setup: настройки | install: зависимости | infra: инфраструктура | up: весь стек в Docker"
	@echo "api/web: разработка | check: проверки и сборка | smoke: проверка API | backup: дамп базы"
setup:
	node scripts/setup.mjs
install: setup
	npm ci
	cd apps/api && go mod download
infra: setup
	docker compose up -d --wait --wait-timeout 360 postgres redis s3 mailpit keycloak
up: setup
	docker compose --profile app up -d --build --wait --wait-timeout 360
down:
	docker compose --profile app down
logs:
	docker compose --profile app logs -f --tail=100
api:
	set -a; . ./.env; set +a; cd apps/api && go run ./cmd/api
web:
	npm run dev
migrate:
	set -a; . ./.env; set +a; cd apps/api && go run ./cmd/migrate up
migrate-status:
	set -a; . ./.env; set +a; cd apps/api && go run ./cmd/migrate status
test:
	cd apps/api && go test -race ./...
	npm test
check: test
	cd apps/api && test -z "$$(gofmt -l .)" && go vet ./... && go build ./cmd/...
	npm run typecheck
	npm run lint
	npm run format:check
	npm run build
fmt:
	cd apps/api && gofmt -w .
	npm run format
smoke:
	npm run smoke
integration: migrate
	set -a; . ./.env; set +a; cd apps/api && go test -race -tags=integration ./internal/tasks ./internal/files ./internal/auth
backup:
	@mkdir -p .local/backups
	docker compose exec -T postgres sh -c 'exec pg_dump -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -Fc' > .local/backups/database-$$(date +%Y%m%d-%H%M%S).dump
