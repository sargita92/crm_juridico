#!/usr/bin/env bash
set -euo pipefail

# Carrega variáveis do .env se existir
if [ -f .env ]; then
    set -a
    source .env
    set +a
fi

DB_HOST="${DATABASE_HOST:-127.0.0.1}"
DB_PORT="${DATABASE_PORT:-3307}"
DB_USER="${DATABASE_USER:-crm}"
DB_PASS="${DATABASE_PASSWORD:-crm_secret}"
DB_NAME="${DATABASE_NAME:-crm_juridico}"

echo "==> Aguardando MySQL ficar pronto..."
for i in $(seq 1 30); do
    if docker compose -f docker-compose.dev.yml exec -T mysql mysqladmin ping -h localhost -u root -p"${DATABASE_ROOT_PASSWORD:-root_secret}" --silent 2>/dev/null; then
        break
    fi
    echo "    tentativa $i/30..."
    sleep 2
done

echo "==> Dropando e recriando banco..."
docker compose -f docker-compose.dev.yml exec -T mysql mysql -u root -p"${DATABASE_ROOT_PASSWORD:-root_secret}" -e "
    DROP DATABASE IF EXISTS ${DB_NAME};
    CREATE DATABASE ${DB_NAME} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
    GRANT ALL ON ${DB_NAME}.* TO '${DB_USER}'@'%';
    FLUSH PRIVILEGES;
"

echo "==> Rodando migrations..."
docker compose -f docker-compose.dev.yml exec -T mysql mysql -u "${DB_USER}" -p"${DB_PASS}" "${DB_NAME}" < migrations/000001_init.up.sql 2>/dev/null || true
docker compose -f docker-compose.dev.yml exec -T mysql mysql -u "${DB_USER}" -p"${DB_PASS}" "${DB_NAME}" < migrations/000002_create_tenants_table.up.sql
docker compose -f docker-compose.dev.yml exec -T mysql mysql -u "${DB_USER}" -p"${DB_PASS}" "${DB_NAME}" < migrations/000003_create_users_table.up.sql
docker compose -f docker-compose.dev.yml exec -T mysql mysql -u "${DB_USER}" -p"${DB_PASS}" "${DB_NAME}" < migrations/000004_create_user_tenants_table.up.sql

echo "==> Carregando fixtures..."
docker compose -f docker-compose.dev.yml exec -T mysql mysql -u "${DB_USER}" -p"${DB_PASS}" "${DB_NAME}" < fixture/fixtures.sql

echo ""
echo "==> Pronto! Banco resetado com dados iniciais."
echo "    Login: admin@teste.com / admin123"
echo "    URL:   http://localhost:${SERVER_PORT:-8533}/login"
