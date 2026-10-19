#!/usr/bin/env bash
# Отдельные каталоги версий не смешивают исходники, секреты и постоянные данные.
set -Eeuo pipefail
umask 077

fail() { echo "Ошибка деплоя: $*" >&2; exit 1; }
[[ $# == 3 ]] || fail 'ожидаются каталог сервера, SHA и архив исходников'
deploy_root=$1
revision=$2
archive=$3
[[ "$deploy_root" =~ ^/[a-zA-Z0-9_/-]+$ && "$deploy_root" != / ]] || fail 'некорректный каталог сервера'
[[ "$revision" =~ ^[a-f0-9]{40}$ ]] || fail 'некорректный SHA коммита'
[[ -f "$archive" ]] || fail 'архив исходников не найден'
for command in docker flock curl tar; do command -v "$command" >/dev/null || fail "не установлен $command"; done
mkdir -p "$deploy_root/releases" "$deploy_root/shared" "$deploy_root/backups"
deploy_root=$(cd "$deploy_root" && pwd -P)
[[ "$deploy_root" != / ]] || fail 'каталог деплоя не может быть корнем файловой системы'
exec 9>"$deploy_root/.deploy.lock"
flock -w 3600 9 || fail 'предыдущий деплой не освободил блокировку за час'
docker info >/dev/null
docker compose version >/dev/null

release=$(mktemp -d "$deploy_root/releases/$revision.XXXXXX")
tar -xzf "$archive" -C "$release"
[[ -f "$release/compose.prod.yaml" ]] || fail 'в архиве нет серверной конфигурации'
if [[ ! -f "$deploy_root/.env" ]]; then
  install -m 600 "$release/.env.example" "$deploy_root/.env"
  fail "задайте APP_DOMAIN и AUTH_DOMAIN в $deploy_root/.env и повторите деплой"
fi
ln -s "$deploy_root/.env" "$release/.env"
ln -s "$deploy_root/shared" "$release/.local"
cd "$release"
# Node нужен только для подготовки настроек; устанавливать его на VPS не требуется.
docker run --rm --user "$(id -u):$(id -g)" \
  -v "$deploy_root:$deploy_root" -w "$release" node:24-alpine \
  node scripts/setup.mjs --production

project_name=$(sed -n 's/^COMPOSE_PROJECT_NAME=//p' "$deploy_root/.env")
[[ "$project_name" =~ ^[a-z0-9][a-z0-9_-]*$ ]] || fail 'некорректный COMPOSE_PROJECT_NAME'
if [[ -f "$deploy_root/.project-name" ]]; then
  [[ "$(cat "$deploy_root/.project-name")" == "$project_name" ]] || fail 'COMPOSE_PROJECT_NAME изменён: это создаст другие тома с пустыми базами'
else
  printf '%s\n' "$project_name" > "$deploy_root/.project-name"
fi
export DEPLOY_SHARED_DIR="$deploy_root/shared" DEPLOY_REVISION="$revision"
export COMPOSE_PARALLEL_LIMIT=1
compose=(docker compose --project-name "$project_name" --env-file "$deploy_root/.env" -f compose.yaml -f compose.prod.yaml --profile app --profile llm)
diagnose() {
  result=$?
  trap - ERR
  echo "Деплой $revision завершился ошибкой; версия: $release" >&2
  "${compose[@]}" ps -a >&2 || true
  # Полные логи доступны на сервере; в Actions не выводим содержимое настроек и сообщений пользователей.
  exit "$result"
}
trap diagnose ERR
"${compose[@]}" config -q
"${compose[@]}" pull --ignore-buildable
# На небольшом VPS параллельные Go-сборки могут исчерпать память.
for service in tasks-service files-service notifications-service audit-service events-service moderation-service realtime-service llm-service gateway-service web-service; do
  "${compose[@]}" build "$service"
done

"${compose[@]}" up -d --wait --wait-timeout 600 \
  tasks-postgres files-postgres notifications-postgres audit-postgres events-postgres moderation-postgres keycloak-postgres \
  gateway-redis files-s3 rabbitmq mailpit llm-ollama

backup="$deploy_root/backups/$(date -u +%Y%m%dT%H%M%SZ)-$revision"
mkdir -p "$backup"
for service in tasks files notifications audit events moderation keycloak; do
  "${compose[@]}" exec -T "$service-postgres" sh -c \
    'exec pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' > "$backup/$service.dump"
done
# Повторяем one-shot контейнеры на каждом деплое, даже если прошлый завершился успешно.
run_once() {
  local service=$1 container exit_code
  "${compose[@]}" up -d --no-deps --no-build --pull never --force-recreate "$service"
  # --all учитывает и быстрые миграции, успевшие завершиться до начала ожидания.
  container=$("${compose[@]}" ps --all --quiet "$service")
  [[ -n "$container" ]] || fail "не найден контейнер $service"
  exit_code=$(docker wait "$container")
  [[ "$exit_code" == 0 ]] || fail "$service завершился с кодом $exit_code"
}
for service in tasks files notifications audit events moderation; do
  run_once "$service-migrate"
done
run_once llm-ollama-pull
"${compose[@]}" up -d --no-build --pull never --wait --wait-timeout 900
# Nginx разрешает IP gateway при старте; после обновления gateway нужен новый web-контейнер.
"${compose[@]}" up -d --no-deps --no-build --pull never --force-recreate --wait --wait-timeout 120 web-service
"${compose[@]}" exec -T web-service wget -q -O /dev/null http://gateway-service:8080/readyz

app_domain=$(sed -n 's/^APP_DOMAIN=//p' "$deploy_root/.env")
auth_domain=$(sed -n 's/^AUTH_DOMAIN=//p' "$deploy_root/.env")
for url in "https://$app_domain/healthz" "https://$app_domain/readyz" "https://$auth_domain/realms/hackathon/.well-known/openid-configuration"; do
  curl --fail --silent --show-error --retry 12 --retry-delay 5 --retry-all-errors \
    --connect-timeout 10 --max-time 20 "$url" -o /dev/null
done
printf '%s\n' "$revision" > .revision
ln -sfn "$release" "$deploy_root/.current-next"
mv -Tf "$deploy_root/.current-next" "$deploy_root/current"
echo "Деплой $revision завершён: https://$app_domain; резервные копии БД: $backup"
