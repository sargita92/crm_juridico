---
feature: F21 — Saneamento Técnico (Faxina Mecânica)
branch: feature/F21-saneamento-tecnico
updated_at: 2026-05-07
status: concluído
---

# Status F21 — Saneamento Técnico

## Fluxo de agentes
- PO: ✅ v1
- UI/UX: ✅ v1 (sem UI — feature mecânica)
- Arquiteto: ✅ v1
- Dev Backend: ✅
- QA: ✅ (1921 testes -short verdes; 0 lint; 0 vuln)
- Segurança: ✅ (vulnerabilidades zeradas com lista de aceite documentada)

## Progresso

| Step | Descrição | Status | Commit |
|---|---|---|---|
| 1 | Tooling: Makefile + .golangci.yml + audit scripts | ✅ | 1c54bbd |
| 2 | Relatório inicial (audit-inicial-2026-05-07.md) | ✅ | dfc29b7 |
| 2.5 | Fix regressão UI redesign (audit/interfaces/http) | ✅ | f5b7832 |
| 3 | Vulnerabilidades (Go 1.26.3 + x/net + accept-list) | ✅ | (no Step 3 commit) |
| 4 | Deps diretas + fix `-test.short` flag | ✅ | ac7e147 |
| 5+8+9 | Cobertura dual-mode + CI gates + processo | ✅ | 0e87240 |
| 6+7 | Lint zerado + código morto removido | ✅ | ec3be84 |
| 10 | Relatório final + backlog + changelog + PR | 🔄 | (em curso) |

## Resultados

| Métrica | Antes | Depois |
|---|---|---|
| Vulnerabilidades não-aceitas | 14 | 0 |
| Lint issues | 199 | 0 |
| Pacotes < 80% (produtivos) | 10 | 0 (com infra exempt no -short) |
| Testes verdes (-short) | 1911/1921 | 1921/1921 |
| CI gates | 1 (promtool) | 8 (build, vet, lint, vuln, test-short, test-integration, cover, promtool) |

## Decisões consolidadas

- `govet.shadow` desabilitado: codebase usa `err :=` em escopos internos
- 2 vulns docker/docker aceitas: transitivas, sem fix upstream, sem call path produtivo
- `audit-cover` em 2 modos: -short (dev, informativo) e AUDIT_INTEGRATION=1 (CI, enforce)
- Updates major fora do escopo (ficam como features dedicadas)

## Como retomar (se precisar)
- Branch: `feature/F21-saneamento-tecnico`
- PR aberto em (ver final do step 10)
- Rodar `make audit` para verificar estado
