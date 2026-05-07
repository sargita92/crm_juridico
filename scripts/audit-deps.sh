#!/usr/bin/env bash
# scripts/audit-deps.sh
# Lista dependencias com update minor/patch disponivel.
# NAO falha o build — informa apenas (decisao manual).
#
# Para falhar em vulnerabilidades, ver audit-vuln (govulncheck).

set -euo pipefail

echo "  -> deps com update disponivel:"
RAW=$(go list -m -u all 2>/dev/null | grep -E '\[' || true)
if [[ -z "$RAW" ]]; then
  echo "  ✅ tudo atualizado"
  exit 0
fi

# Filtra apenas minor/patch (heuristica: se a versao base e major-bump diferente, marcar como [major])
echo "$RAW" | while read -r line; do
  pkg=$(echo "$line" | awk '{print $1}')
  cur=$(echo "$line" | awk '{print $2}')
  next=$(echo "$line" | sed -E 's/.*\[([^]]+)\].*/\1/')
  cur_major=$(echo "$cur" | sed -E 's/^v([0-9]+)\..*/\1/')
  next_major=$(echo "$next" | sed -E 's/^v([0-9]+)\..*/\1/')
  if [[ "$cur_major" != "$next_major" ]]; then
    printf "    [MAJOR]  %-50s %s -> %s\n" "$pkg" "$cur" "$next"
  else
    printf "    [m/p]    %-50s %s -> %s\n" "$pkg" "$cur" "$next"
  fi
done

echo ""
echo "  ℹ️  audit-deps nao quebra build — atualizacoes major sao tratadas em features dedicadas"
exit 0
