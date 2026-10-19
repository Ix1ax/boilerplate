SHELL := /bin/sh
K6_IMAGE := grafana/k6:0.54.0
.DEFAULT_GOAL := help
.PHONY: build prefetch prefetch-llm reset slides loadtest demo gateway tasks files notifications audit events llm-svc llm help setup install infra up down logs api web web-client web-admin migrate migrate-status test check fmt smoke integration backup seed e2e docs-sync openapi
help:
	@echo "setup: настройки | install: зависимости | infra: инфраструктура | up: весь стек в Docker"
	@echo "gateway/tasks/files/web: разработка | check: проверки и сборка | smoke: проверка API | backup: дамп базы"
setup:
	node scripts/setup.mjs
install: setup
	npm ci
	for s in gateway tasks files llm notifications audit realtime events; do (cd server/$$s && go mod download) || exit 1; done
infra: setup
	docker compose up -d --wait --wait-timeout 360 tasks-postgres files-postgres notifications-postgres audit-postgres events-postgres keycloak-postgres gateway-redis files-s3 rabbitmq mailpit keycloak
# Собираем только сервисы с Dockerfile и по одному: параллельная сборка не помещается в 8 ГБ памяти Docker.
build: setup
	@set -e; services=$$(docker compose --profile app config --format json | \
		node -e 'const c = JSON.parse(require("node:fs").readFileSync(0, "utf8")); console.log(Object.entries(c.services).filter(([, s]) => s.build).map(([name]) => name).join(" "))'); \
	for s in $$services; do docker compose --profile app build $$s; done
up: build
	docker compose --profile app up -d --wait --wait-timeout 360
# Qwen в Ollama поднимается отдельно: образ и модель весят несколько гигабайт.
llm: setup
	docker compose --profile llm up -d --wait --wait-timeout 900 llm-ollama llm-ollama-pull
# Скачать заранее всё, что нужно для работы без интернета: образы, базовые образы сборки,
# зависимости Go и npm, Chromium для E2E, k6 и модель Qwen. Без модели: make prefetch SKIP_LLM=1.
prefetch: setup
	docker compose --profile app pull --ignore-buildable
	$(MAKE) build
	docker pull $(K6_IMAGE)
	for s in gateway tasks files llm notifications audit realtime events; do (cd server/$$s && go mod download) || exit 1; done
	npm ci
	cd e2e && npm ci && npx playwright install chromium
	@if [ "$(SKIP_LLM)" != "1" ]; then $(MAKE) prefetch-llm; else echo "Модель Qwen пропущена (SKIP_LLM=1)"; fi
	@echo "Готово: всё скачано, проект запускается без интернета."
# Образ Ollama (~1,7 ГБ) и модель (~1 ГБ) — только здесь, чтобы SKIP_LLM=1 ничего из этого не качал.
prefetch-llm: llm
	docker compose --profile llm stop llm-ollama
# Чистое демо перед защитой: стирает данные всех сервисов (модель Qwen сохраняется) и поднимает стек заново.
reset:
	@if [ "$(YES)" != "1" ]; then printf "Стереть все данные сервисов и поднять чистое демо? [y/N] "; read answer; [ "$$answer" = "y" ] || { echo "Отменено"; exit 1; }; fi
	docker compose --profile app --profile llm down --remove-orphans
	for v in tasks-postgres-data files-postgres-data notifications-postgres-data audit-postgres-data events-postgres-data keycloak-postgres-data files-s3-data rabbitmq-data; do docker volume rm -f "$${COMPOSE_PROJECT_NAME:-hackathon-starter}_$$v" >/dev/null; done
	$(MAKE) demo
# Презентация из docs/presentation.md (Marp) → docs/presentation.pdf. Использует Chromium из Playwright (make prefetch),
# иначе системный Chrome или CHROME_PATH.
slides:
	@HS=$$(ls -d $$HOME/Library/Caches/ms-playwright/chromium_headless_shell-*/chrome-*/headless_shell $$HOME/.cache/ms-playwright/chromium_headless_shell-*/chrome-*/headless_shell 2>/dev/null | head -1); \
	CHROME_PATH="$${CHROME_PATH:-$$HS}" npx marp docs/presentation.md --pdf --allow-local-files -o docs/presentation.pdf
# Нагрузочный тест k6 против запущенного стека (make up && make seed). Итоги — в loadtest/result.json.
loadtest:
	set -a; . ./.env; set +a; docker run --rm --network "$${COMPOSE_PROJECT_NAME:-hackathon-starter}_default" \
	  -v "$$PWD/loadtest:/scripts" -e DEMO_EMAIL -e DEMO_PASSWORD -e LOAD_VUS -e LOAD_DURATION \
	  $(K6_IMAGE) run /scripts/k6.js
# Показать проект с нуля: стек, демо-данные и ссылки (одна команда для защиты).
demo: up seed
	@node scripts/demo-info.mjs
	@command -v open >/dev/null 2>&1 && open http://localhost:3000 || true
down:
	docker compose --profile app --profile llm down --remove-orphans
logs:
	docker compose --profile app logs -f --tail=100
gateway:
	set -a; . ./.env; set +a; cd server/gateway && go run ./cmd/gateway
tasks:
	set -a; . ./.env; set +a; export DATABASE_URL=$$TASKS_DATABASE_URL HTTP_ADDR=:8081; cd server/tasks && go run ./cmd/tasks
files:
	set -a; . ./.env; set +a; export DATABASE_URL=$$FILES_DATABASE_URL HTTP_ADDR=:8082; cd server/files && go run ./cmd/files
notifications:
	set -a; . ./.env; set +a; export DATABASE_URL=$$NOTIFICATIONS_DATABASE_URL HTTP_ADDR=:8084; cd server/notifications && go run ./cmd/notifications
audit:
	set -a; . ./.env; set +a; export DATABASE_URL=$$AUDIT_DATABASE_URL HTTP_ADDR=:8085; cd server/audit && go run ./cmd/audit
realtime:
	set -a; . ./.env; set +a; export HTTP_ADDR=:8086; cd server/realtime && go run ./cmd/realtime
events:
	set -a; . ./.env; set +a; export DATABASE_URL=$$EVENTS_DATABASE_URL HTTP_ADDR=:8087; cd server/events && go run ./cmd/events
llm-svc:
	set -a; . ./.env; set +a; export HTTP_ADDR=:8083; cd server/llm && go run ./cmd/llm
web:
	npm run dev
web-client:
	npm run dev:client
web-admin:
	npm run dev:admin
migrate:
	set -a; . ./.env; set +a; (cd server/tasks && DATABASE_URL=$$TASKS_DATABASE_URL go run ./cmd/tasks migrate up) && (cd server/files && DATABASE_URL=$$FILES_DATABASE_URL go run ./cmd/files migrate up) && (cd server/notifications && DATABASE_URL=$$NOTIFICATIONS_DATABASE_URL go run ./cmd/notifications migrate up) && (cd server/audit && DATABASE_URL=$$AUDIT_DATABASE_URL go run ./cmd/audit migrate up) && (cd server/events && DATABASE_URL=$$EVENTS_DATABASE_URL go run ./cmd/events migrate up)
migrate-status:
	set -a; . ./.env; set +a; (cd server/tasks && DATABASE_URL=$$TASKS_DATABASE_URL go run ./cmd/tasks migrate status) && (cd server/files && DATABASE_URL=$$FILES_DATABASE_URL go run ./cmd/files migrate status)
test:
	for s in gateway tasks files llm notifications audit realtime events; do (cd server/$$s && go test -race ./...) || exit 1; done
	npm test
check: test
	for s in gateway tasks files llm notifications audit realtime events; do (cd server/$$s && test -z "$$(gofmt -l .)" && go vet ./... && go build ./...) || exit 1; done
	npm run typecheck
	npm run lint
	npm run format:check
	npm run build
fmt:
	for s in gateway tasks files llm notifications audit realtime events; do (cd server/$$s && gofmt -w .) || exit 1; done
	npm run format
e2e:
	cd e2e && npm ci && npx playwright install chromium && npx playwright test
# Общий OpenAPI из спецификаций сервисов → TS-типы → копия для Swagger UI в gateway.
openapi:
	node scripts/openapi.mjs
	npx prettier --write packages/contracts/openapi.yaml server/*/api/openapi.yaml > /dev/null
	npm run gen -w @starter/contracts
	$(MAKE) docs-sync
docs-sync:
	cp packages/contracts/openapi.yaml server/gateway/internal/apidocs/openapi.yaml
seed:
	node scripts/seed.mjs
smoke:
	npm run smoke
integration: migrate
	set -a; . ./.env; set +a; (cd server/tasks && DATABASE_URL=$$TASKS_DATABASE_URL go test -race -tags=integration ./internal/tasks ./internal/broker ./internal/platform) && (cd server/files && DATABASE_URL=$$FILES_DATABASE_URL go test -race -tags=integration ./internal/files) && (cd server/notifications && DATABASE_URL=$$NOTIFICATIONS_DATABASE_URL go test -race -tags=integration ./internal/notifications) && (cd server/realtime && go test -race -tags=integration ./internal/realtime) && (cd server/events && DATABASE_URL=$$EVENTS_DATABASE_URL go test -race -tags=integration ./internal/events) && (cd server/gateway && go test -race -tags=integration ./internal/auth)
backup:
	@mkdir -p .local/backups
	for s in tasks files notifications audit events; do docker compose exec -T $$s-postgres sh -c 'exec pg_dump -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -Fc' > .local/backups/$$s-$$(date +%Y%m%d-%H%M%S).dump || exit 1; done
