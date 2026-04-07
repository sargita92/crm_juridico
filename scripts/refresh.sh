#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# refresh.sh — reseta o banco e reinicia a aplicação
#
# Uso:
#   ./scripts/refresh.sh                # drop + recreate + migrate + fixtures + restart app
#   ./scripts/refresh.sh --no-fixtures  # sem fixtures
#   ./scripts/refresh.sh --no-restart   # sem restart da app
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Carrega variáveis do .env se existir
if [ -f "$PROJECT_ROOT/.env" ]; then
    set -a
    source "$PROJECT_ROOT/.env"
    set +a
fi

# Para conexão de fora do container (host), usamos 127.0.0.1 e a porta mapeada
DB_DOCKER_PORT="${DATABASE_PORT:-3307}"
DB_USER="${DATABASE_USER:-crm}"
DB_PASS="${DATABASE_PASSWORD:-crm_secret}"
DB_NAME="${DATABASE_NAME:-crm_juridico}"
DB_ROOT_PASS="${DATABASE_ROOT_PASSWORD:-root_secret}"
COMPOSE_FILE="$PROJECT_ROOT/docker-compose.dev.yml"

LOAD_FIXTURES=true
RESTART_APP=true

for arg in "$@"; do
    case $arg in
        --no-fixtures) LOAD_FIXTURES=false ;;
        --no-restart)  RESTART_APP=false ;;
        --help|-h)
            echo "Uso: $0 [--no-fixtures] [--no-restart]"
            echo ""
            echo "  --no-fixtures   Não carrega fixture/fixtures.sql"
            echo "  --no-restart    Não reinicia o container da app"
            exit 0
            ;;
        *)
            echo "Opção desconhecida: $arg"
            echo "Use $0 --help para ver opções."
            exit 1
            ;;
    esac
done

mysql_root() {
    docker compose -f "$COMPOSE_FILE" exec -T mysql mysql -u root -p"${DB_ROOT_PASS}" "$@" 2>/dev/null
}

mysql_user() {
    docker compose -f "$COMPOSE_FILE" exec -T mysql mysql -u "${DB_USER}" -p"${DB_PASS}" "${DB_NAME}" 2>/dev/null
}

echo "==> Verificando se MySQL está rodando..."
if ! docker compose -f "$COMPOSE_FILE" ps mysql --format '{{.State}}' 2>/dev/null | grep -q running; then
    echo "    MySQL não está rodando. Subindo containers..."
    docker compose -f "$COMPOSE_FILE" up -d mysql
fi

echo "==> Aguardando MySQL ficar pronto..."
for i in $(seq 1 30); do
    if docker compose -f "$COMPOSE_FILE" exec -T mysql mysqladmin ping -h localhost -u root -p"${DB_ROOT_PASS}" --silent 2>/dev/null; then
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "    ERRO: MySQL não ficou pronto após 60s"
        exit 1
    fi
    echo "    tentativa $i/30..."
    sleep 2
done

echo "==> Dropando e recriando banco '${DB_NAME}'..."
mysql_root -e "
    DROP DATABASE IF EXISTS ${DB_NAME};
    CREATE DATABASE ${DB_NAME} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
    GRANT ALL ON ${DB_NAME}.* TO '${DB_USER}'@'%';
    FLUSH PRIVILEGES;
"

echo "==> Rodando migrations via golang-migrate..."
MIGRATE_URL="mysql://${DB_USER}:${DB_PASS}@tcp(127.0.0.1:${DB_DOCKER_PORT})/${DB_NAME}?multiStatements=true"

if command -v migrate &>/dev/null; then
    migrate -path "$PROJECT_ROOT/migrations" -database "${MIGRATE_URL}" up
elif [ -f "$(go env GOPATH)/bin/migrate" ]; then
    "$(go env GOPATH)/bin/migrate" -path "$PROJECT_ROOT/migrations" -database "${MIGRATE_URL}" up
else
    echo "    ERRO: golang-migrate CLI não encontrado."
    echo "    Instale com: go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    exit 1
fi

if [ "$LOAD_FIXTURES" = true ]; then
    echo "==> Carregando fixtures..."
    mysql_user < "$PROJECT_ROOT/fixture/fixtures.sql"
fi

if [ "$RESTART_APP" = true ]; then
    echo "==> Reiniciando app..."
    docker compose -f "$COMPOSE_FILE" restart app
    sleep 5
    if docker compose -f "$COMPOSE_FILE" logs app --tail 3 2>&1 | grep -q "server started\|listening"; then
        echo "    App rodando!"
    else
        echo "    App reiniciada. Verifique logs: docker compose -f $COMPOSE_FILE logs app --tail 20"
    fi
fi

echo ""
echo "==> Pronto! Banco resetado com dados iniciais."
echo "    Login: admin@teste.com / admin123"
echo "    URL:   http://localhost:${SERVER_PORT:-8533}/login"
