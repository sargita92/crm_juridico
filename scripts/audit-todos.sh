#!/usr/bin/env bash
# scripts/audit-todos.sh
# Lista TODO/FIXME no codigo Go com idade (via git blame).
# Falha se algum tiver mais de MAX_AGE_DAYS dias.

set -euo pipefail

MAX_AGE_DAYS="${MAX_AGE_DAYS:-90}"
NOW_TS=$(date +%s)

# Localiza ocorrencias em arquivos .go (exclui scripts, docs)
matches=$(grep -rnE 'TODO|FIXME' --include='*.go' . 2>/dev/null || true)

if [[ -z "$matches" ]]; then
  echo "  ✅ nenhum TODO/FIXME encontrado"
  exit 0
fi

declare -a OLD_ITEMS=()
total=0
while IFS= read -r line; do
  total=$((total+1))
  file=$(echo "$line" | cut -d: -f1)
  lineno=$(echo "$line" | cut -d: -f2)
  text=$(echo "$line" | cut -d: -f3-)

  # blame da linha
  blame=$(git blame -L "$lineno,$lineno" --porcelain "$file" 2>/dev/null | grep '^author-time' | head -1 | awk '{print $2}' || echo "")
  if [[ -z "$blame" ]]; then continue; fi

  age_days=$(( (NOW_TS - blame) / 86400 ))
  if (( age_days > MAX_AGE_DAYS )); then
    OLD_ITEMS+=("$age_days|$file:$lineno|$text")
  fi
done <<< "$matches"

echo "  total TODO/FIXME: $total"
echo "  threshold idade: ${MAX_AGE_DAYS} dias"

if (( ${#OLD_ITEMS[@]} == 0 )); then
  echo "  ✅ OK: nenhum TODO/FIXME mais antigo que ${MAX_AGE_DAYS} dias"
  exit 0
fi

echo "  ❌ FAIL: ${#OLD_ITEMS[@]} TODOs/FIXMEs com idade > ${MAX_AGE_DAYS} dias:"
# ordena desc por idade
printf '%s\n' "${OLD_ITEMS[@]}" | sort -t'|' -k1,1nr | while IFS='|' read -r age loc txt; do
  printf "    %4d dias  %s\n              %s\n" "$age" "$loc" "$(echo "$txt" | sed 's/^[[:space:]]*//')"
done
exit 1
