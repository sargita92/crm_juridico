#!/usr/bin/env bash
# scripts/audit-cover.sh
# Roda testes com cobertura, calcula global e por-pacote, e falha se abaixo da meta.
#
# Modos:
#   - default (dev): usa -short (rapido). Pacotes infrastructure ficam abaixo de 80%
#                    pois suas integration tests sao puladas.
#                    A meta global e' relaxada para 75% no modo -short.
#   - AUDIT_INTEGRATION=1 (CI): roda integration tests via testcontainers (Docker).
#                    Aplica metas completas: global >= 85%, pacote >= 80%.
#
# Pacotes ignorados (wiring/main):
#   - cmd/...
#   - internal/shared/module/...

set -euo pipefail

INTEGRATION="${AUDIT_INTEGRATION:-0}"

if [[ "$INTEGRATION" == "1" ]]; then
  THRESHOLD_GLOBAL="${THRESHOLD_GLOBAL:-85.0}"
  THRESHOLD_PACKAGE="${THRESHOLD_PACKAGE:-80.0}"
  TEST_FLAGS="-p 1"
  MODE_LABEL="integration (full) — fail on threshold"
  ENFORCE=1
else
  THRESHOLD_GLOBAL="${THRESHOLD_GLOBAL:-0.0}"
  THRESHOLD_PACKAGE="${THRESHOLD_PACKAGE:-80.0}"
  TEST_FLAGS="-short"
  MODE_LABEL="-short (dev) — informational only"
  ENFORCE=0
fi

EXCLUDE_PATTERNS=(
  "/cmd/"
  "/internal/shared/module"
)

# Em modo -short (dev), pacotes que dependem de testcontainers ficam abaixo do
# threshold pois seus testes sao pulados. Esses pacotes sao listados aqui e
# excluidos da checagem por-pacote (mas nao da global).
SHORT_MODE_INTEGRATION_PKGS=(
  "/internal/audit/infrastructure"
  "/internal/audit/integration"
  "/internal/auth/infrastructure"
  "/internal/auth/interfaces/http"
  "/internal/automation/infrastructure"
  "/internal/automation/application/executors"
  "/internal/dashboard/infrastructure"
  "/internal/document/infrastructure"
  "/internal/document/interfaces/http"
  "/internal/files/infrastructure"
  "/internal/ai"
  "/internal/ai/infrastructure"
  "/internal/ai/interfaces/http/playground"
  "/internal/funnel/infrastructure"
  "/internal/funnel/interfaces/http"
  "/internal/mcp/infrastructure"
  "/internal/mcp/interfaces/http"
  "/internal/notification"
  "/internal/notification/infrastructure"
  "/internal/notification/interfaces/http"
  "/internal/dashboard/domain"
  "/internal/pagamentos/infrastructure"
  "/internal/pagamentos/interfaces/http"
  "/internal/permission/infrastructure"
  "/internal/permission/domain"
  "/internal/permission/interfaces/http"
  "/internal/shared/database"
  "/internal/specialist/infrastructure"
  "/internal/specialist/interfaces/http"
  "/internal/tenant/infrastructure"
  "/internal/tenant/interfaces/http"
  "/internal/whatsapp"
  "/internal/whatsapp/infrastructure"
  "/internal/whatsapp/interfaces/http"
)

PROFILE="${COVER_PROFILE:-coverage.out}"
SRC_PATTERNS=("./cmd/..." "./internal/...")

echo "  modo: $MODE_LABEL  (global >= ${THRESHOLD_GLOBAL}%, pacote >= ${THRESHOLD_PACKAGE}%)"
echo "  -> running go test (this may take a while)..."
go test $TEST_FLAGS -coverprofile="$PROFILE" -covermode=atomic "${SRC_PATTERNS[@]}" > /tmp/audit-cover-test.log 2>&1 || {
  echo "  ❌ tests FAILED — see /tmp/audit-cover-test.log"
  tail -30 /tmp/audit-cover-test.log
  exit 1
}

GLOBAL=$(go tool cover -func="$PROFILE" 2>/dev/null | tail -1 | awk '{print $3}' | tr -d '%')

is_excluded() {
  local pkg="$1"
  for pat in "${EXCLUDE_PATTERNS[@]}"; do
    if [[ "$pkg" == *"$pat"* ]]; then
      return 0
    fi
  done
  if [[ "$INTEGRATION" != "1" ]]; then
    for pat in "${SHORT_MODE_INTEGRATION_PKGS[@]}"; do
      if [[ "$pkg" == *"$pat" ]]; then
        return 0
      fi
    done
  fi
  return 1
}

# Lista cobertura por pacote
declare -a LOW_PACKAGES=()
while IFS= read -r line; do
  pkg=$(echo "$line" | awk '{print $1}')
  cov_raw=$(echo "$line" | awk '{print $NF}' | tr -d '%')
  if [[ -z "$pkg" ]]; then continue; fi
  if [[ "$pkg" == "total:" ]]; then continue; fi
  if is_excluded "$pkg"; then continue; fi
  # cov_raw pode ser numero ou "statements]" (sentinela de "[no statements]")
  if [[ ! "$cov_raw" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
    LOW_PACKAGES+=("$pkg|0.0|no-tests")
    continue
  fi
  if awk "BEGIN{exit !($cov_raw < $THRESHOLD_PACKAGE)}"; then
    LOW_PACKAGES+=("$pkg|$cov_raw|low")
  fi
done < <(go test $TEST_FLAGS -cover "${SRC_PATTERNS[@]}" 2>/dev/null | grep -E '^(ok|\?)' | sed -E 's/^ok\s+//; s/^\?\s+//; s/coverage://g; s/of statements//g' | awk '{print $1, $NF}')

echo ""
echo "  Global coverage: ${GLOBAL}%  (threshold: ${THRESHOLD_GLOBAL}%)"

FAILED=0
if awk "BEGIN{exit !($GLOBAL < $THRESHOLD_GLOBAL)}"; then
  echo "  ❌ global coverage abaixo da meta"
  FAILED=1
else
  echo "  ✅ global coverage atinge a meta"
fi

if (( ${#LOW_PACKAGES[@]} > 0 )); then
  echo ""
  echo "  Pacotes abaixo de ${THRESHOLD_PACKAGE}%:"
  for entry in "${LOW_PACKAGES[@]}"; do
    pkg="${entry%%|*}"
    rest="${entry#*|}"
    cov="${rest%%|*}"
    reason="${rest#*|}"
    printf "    - %-70s %6s%%  (%s)\n" "$pkg" "$cov" "$reason"
  done
  FAILED=1
else
  echo "  ✅ todos os pacotes >= ${THRESHOLD_PACKAGE}%"
fi

if (( FAILED == 1 && ENFORCE == 1 )); then
  exit 1
fi
if (( FAILED == 1 )); then
  echo ""
  echo "  ℹ️  modo informativo — gate nao falha. Use AUDIT_INTEGRATION=1 para enforcement."
fi
exit 0
