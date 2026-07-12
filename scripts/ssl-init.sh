#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

set -a
# shellcheck disable=SC1091
source .env
set +a

DOMAIN="${DOMAIN:-asutport.ru}"
EMAIL="${CERTBOT_EMAIL:-erman.ai@yandex.ru}"
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"

echo "=== Stopping nginx for ACME standalone ==="
$COMPOSE stop nginx || true

certbot certonly --standalone \
  --non-interactive \
  --agree-tos \
  --email "$EMAIL" \
  -d "$DOMAIN" \
  -d "www.$DOMAIN"

mkdir -p nginx/ssl
cp "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" nginx/ssl/
cp "/etc/letsencrypt/live/${DOMAIN}/privkey.pem" nginx/ssl/

cp nginx/conf.d/https.conf.sample nginx/conf.d/https.conf
sed -i "s/DOMAIN_PLACEHOLDER/${DOMAIN}/g" nginx/conf.d/https.conf

$COMPOSE up -d --build nginx
echo "SSL initialized for ${DOMAIN}"
