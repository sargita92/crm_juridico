---
feature: F21 — Saneamento Técnico (Faxina Mecânica)
branch: feature/F21-saneamento-tecnico
updated_at: 2026-05-07
status: em andamento
---

# Status F21 — Saneamento Técnico

## Fluxo de agentes
- PO: ✅ v1
- UI/UX: ✅ v1 (sem UI — feature mecânica)
- Arquiteto: ✅ v1
- Dev Backend: 🔄 em execução
- QA: ⏳ pendente
- Segurança: ⏳ pendente

## Progresso (Steps)

| Step | Descrição | Status | Commit |
|---|---|---|---|
| 1 | Implementar `make audit` | pendente | — |
| 2 | Relatório inicial e priorização | pendente | — |
| 3 | Correção de vulnerabilidades | pendente | — |
| 4 | Atualização de dependências (minor/patch) | pendente | — |
| 5 | Cobertura de pacotes < 80% | pendente | — |
| 6 | Lint e vet — zerar warnings | pendente | — |
| 7 | Código morto e TODOs órfãos | pendente | — |
| 8 | Gates no CI | pendente | — |
| 9 | Processo recorrente | pendente | — |
| 10 | Relatório final | pendente | — |

## Decisões

- Sem refactors arquiteturais (viram F23+ se aparecerem)
- Updates major fora do escopo
- Cobertura: meta global ≥ 85% e por-pacote ≥ 80%
- `govulncheck` deve zerar
- `golangci-lint` configurado para o projeto

## Como retomar
1. `git checkout feature/F21-saneamento-tecnico`
2. Identificar próximo step pendente nesta tabela
3. Seguir plano em `docs/artefatos/F21-saneamento-tecnico/arquiteto-design/v1.md`
