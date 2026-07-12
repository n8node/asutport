#!/usr/bin/env bash
set -euo pipefail

ROOT="/opt/asutport"
DOMAIN="${DOMAIN:-asutport.ru}"

mkdir -p "${ROOT}/nginx/ssl"
cp "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" "${ROOT}/nginx/ssl/"
cp "/etc/letsencrypt/live/${DOMAIN}/privkey.pem" "${ROOT}/nginx/ssl/"

cd "${ROOT}"
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T nginx nginx -s reload
