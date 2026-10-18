SHELL := /bin/sh
.DEFAULT_GOAL := help
.PHONY: demo gateway tasks files notifications llm-svc llm help setup install infra up down logs api web migrate migrate-status test check fmt smoke integration backup seed e2e docs-sync
help:
	@echo "setup: настройки | install: зависимости | infra: инфраструктура | up: весь стек в Docker"
	@echo "gateway/tasks/files/web: разработка | check: проверки и сборка | smoke: проверка API | backup: дамп базы"
setup:
	node scripts/setup.mjs
install: setup
	npm ci
	for s in gateway tasks files llm notifications; do (cd apps/$$s && go mod download) || exit 1; done
infra: setup
	docker compose up -d --wait --wait-timeout 360 tasks-postgres files-postgres notifications-postgres keycloak-postgres gateway-redis files-s3 rabbitmq mailpit keycloak
up: setup
	docker compose --profile app up -d --build --wait --wait-timeout 360
# Qwen в Ollama поднимается отдельно: образ и модель весят несколько гигабайт.
llm: setup
	docker compose --profile llm up -d --wait --wait-timeout 900 llm-ollama llm-ollama-pull
# Показать проект с нуля: стек, демо-данные и ссылки (одна команда для защиты).
demo: up seed
	@node scripts/demo-info.mjs
	@command -v open >/dev/null 2>&1 && open http://localhost:3000 || true
down:
	docker compose --profile app --profile llm down --remove-orphans
logs:
	docker compose --profile app logs -f --tail=100
gateway:
	set -a; . ./.env; set +a; cd apps/gateway && go run ./cmd/gateway
tasks:
	set -a; . ./.env; set +a; export DATABASE_URL=$$TASKS_DATABASE_URL HTTP_ADDR=:8081; cd apps/tasks && go run ./cmd/tasks
files:
	set -a; . ./.env; set +a; export DATABASE_URL=$$FILES_DATABASE_URL HTTP_ADDR=:8082; cd apps/files && go run ./cmd/files
notifications:
	set -a; . ./.env; set +a; export DATABASE_URL=$$NOTIFICATIONS_DATABASE_URL HTTP_ADDR=:8084; cd apps/notifications && go run ./cmd/notifications
llm-svc:
	set -a; . ./.env; set +a; export HTTP_ADDR=:8083; cd apps/llm && go run ./cmd/llm
web:
	npm run dev
migrate:
	set -a; . ./.env; set +a; (cd apps/tasks && DATABASE_URL=$$TASKS_DATABASE_URL go run ./cmd/tasks migrate up) && (cd apps/files && DATABASE_URL=$$FILES_DATABASE_URL go run ./cmd/files migrate up) && (cd apps/notifications && DATABASE_URL=$$NOTIFICATIONS_DATABASE_URL go run ./cmd/notifications migrate up)
migrate-status:
	set -a; . ./.env; set +a; (cd apps/tasks && DATABASE_URL=$$TASKS_DATABASE_URL go run ./cmd/tasks migrate status) && (cd apps/files && DATABASE_URL=$$FILES_DATABASE_URL go run ./cmd/files migrate status)
test:
	for s in gateway tasks files llm notifications; do (cd apps/$$s && go test -race ./...) || exit 1; done
	npm test
check: test
	for s in gateway tasks files llm notifications; do (cd apps/$$s && test -z "$$(gofmt -l .)" && go vet ./... && go build ./...) || exit 1; done
	npm run typecheck
	npm run lint
	npm run format:check
	npm run build
fmt:
	for s in gateway tasks files llm notifications; do (cd apps/$$s && gofmt -w .) || exit 1; done
	npm run format
e2e:
	cd e2e && npm ci && npx playwright install chromium && npx playwright test
docs-sync:
	cp packages/contracts/openapi.yaml apps/gateway/internal/apidocs/openapi.yaml
seed:
	node scripts/seed.mjs
smoke:
	npm run smoke
integration: migrate
	set -a; . ./.env; set +a; (cd apps/tasks && DATABASE_URL=$$TASKS_DATABASE_URL go test -race -tags=integration ./internal/tasks ./internal/broker ./internal/platform) && (cd apps/files && DATABASE_URL=$$FILES_DATABASE_URL go test -race -tags=integration ./internal/files) && (cd apps/gateway && go test -race -tags=integration ./internal/auth)
backup:
	@mkdir -p .local/backups
	for s in tasks files notifications; do docker compose exec -T $$s-postgres sh -c 'exec pg_dump -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -Fc' > .local/backups/$$s-$$(date +%Y%m%d-%H%M%S).dump || exit 1; done
