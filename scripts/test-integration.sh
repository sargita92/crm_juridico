#!/usr/bin/env bash
# scripts/test-integration.sh
# Roda os testes de integração: go test SEM -short, ou seja, executa também os
# testes gated por testing.Short() — incluindo os que sobem MySQL via testcontainers.
# Mesma invocação do antigo job `test-integration` do CI (removido; ver ci.yml).
#
# Timeout: testcontainers trava no ambiente (WSL2). INTEGRATION_TIMEOUT (default 180s)
# limita a execução para não pendurar quem chama (ex.: o hook de pre-push).
#
# Exit code real, para uso manual e por outros scripts:
#   0   = testes passaram
#   !=0 = testes falharam
#   124 = timeout (provável trava do testcontainers)

set -uo pipefail

TIMEOUT="${INTEGRATION_TIMEOUT:-180}"
PATTERNS=("./cmd/..." "./internal/...")

echo "  -> timeout ${TIMEOUT}s go test -p 1 -count=1 ${PATTERNS[*]}"

timeout "$TIMEOUT" go test -p 1 -count=1 "${PATTERNS[@]}" < /dev/null
code=$?

if [[ $code -eq 124 ]]; then
  echo "  ⚠️  timeout após ${TIMEOUT}s (testcontainers provavelmente travou)"
fi

exit $code
