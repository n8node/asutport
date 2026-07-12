# Промпт для Cursor — Фаза 1 (ASUTPORT)

Вставь в Cursor:

---

Прочитай все правила в `.cursor/rules/` — там 8 файлов проекта ASUTPORT (единый портал поддержки АСУ ТП):
- asutport-rules.mdc — архитектура, стек, S3, критические правила
- asutport-project.mdc — продукт, роли, кабинеты, биллинг
- asutport-agent.mdc — ИИ-контур (vision-пайплайн, агент, квалификация)
- asutport-git-deploy.mdc — git, деплой, SSL, бэкапы, Beget S3
- asutport-design-system.mdc — тёмная HMI-тема, токены, шрифты, терминология
- asutport-llm-provider-privacy.mdc — vendor-neutral copy
- asutport-product-directives.mdc — замороженный скоуп MVP
- asutport-backlog.mdc — отложенное

Также прочитай ROADMAP.md и design/asutport-prototype.html (визуальный источник истины).

Выполни **Фазу 1: Скелет и инфраструктура.** Создай ВСЁ за один проход:

**1. Backend (Go):**
- Go-модуль `github.com/n8node/asutport`
- Зависимости: chi, pgx/v5, goose, caarlos0/env, go-playground/validator, aws-sdk-go-v2 (s3 + presign)
- `cmd/server/main.go` — запуск HTTP, подключение Postgres, **goose-миграции при старте** (лог `migrations applied`)
- `internal/config/config.go` — env, включая S3_ENDPOINT/S3_REGION/S3_BUCKET/S3_ACCESS_KEY/S3_SECRET_KEY/S3_USE_PATH_STYLE
- `internal/s3/client.go` — S3-клиент (endpoint из env, path-style по флагу), `presign.go` (GET, TTL 1ч), `keys.go` (схема ключей из правил)
- `internal/server/server.go` — chi-роутер, middleware, маунт
- `internal/middleware/logging.go` (slog), `cors.go`
- `internal/handler/health.go` — `GET /health` → `{"status":"ok","version":"0.1.0","postgres":"ok","s3":"ok"}` (реальные проверки: ping БД, HeadBucket S3)
- `internal/repository/postgres.go` — pgx pool
- `migrations/001_init.up.sql` / `.down.sql` — users, organizations (type IN client_org/manufacturer/partner/integrator), org_members (role), api_keys — по схеме из asutport-rules.mdc
- `Dockerfile` (multi-stage, alpine), `Dockerfile.dev` (air), `.air.toml`

**2. Frontend (Next.js):**
- Next.js 15, App Router, TypeScript strict, Tailwind
- `next.config.ts`: **БЕЗ basePath**, `output: 'standalone'`
- Route-группы: `(public)/kb/page.tsx` (светлая, заглушка «База знаний»), `(dashboard)/dashboard/page.tsx`,
  `(vendor)/vendor/page.tsx`, `(admin)/admin/page.tsx` — тёмная HMI-тема
- Глобальные CSS-переменные из asutport-design-system.mdc (— bg #131619, panel #1B2025, acc #3FC8B7, лампы g/a/r)
- Шрифты через next/font/google: Unbounded (лого), Onest (UI), JetBrains Mono (данные) — кириллические сабсеты
- Компонент `<Lamp state="g|a|r|off">` с анимацией и prefers-reduced-motion
- Заглушки страниц: топбар с wordmark «ASUTPORT» + лампа-статус
- `Dockerfile`, `Dockerfile.dev`

**3. WordPress:** `wordpress/Dockerfile` (wordpress:6-php8.3-apache + wp-cli)

**4. Nginx:**
- `nginx/Dockerfile` (nginx:1.27-alpine), `nginx.conf` (worker_processes auto, gzip on)
- `conf.d/default.conf`:
  - `/api/` → backend:8080 (timeout 120s, body 100M — PDF документации!)
  - `/health` → backend
  - `/kb`, `/dashboard`, `/vendor`, `/admin` → frontend:3000 (websocket upgrade)
  - `/_next/` → frontend (cache 365d)
  - `/wp-admin/`, `/wp-content/`, `/wp-includes/`, `/wp-json/`, `/wp-login.php` → wordpress
  - `/` → wordpress
  - `conf.d/https.conf.sample` — SSL server block по deploy-правилам
- `nginx/ssl/.gitkeep`

**5. Docker Compose:**
- `docker-compose.yml` — nginx, backend, frontend, wordpress, postgres (pgvector:pg16), mysql, **minio + minio-init (создание бакета asutport)**
- `docker-compose.prod.yml` — replicas, limits, logging json-file 10MB×3; **БЕЗ minio** (prod = Beget S3); nats/dragonfly закомментированы
- Порты наружу ТОЛЬКО у nginx (80, 443). Healthcheck postgres, mysql, minio
- Volumes: postgres_data, mysql_data, wp_uploads, minio_data

**6. Инфраструктура:**
- `.env.example` — ВСЕ переменные (Postgres, MySQL/WP, OpenRouter с моделями QUALIFY/ANSWER/VISION/KB/EMBED, S3_*, JWT, `DOMAIN=asutport.ru`, CERTBOT_EMAIL)
- `.gitignore` — полный (Go, Node, Docker, .env, nginx/ssl, https.conf, uploads, backups, IDE, OS)
- `Makefile`: dev, prod, down, test, lint, logs, logs-prod, psql, wp-cli, deploy, backup-db, backup-wp, status, golden (заглушка)
- `scripts/`: deploy.sh, backup.sh, s3-sync.sh (заглушка с TODO), ssl-init.sh, ssl-deploy-hook.sh, setup.sh
- `backups/.gitkeep`, `design/.gitkeep` (положу прототип сам)

**7. Git:** init, первый коммит `feat: initial project structure (Phase 1)`, команды для remote и push.

**Критерий успеха:**
1. `cp .env.example .env` + пароли
2. `make dev` — все контейнеры стартуют
3. `curl http://localhost/health` → `{"status":"ok",...,"postgres":"ok","s3":"ok"}`
4. `http://localhost/dashboard` → тёмная HMI-страница с лампой
5. `http://localhost/kb` → светлая заглушка
6. `http://localhost/` → WordPress wizard
7. В MinIO-консоли виден бакет `asutport`

Не пропускай ни один файл. После завершения покажи список созданных файлов, команды запуска, push и первого деплоя.
