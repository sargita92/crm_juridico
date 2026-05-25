#!/usr/bin/env bash
# scripts/install-hooks.sh
# Instala os git hooks versionados (scripts/hooks/) apontando core.hooksPath para lá.
# Rodar UMA VEZ após clonar o repo — o git não auto-instala hooks por segurança.

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

git config core.hooksPath scripts/hooks
chmod +x scripts/hooks/* 2>/dev/null || true

echo "✅ git hooks instalados (core.hooksPath = scripts/hooks)"
echo "   hooks ativos:"
for h in scripts/hooks/*; do
  [[ -f "$h" ]] && echo "     - $(basename "$h")"
done
