#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BRANCH="${1:-main}"
PROJECT_DIR="/opt/asutport"

echo "=== Deploy ASUTPORT branch ${BRANCH} on server ==="
echo "Run on server (already in SSH session):"
cat <<EOF
cd ${PROJECT_DIR}
git fetch origin && git checkout ${BRANCH} && git pull origin ${BRANCH}
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T nginx nginx -t
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T nginx nginx -s reload
curl -sf http://127.0.0.1/health
EOF
