# Status F26 — Gargalo intermitente de banco

**Branch**: `feature/F26-gargalo-banco`
**Status**: em andamento — entrega de **instrumentação** (Fase 1 → 2 do debugging)
**Doc da feature**: [../../features/F26-gargalo-banco.md](../../features/F26-gargalo-banco.md)
**Investigação**: [investigacao-v1.md](investigacao-v1.md)
**Plano**: [plan-v1.md](plan-v1.md)

## Resumo

Bug de performance intermitente (latência ~19s). Causa-raiz **não confirmada**.
Esta entrega instrumenta o banco para localizar o gargalo por evidência, sem
alterar comportamento de negócio. A correção definitiva vem em entrega posterior.

## Fluxo de agentes

- PO: doc da feature (relato + critérios) — concluído
- Arquiteto: investigação Fase 1 + plano de steps — concluído
- Dev Backend: em andamento (instrumentação, TDD por step)
- QA: pendente (testes por step + OWASP no /debug/pprof)
- Segurança: pendente (acesso ao pprof, vazamento de dados em logs)

## Progresso por step

| Step | Descrição | Status |
|------|-----------|--------|
| 1 | Métricas do pool (`sql.DBStats` → /metrics) | pendente |
| 2 | Slow-query log (Gorm + zap, limiar configurável) | pendente |
| 3 | Tracing OTel por query (plugin Gorm) | pendente |
| 4 | Endpoint pprof protegido (flag + admin) | pendente |
| 5 | Config, dashboards, docs e .http | pendente |

## Commits da feature

| Commit | Descrição |
|--------|-----------|
| _(pendente)_ | |
