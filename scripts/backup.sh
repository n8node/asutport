#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p backups
RETENTION_DAYS=30

COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"
TS="$(date +%Y%m%d_%H%M%S)"

set -a
# shellcheck disable=SC1091
source .env
set +a

$COMPOSE exec -T postgres pg_dump -U "${POSTGRES_USER}" "${POSTGRES_DB}" \
  | gzip > "backups/pg_${TS}.sql.gz"

$COMPOSE exec -T mysql mysqldump -u "${WP_DB_USER}" -p"${WP_DB_PASSWORD}" "${WP_DB_NAME}" \
  | gzip > "backups/wp_${TS}.sql.gz"

find backups -name '*.sql.gz' -mtime +"${RETENTION_DAYS}" -delete
echo "Backups written to backups/ (retention ${RETENTION_DAYS} days)"
