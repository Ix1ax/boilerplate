# CI/CD: от нового репозитория до работающего домена

Пошаговая инструкция «проект залит в GitHub — что дальше». Здесь только действия: что сделать на сервере, что в настройках GitHub, куда вписать какие значения и как привязать домен. Устройство серверного скрипта, бэкапы и восстановление — в [deployment.md](deployment.md).

Как это работает:

```text
push в main → GitHub Actions (.github/workflows/deploy.yml)
            → архив коммита по SSH на VPS
            → deploy/deploy.sh: docker compose build + migrate + up
            → проверка https://APP_DOMAIN/readyz → версия считается успешной
```

GitHub-раннер **не** собирает образы и ничего не хранит: он только передаёт исходники нужного коммита и запускает скрипт. Вся сборка идёт на вашем VPS.

## 0. Что уже в репозитории — писать не нужно

| Файл                           | Назначение                                                            |
| ------------------------------ | --------------------------------------------------------------------- |
| `.github/workflows/deploy.yml` | Деплой при push в `main` и по кнопке Run workflow                     |
| `deploy/deploy.sh`             | Всё, что выполняется на сервере: сборка, миграции, бэкапы, проверки   |
| `compose.prod.yaml`            | Серверное наложение: порты наружу только у Caddy, `restart` политики  |
| `deploy/Caddyfile.example`     | HTTPS, сертификаты Let's Encrypt, проксирование приложения и Keycloak |
| `scripts/setup.mjs`            | Генерация серверных секретов в `.env` (запускается в контейнере)      |

Остаётся подготовить VPS, DNS и вписать 4 значения в настройки GitHub.

## 1. Что понадобится

- **VPS с Linux** (проверялось на Ubuntu 22.04/24.04): от 8 ГБ RAM и от 40 ГБ диска. Стек — это 12 образов приложения, 8 баз PostgreSQL, Redis, RabbitMQ, S3, Keycloak, Mailpit и Ollama с Qwen. На 2–4 ГБ сборка упадёт по памяти.
- **Домен** и доступ к его DNS-записям. Нужны **два разных имени** (приложение и Keycloak) — так требует `scripts/setup.mjs`.
- Доступ к репозиторию на GitHub с правом открыть **Settings** (владелец или admin).

## 2. Домен и DNS — делается первым

Caddy выпускает сертификат при первом запуске, а для этого Let's Encrypt должен попасть на ваш сервер по имени. Если DNS ещё не распространился, деплой упадёт на проверке HTTPS. Поэтому записи создаём заранее.

Создайте две A-записи на IP вашего VPS:

| Тип | Имя    | Значение       | Что это                   |
| --- | ------ | -------------- | ------------------------- |
| A   | `app`  | `203.0.113.10` | Приложение (`APP_DOMAIN`) |
| A   | `auth` | `203.0.113.10` | Keycloak (`AUTH_DOMAIN`)  |

Имена любые, главное — **разные**: `app.example.org` и `auth.example.org`, либо корень `example.org` (имя `@`) для приложения и `auth.example.org` для входа.

Проверьте с вашего компьютера, что имена уже ведут на VPS:

```bash
dig +short app.example.org auth.example.org
```

Важные частности:

- **AAAA-записи.** Если они есть, они тоже должны указывать на этот VPS (иначе часть клиентов и сам Let's Encrypt придут по IPv6 в пустоту). Нет IPv6 на сервере — удалите AAAA.
- **Cloudflare и подобные прокси.** Для первого выпуска сертификата поставьте режим **DNS only** (серая тучка). Оранжевый прокси подменяет TLS, и проверка `https://APP_DOMAIN/healthz` из скрипта деплоя пойдёт не к Caddy.
- **TTL** перед первым деплоем лучше поставить 300 секунд: проще исправить опечатку.

## 3. Сервер: подготовка (один раз)

Зайдите на VPS по SSH под root.

```bash
apt-get update && apt-get install -y curl tar util-linux ca-certificates
curl -fsSL https://get.docker.com | sh
docker compose version
```

`docker compose version` должен показать **не ниже 2.24.4**: серверное наложение использует синтаксис `!override` и `!reset`. Версия ниже — обновите Docker, иначе `compose config` упадёт.

Отдельный пользователь для деплоя (вместо root) и каталог приложения:

```bash
adduser --disabled-password --gecos '' deploy
usermod -aG docker deploy
install -d -o deploy -g deploy -m 755 /srv/boilerplate
```

Строка `usermod -aG docker deploy` даёт пользователю полный доступ к Docker, то есть фактически права root на этой машине. Это требование деплоя; не выдавайте этого пользователя никому лишнему.

Файрвол — наружу только SSH и веб:

```bash
ufw allow 22/tcp && ufw allow 80/tcp && ufw allow 443/tcp && ufw allow 443/udp
ufw --force enable
```

Порты баз, RabbitMQ, S3, Ollama и Keycloak наружу не открывайте: `compose.prod.yaml` их и так убирает, публичные порты есть только у Caddy (80/443, UDP 443 для HTTP/3).

## 4. Серверный `.env`

Все серверные секреты живут в `/srv/boilerplate/.env` и создаются **на сервере**. Локальный `.env` разработчика и каталог `.local` на VPS не переносите.

Проще всего дать первому деплою создать файл за вас: он скопирует `.env.example`, остановится и напишет, что заполнить. После этого на сервере:

```bash
sudo -u deploy nano /srv/boilerplate/.env
```

Заполните три строки:

```ini
APP_DOMAIN=app.example.org
AUTH_DOMAIN=auth.example.org
COMPOSE_PROJECT_NAME=hackathon-starter
```

- Домены — **без** `https://` и без пути, и обязательно разные.
- `COMPOSE_PROJECT_NAME` выберите один раз и больше не меняйте: это префикс имён томов. Смена имени — это другие тома, то есть пустые базы. Скрипт запоминает его в `/srv/boilerplate/.project-name` и останавливает деплой при расхождении.
- Пароли и токены (`*_DB_PASSWORD`, `INTERNAL_TOKEN`, `METRICS_TOKEN`, `MAILPIT_PASSWORD`…) оставьте пустыми: `scripts/setup.mjs --production` сгенерирует отсутствующие и не тронет уже заданные.

Права на файл — только владельцу:

```bash
chmod 600 /srv/boilerplate/.env
```

## 5. Вход по SSH: пароль или ключ

Деплою нужен способ зайти на сервер. Workflow принимает **ровно один** из двух секретов: `DEPLOY_PASSWORD` или `DEPLOY_SSH_KEY`. Если задать оба или ни одного — запуск остановится с понятной ошибкой.

В обоих вариантах нужен `DEPLOY_KNOWN_HOSTS` — отпечаток ключа сервера. Без него нельзя отличить ваш VPS от чужого, который перехватил соединение; при входе по паролю это особенно важно, потому что пароль уходит на ту сторону.

### Вариант А. Логин и пароль

1. Задайте пользователю пароль (длинный, 20+ символов, только для деплоя):

   ```bash
   passwd deploy
   ```

2. Разрешите вход по паролю в SSH. Проверьте, что в `/etc/ssh/sshd_config` (и в файлах `/etc/ssh/sshd_config.d/*`, у многих провайдеров настройка лежит там) стоит:

   ```ini
   PasswordAuthentication yes
   ```

   Затем `systemctl restart ssh` (на некоторых системах `sshd`). Многие хостеры по умолчанию ставят `PasswordAuthentication no` — тогда деплой получит `Permission denied`.

3. Снимите отпечаток сервера — со своего компьютера:

   ```bash
   ssh-keyscan -p 22 app.example.org
   ```

   Весь вывод (все строки) пойдёт в секрет `DEPLOY_KNOWN_HOSTS`. Сверьте отпечаток с тем, что показывает сам сервер в консоли провайдера:

   ```bash
   ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub
   ```

4. Пароль пользователя впишите в секрет `DEPLOY_PASSWORD`.

Что стоит знать про этот вариант: пароль хранится в секретах GitHub, а открытый `PasswordAuthentication` круглосуточно перебирают боты. Если оставляете его — поставьте `fail2ban` (`apt-get install -y fail2ban`), держите `PermitRootLogin no` и не используйте этот пароль больше нигде. Вариант рабочий, просто требует этой гигиены.

### Вариант Б. Ключ (надёжнее, три команды)

На своём компьютере:

```bash
ssh-keygen -t ed25519 -f ~/deploy_key -N '' -C 'github-actions'
ssh-copy-id -i ~/deploy_key.pub deploy@app.example.org
ssh-keyscan -p 22 app.example.org
```

- `DEPLOY_SSH_KEY` — содержимое **приватного** файла `~/deploy_key` целиком, вместе со строками `-----BEGIN OPENSSH PRIVATE KEY-----` и `-----END OPENSSH PRIVATE KEY-----`. Ключ создавайте без парольной фразы (`-N ''`): раннер не сможет её ввести.
- `DEPLOY_KNOWN_HOSTS` — вывод `ssh-keyscan`.
- Файл `~/deploy_key.pub` остаётся в `authorized_keys` на сервере; приватный после копирования в секрет можно удалить со своей машины.

## 6. Куда вводить данные в GitHub

Откройте репозиторий → **Settings** → в левом меню **Secrets and variables** → **Actions**. Там две вкладки: **Secrets** (кнопка _New repository secret_) и **Variables** (_New repository variable_). Добавляйте именно **repository**-секреты, не _Environment secrets_: workflow читает `secrets.*` без окружения.

Секреты — вкладка **Secrets**:

| Имя                  | Что вписать                                                      | Пример                                  |
| -------------------- | ---------------------------------------------------------------- | --------------------------------------- |
| `DEPLOY_HOST`        | Домен или IPv4 вашего VPS                                        | `app.example.org`                       |
| `DEPLOY_USER`        | Пользователь SSH с доступом к Docker                             | `deploy`                                |
| `DEPLOY_KNOWN_HOSTS` | Весь вывод `ssh-keyscan` (можно несколько строк)                 | `app.example.org ssh-ed25519 AAAAC3Nz…` |
| `DEPLOY_PASSWORD`    | Пароль пользователя — **вариант А**                              | `…`                                     |
| `DEPLOY_SSH_KEY`     | Приватный ключ целиком — **вариант Б**, вместо `DEPLOY_PASSWORD` | `-----BEGIN OPENSSH PRIVATE KEY-----…`  |

Переменные — вкладка **Variables**, обе необязательные:

| Имя           | По умолчанию       | Когда менять                                                              |
| ------------- | ------------------ | ------------------------------------------------------------------------- |
| `DEPLOY_PORT` | `22`               | Нестандартный порт SSH. Тогда в `known_hosts` нужен формат `[хост]:порт`  |
| `DEPLOY_PATH` | `/srv/boilerplate` | Другой каталог. Абсолютный путь без пробелов, пользователь — его владелец |

Значение секрета после сохранения посмотреть нельзя — только перезаписать. В логах Actions секреты маскируются; сам скрипт деплоя тоже не печатает содержимое `.env` и сообщений пользователей.

## 7. Первый деплой

Запуск происходит при **push в `main`** (включая merge pull request) либо вручную: **Actions** → workflow **Деплой** → **Run workflow**. Ручной запуск удобен как раз для повторной попытки после правки `.env`.

Что увидите:

1. Если `.env` на сервере ещё нет — деплой создаст его из шаблона и остановится с сообщением «задайте APP_DOMAIN и AUTH_DOMAIN». Это ожидаемо: заполните файл (шаг 4) и запустите снова.
2. Второй запуск собирает образы по очереди (так меньше пиковая память), поднимает базы, делает `pg_dump` всех баз, прогоняет миграции, скачивает модель Qwen и ждёт готовности контейнеров.
3. В конце — публичные проверки `https://APP_DOMAIN/healthz`, `/readyz` и OIDC-discovery Keycloak. Только после них ссылка `current` переключается на новую версию.

Первый деплой идёт долго: двенадцать сборок плюс несколько гигабайт модели. Закладывайте 30–60 минут и больше на слабом VPS; лимит задачи — 90 минут. Следующие деплои короче за счёт кешей Docker.

Деплои не идут параллельно: `concurrency` в workflow ставит их в очередь, а `flock` на сервере страхует от второго запуска извне.

## 8. Проверка после деплоя

```bash
curl -sS https://app.example.org/healthz
curl -sS https://app.example.org/readyz
curl -sS https://auth.example.org/realms/hackathon/.well-known/openid-configuration | head -c 120
```

- Приложение — `https://app.example.org`, админка — `https://app.example.org/admin/`.
- Тестовая почта (письма регистрации и сброса пароля) — `https://app.example.org/mail/`, логин и пароль из `.env`:

  ```bash
  sudo sed -n 's/^MAILPIT_\(USER\|PASSWORD\)=/\1=/p' /srv/boilerplate/.env
  ```

- Админка Keycloak — `https://auth.example.org/admin/`, логин `KEYCLOAK_ADMIN` и пароль `KEYCLOAK_ADMIN_PASSWORD` из того же файла.
- Production-realm создаётся **без пользователей**. Администраторов в Keycloak нет: у админки приложения свой вход (`https://APP_DOMAIN/admin/`). Суперадминистратора сервис `identity` создаёт при первом старте из `SUPERADMIN_EMAIL` и `SUPERADMIN_PASSWORD` в серверном `.env` (пароль генерирует `make setup`). Остальных админов он заводит в админке.

Состояние контейнеров на сервере:

```bash
cd /srv/boilerplate/current && docker compose \
  --env-file /srv/boilerplate/.env -f compose.yaml -f compose.prod.yaml \
  --profile app --profile llm ps
```

## 9. Сводка: что куда вписывается

| Значение                    | Где живёт                   | Кто задаёт                     |
| --------------------------- | --------------------------- | ------------------------------ |
| A-записи `app` и `auth`     | DNS-панель домена           | вы, вручную                    |
| `APP_DOMAIN`, `AUTH_DOMAIN` | `/srv/boilerplate/.env`     | вы, вручную                    |
| `COMPOSE_PROJECT_NAME`      | `/srv/boilerplate/.env`     | вы, один раз и навсегда        |
| Пароли БД, токены, Mailpit  | `/srv/boilerplate/.env`     | `scripts/setup.mjs` при деплое |
| `DEPLOY_*`                  | GitHub → Settings → Actions | вы, вручную                    |
| HTTPS-сертификаты           | том `caddy-data` на сервере | Caddy, автоматически           |

Домены задаются **дважды и по-разному**: в DNS (куда идёт трафик) и в `.env` (что приложение считает своим адресом — `APP_ORIGIN`, `OIDC_ISSUER`, hostname Keycloak). Поменяли домен — правьте оба места.

## 10. Типичные ошибки

| Что видно                                            | Причина и что делать                                                                                                                                    |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Не задан секрет DEPLOY_HOST`                        | Секрет не добавлен или добавлен как Environment secret. Нужен repository secret во вкладке Actions                                                      |
| `Задайте DEPLOY_SSH_KEY … или DEPLOY_PASSWORD`       | Не задан ни один способ входа, либо заданы оба — удалите лишний секрет                                                                                  |
| `Permission denied (publickey,password)`             | Неверный пользователь или пароль; либо на сервере `PasswordAuthentication no`; либо ключ не попал в `authorized_keys`                                   |
| `Host key verification failed`                       | `DEPLOY_KNOWN_HOSTS` пустой, от другого хоста, или сервер пересоздан. Снимите `ssh-keyscan` заново; для нестандартного порта нужен формат `[хост]:порт` |
| `задайте APP_DOMAIN и AUTH_DOMAIN`                   | Нормальный результат первого запуска — заполните `.env` и запустите снова                                                                               |
| `не установлен flock` / `curl` / `tar`               | `apt-get install -y util-linux curl tar` на сервере                                                                                                     |
| `COMPOSE_PROJECT_NAME изменён`                       | Верните прежнее значение из `/srv/boilerplate/.project-name`, иначе получите пустые базы                                                                |
| Сборка падает с `killed` / кодом 137                 | Не хватает памяти. Добавьте RAM или swap (`fallocate -l 4G /swapfile && mkswap /swapfile && swapon /swapfile`)                                          |
| `curl … https://app.example.org/healthz` не проходит | DNS не ведёт на VPS, закрыты 80/443, включён прокси Cloudflare, или сертификат не выпущен. Смотрите логи Caddy: `docker logs hackathon-starter-caddy-1` |
| После входа Keycloak пишет про `redirect_uri`        | Realm уже существует, а импорт существующий realm не обновляет. Поправьте redirect URI, web origins и logout URI в админке Keycloak вручную             |
| Пользователей выбросило из системы после деплоя      | Пересоздан Redis — сессии там временные. Нужно войти заново                                                                                             |

Подробный лог самого деплоя остаётся на сервере: `docker compose … logs <сервис>` из каталога `current`.

## 11. Чего этот CI/CD не делает

- **Не запускает тесты.** `make check`, `make integration`, `make smoke`, `make e2e` — локальные команды. Workflow только разворачивает стек и проверяет, что он отвечает. Если в branch protection остались обязательные статусы от прежнего CI — снимите их в настройках GitHub, иначе merge будет заблокирован.
- **Нет автоматического rollback.** При сбое часть изменений может уже примениться. Для ручного возврата есть каталоги `releases/`, образы с тегом SHA и дампы в `backups/` — порядок описан в [deployment.md](deployment.md#резервные-копии-и-восстановление).
- **Нет staging.** Один сервер, одна ветка `main`. Нужна отдельная среда — второй VPS, второй набор доменов и второй набор секретов (например, через GitHub Environments).
- **Бэкапы — только базы и только перед миграциями.** `.env`, S3-файлы, тома брокера и сертификаты в них не входят; вывозите их с VPS отдельно.
- **Обновление не бесшовное.** Короткий перерыв при пересоздании контейнеров возможен.
