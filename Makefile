SHELL := /bin/sh
.DEFAULT_GOAL := help
.PHONY: gateway tasks files llm help setup install infra up down logs api web migrate migrate-status test check fmt smoke integration backup
help:
	@echo "setup: настройки | install: зависимости | infra: инфраструктура | up: весь стек в Docker"
	@echo "gateway/tasks/files/web: разработка | check: проверки и сборка | smoke: проверка API | backup: дамп базы"
setup:
	node scripts/setup.mjs
install: setup
	npm ci
	for s in gateway tasks files; do (cd apps/$$s && go mod download) || exit 1; done
infra: setup
	docker compose up -d --wait --wait-timeout 360 postgres redis s3 mailpit keycloak
up: setup
	docker compose --profile app up -d --build --wait --wait-timeout 360
# Qwen в Ollama поднимается отдельно: образ и модель весят несколько гигабайт.
llm: setup
	docker compose --profile llm up -d --wait --wait-timeout 900 ollama ollama-pull
down:
	docker compose --profile app --profile llm down
logs:
	docker compose --profile app logs -f --tail=100
gateway:
	set -a; . ./.env; set +a; cd apps/gateway && go run ./cmd/gateway
tasks:
	set -a; . ./.env; set +a; export DATABASE_URL=$$TASKS_DATABASE_URL HTTP_ADDR=:8081; cd apps/tasks && go run ./cmd/tasks
files:
	set -a; . ./.env; set +a; export DATABASE_URL=$$FILES_DATABASE_URL HTTP_ADDR=:8082; cd apps/files && go run ./cmd/files
web:
	npm run dev
migrate:
	set -a; . ./.env; set +a; (cd apps/tasks && DATABASE_URL=$$TASKS_DATABASE_URL go run ./cmd/tasks migrate up) && (cd apps/files && DATABASE_URL=$$FILES_DATABASE_URL go run ./cmd/files migrate up)
migrate-status:
	set -a; . ./.env; set +a; (cd apps/tasks && DATABASE_URL=$$TASKS_DATABASE_URL go run ./cmd/tasks migrate status) && (cd apps/files && DATABASE_URL=$$FILES_DATABASE_URL go run ./cmd/files migrate status)
test:
	for s in gateway tasks files; do (cd apps/$$s && go test -race ./...) || exit 1; done
	npm test
check: test
	for s in gateway tasks files; do (cd apps/$$s && test -z "$$(gofmt -l .)" && go vet ./... && go build ./cmd/...) || exit 1; done
	npm run typecheck
	npm run lint
	npm run format:check
	npm run build
fmt:
	for s in gateway tasks files; do (cd apps/$$s && gofmt -w .) || exit 1; done
	npm run format
smoke:
	npm run smoke
integration: migrate
	set -a; . ./.env; set +a; (cd apps/tasks && DATABASE_URL=$$TASKS_DATABASE_URL go test -race -tags=integration ./internal/tasks) && (cd apps/files && DATABASE_URL=$$FILES_DATABASE_URL go test -race -tags=integration ./internal/files) && (cd apps/gateway && go test -race -tags=integration ./internal/auth)
backup:
	@mkdir -p .local/backups
	docker compose exec -T postgres sh -c 'exec pg_dump -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -Fc' > .local/backups/database-$$(date +%Y%m%d-%H%M%S).dump
