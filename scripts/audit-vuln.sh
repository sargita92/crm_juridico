#!/usr/bin/env bash
# scripts/audit-vuln.sh
# Roda govulncheck em JSON, filtra IDs aceitas (.audit-accepted-vulns.txt),
# e falha se sobrar qualquer vuln com call traces no codigo produtivo.

set -euo pipefail

ACCEPTED_FILE="${ACCEPTED_FILE:-.audit-accepted-vulns.txt}"
GOVULNCHECK="${GOVULNCHECK:-$(go env GOPATH)/bin/govulncheck}"
PATTERNS=("./cmd/..." "./internal/...")

if [[ ! -x "$GOVULNCHECK" ]]; then
  echo "  ❌ govulncheck nao encontrado em $GOVULNCHECK"
  echo "     instale com: make tools"
  exit 2
fi

# IDs aceitas (uma por linha, primeira coluna até o '|' ou whitespace).
declare -A ACCEPTED
if [[ -f "$ACCEPTED_FILE" ]]; then
  while IFS= read -r line; do
    [[ -z "$line" || "$line" =~ ^# ]] && continue
    id=$(echo "$line" | awk -F'|' '{print $1}' | tr -d '[:space:]')
    [[ -z "$id" ]] && continue
    ACCEPTED["$id"]=1
  done < "$ACCEPTED_FILE"
fi

OUT=$(mktemp)
trap 'rm -f "$OUT"' EXIT

# Roda em modo source; captura saida humana
"$GOVULNCHECK" "${PATTERNS[@]}" > "$OUT" 2>&1 || true

# Extrai todas as IDs reportadas
ALL_IDS=$(grep -oE 'GO-[0-9]+-[0-9]+' "$OUT" | sort -u || true)

if [[ -z "$ALL_IDS" ]]; then
  echo "  ✅ nenhuma vulnerabilidade encontrada"
  exit 0
fi

UNACCEPTED=()
ACCEPTED_HITS=()
for id in $ALL_IDS; do
  if [[ -n "${ACCEPTED[$id]:-}" ]]; then
    ACCEPTED_HITS+=("$id")
  else
    UNACCEPTED+=("$id")
  fi
done

if (( ${#ACCEPTED_HITS[@]} > 0 )); then
  echo "  ℹ️  ${#ACCEPTED_HITS[@]} vuln(s) aceita(s) (ver $ACCEPTED_FILE):"
  for id in "${ACCEPTED_HITS[@]}"; do
    echo "    - $id"
  done
fi

if (( ${#UNACCEPTED[@]} > 0 )); then
  echo ""
  echo "  ❌ FAIL: ${#UNACCEPTED[@]} vuln(s) NAO aceita(s):"
  for id in "${UNACCEPTED[@]}"; do
    echo "    - $id"
  done
  echo ""
  echo "  Saida completa do govulncheck:"
  cat "$OUT"
  exit 1
fi

echo "  ✅ OK: 0 vuln nao aceitas"
exit 0
