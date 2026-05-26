# Status F26 — Gargalo intermitente de banco

**Branches**: `feature/F26-gargalo-banco` (instrumentação, PR #15) · `feature/F26-sse-conexao-unica` (correção, PR #16)
**Status**: **RESOLVIDO** — causa-raiz era SSE (não o banco); correção validada (usuário: "funcionando sem engasgar")

> Correção final: **SharedWorker** = 1 conexão SSE por navegador (`sse-worker.js` + `sse-bridge.js`),
> compartilhada entre abas e fechando na hora ao navegar. Evidência: 58/60 conexões `/tenant/stream`
> fecharam em <2s (mediana 407ms), `sse_active_streams=1`, nenhum slow request fora do SSE.
> "1 SSE por página" sozinho não bastou (page load recria a conexão). Ver revisão v2 em
> [correcao-sse-design-v1.md](correcao-sse-design-v1.md).
**Doc da feature**: [../../features/F26-gargalo-banco.md](../../features/F26-gargalo-banco.md)
**Investigação**: [investigacao-v1.md](investigacao-v1.md)
**Plano (instrumentação)**: [plan-v1.md](plan-v1.md)
**Design (correção)**: [correcao-sse-design-v1.md](correcao-sse-design-v1.md)

## Causa-raiz confirmada (diagnóstico ao vivo)

Com a instrumentação ligada, durante a lentidão o app estava a ~0% CPU, pool com
`go_sql_wait_count=0`, MySQL ocioso e **6 conexões TCP** no app = teto de ~6 do
HTTP/1.1. **Não era o banco**: cada página abria **2 SSE persistentes** (sino +
WhatsApp); ao trocar de aba rápido (page load completo), as conexões se sobrepõem
e estouram o limite → requests enfileiram no browser. Correção: **uma única
conexão SSE por página** (endpoint unificado `/tenant/stream`).

## Resumo

Bug de performance intermitente (latência ~19s). Causa-raiz **não confirmada**.
Esta entrega instrumenta o banco para localizar o gargalo por evidência, sem
alterar comportamento de negócio. A correção definitiva vem em entrega posterior.

## Fluxo de agentes

- PO: doc da feature (relato + critérios) — concluído
- Arquiteto: investigação Fase 1 + plano de steps — concluído
- Dev Backend: instrumentação (5 steps, TDD) — concluído
- QA: testes por step + OWASP no /debug/pprof — concluído (unitários verdes)
- Segurança: pprof atrás de auth+admin (401/403), SQL logado sem valores (placeholders) — coberto; revisão final pendente

## Progresso por step

| Step | Descrição | Status |
|------|-----------|--------|
| 1 | Métricas do pool (`sql.DBStats` → /metrics) | concluído |
| 2 | Slow-query log (Gorm + zap, limiar configurável) | concluído |
| 3 | Tracing OTel por query (tracer custom, sem deps novas) | concluído |
| 4 | Endpoint pprof protegido (flag + admin) | concluído |
| 5 | Config, dashboard `banco`, docs e .http | concluído |
| 6 | Middleware `ResponseTime` (header X-Response-Time + Warn slow request) | concluído |

> Step 6 (pós-feedback): repro relatado é **lentidão ao trocar de aba rápido**
> (rajada de requests concorrentes → reforça H4 exaustão de pool). O middleware
> expõe a latência por request para flagrar qual endpoint trava na rajada.

> Nota Step 3: descartado o plugin oficial `gorm.io/plugin/opentelemetry` por
> arrastar drivers clickhouse/postgres (~18 deps) num app MySQL-only; implementado
> tracer custom via callbacks do Gorm, zero dependências novas.

## Commits da feature

| Commit | Descrição |
|--------|-----------|
| `0489451` | docs(f26): investigação Fase 1 + plano |
| `604c2a6` | feat(observability): métricas do pool (sql.DBStats) |
| `f33b166` | feat(database): slow-query log (Gorm + zap) |
| `9eb1a20` | feat(database): tracing OTel por query |
| `7938bd8` | feat(profiling): /debug/pprof protegido |
| _(pendente)_ | docs/config/dashboard (Step 5) |

## Próximo passo (após esta entrega)

Observar/reproduzir com a instrumentação ligada e **confirmar a causa-raiz**
(H4 pool / H5 query lenta / H6 fora do banco — ver investigacao-v1.md). A
correção definitiva é uma entrega separada.
