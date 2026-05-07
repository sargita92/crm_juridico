#!/usr/bin/env bash
# scripts/audit-cover.sh
# Roda testes com cobertura, calcula global e por-pacote, e falha se abaixo da meta.
#
# Metas:
#   - Global: >= 85%
#   - Por pacote (produtivo): >= 80%
#
# Pacotes ignorados (wiring/main):
#   - cmd/...
#   - internal/shared/module/...
#   - .../module.go (wiring)

set -euo pipefail

THRESHOLD_GLOBAL="${THRESHOLD_GLOBAL:-85.0}"
THRESHOLD_PACKAGE="${THRESHOLD_PACKAGE:-80.0}"

EXCLUDE_PATTERNS=(
  "/cmd/"
  "/internal/shared/module"
)

PROFILE="${COVER_PROFILE:-coverage.out}"

SRC_PATTERNS=("./cmd/..." "./internal/...")

echo "  -> running go test -cover (this may take a while)..."
go test -coverprofile="$PROFILE" -covermode=atomic "${SRC_PATTERNS[@]}" > /tmp/audit-cover-test.log 2>&1 || {
  echo "  ❌ tests FAILED — see /tmp/audit-cover-test.log"
  tail -30 /tmp/audit-cover-test.log
  exit 1
}

GLOBAL=$(go tool cover -func="$PROFILE" | tail -1 | awk '{print $3}' | tr -d '%')

is_excluded() {
  local pkg="$1"
  for pat in "${EXCLUDE_PATTERNS[@]}"; do
    if [[ "$pkg" == *"$pat"* ]]; then
      return 0
    fi
  done
  return 1
}

# Lista cobertura por pacote
declare -a LOW_PACKAGES=()
while IFS= read -r line; do
  pkg=$(echo "$line" | awk '{print $1}')
  cov=$(echo "$line" | awk '{print $2}' | tr -d '%')
  if [[ -z "$pkg" || -z "$cov" ]]; then continue; fi
  if [[ "$pkg" == "total:" ]]; then continue; fi
  if is_excluded "$pkg"; then continue; fi
  if [[ "$cov" == "[no" ]]; then
    # "[no test files]" — considerar como abaixo se não excluído
    LOW_PACKAGES+=("$pkg|0.0|no-tests")
    continue
  fi
  if awk "BEGIN{exit !($cov < $THRESHOLD_PACKAGE)}"; then
    LOW_PACKAGES+=("$pkg|$cov|low")
  fi
done < <(go test -cover "${SRC_PATTERNS[@]}" 2>/dev/null | grep -E '^(ok|---|\?)' | sed -E 's/^ok\s+//; s/^\?\s+//; s/\[no test files\]/[no/g; s/coverage://g; s/of statements//g' | awk '{print $1, $NF}' | grep -v '^---')

echo ""
echo "  Global coverage: ${GLOBAL}%  (threshold: ${THRESHOLD_GLOBAL}%)"

FAILED=0
if awk "BEGIN{exit !($GLOBAL < $THRESHOLD_GLOBAL)}"; then
  echo "  ❌ FAIL: global coverage abaixo da meta"
  FAILED=1
else
  echo "  ✅ OK:   global coverage atinge a meta"
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
fi

if (( FAILED == 1 )); then
  exit 1
fi

echo "  ✅ OK:   todos os pacotes >= ${THRESHOLD_PACKAGE}%"
