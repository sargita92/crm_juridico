---
feature: F26 — Gargalo intermitente de banco
agent: Arquiteto
version: 1
created_at: 2026-05-26
reason: Plano técnico (steps) da entrega de instrumentação — pré-requisito para confirmar a causa-raiz
---

# F26 — Plano técnico (instrumentação)

Objetivo desta entrega: **tornar o gargalo visível** sem alterar comportamento de
negócio. Cada step é incremental, validado isoladamente, com TDD e commit
atômico (ver `docs/processo/definition-of-done.md`).

> Nada aqui é "a correção". A correção sai numa entrega seguinte, guiada pela
> evidência que estes instrumentos vão capturar (hipóteses H4/H5/H6 em
> [`investigacao-v1.md`](investigacao-v1.md)).

## Step 1 — Métricas do pool de conexões (`sql.DBStats`)

- Registrar `collectors.NewDBStatsCollector(sqlDB, dbName)` no registry default
  do Prometheus (já exposto em `/metrics`).
- Expõe `go_sql_*`: `open_connections`, `in_use`, `idle`, **`wait_count`**,
  **`wait_duration_seconds`**, `max_open_connections`.
- `wait_count`/`wait_duration` crescendo = **prova de exaustão de pool** (H4).
- **TDD**: registrar num registry isolado e assertar que a família
  `go_sql_open_connections` aparece em `Gather()`.

## Step 2 — Slow-query log estruturado (Gorm + zap)

- Substituir o logger `Silent` por um logger Gorm custom respaldado em zap.
- Loga em `Warn` queries acima de `SlowThreshold` (configurável); inclui SQL,
  duração, linhas afetadas e contexto (request_id, tenant_id, user_id) extraído
  do `ctx`.
- Config nova: `Database.SlowQueryThresholdMs` (env `DB_SLOW_QUERY_THRESHOLD_MS`,
  default 200ms). `0` desativa.
- **TDD**: `zaptest/observer` — acima do limiar loga `Warn` com SQL/duração;
  abaixo, não loga em `Warn`.

## Step 3 — Tracing OTel por query (Gorm plugin)

- Instalar `gorm.io/plugin/opentelemetry/tracing` e `db.Use(tracing.NewPlugin(...))`.
- Spans por query, correlacionados com o trace do request (localiza **qual**
  query é lenta — H5).
- **TDD**: provider OTel de teste (in-memory exporter) + uma query → assertar que
  um span de DB foi emitido como filho do span do request.

## Step 4 — Endpoint pprof protegido

- Registrar `net/http/pprof` em `/debug/pprof/*`, **gated** por flag
  (`PPROF_ENABLED`, default `false`) **e** middleware admin.
- Captura CPU/heap/goroutine se o gargalo for fora do banco (H6).
- **TDD**: flag on → rota responde; flag off → 404; sem admin → 401/403.

## Step 5 — Config, dashboards e docs

- Defaults de config + binds de env documentados.
- Queries/painel Grafana para pool (`wait_count`, `in_use`) e slow-query.
- Atualizar `rest/00-health.http` (nota sobre métricas de pool) e adicionar
  `/debug/pprof` em `rest/` com teste OWASP de acesso não autorizado.
- Atualizar `status.md` e `docs/processo/backlog.md`.

## Não-objetivos

- Alterar tamanho do pool, adicionar índices, statement timeout ou mexer em
  queries — **só após** a causa-raiz confirmada.
- Telas/UX (bug de backend).

## Ordem e validação

`Step 1 → 2 → 3 → 4 → 5`. Ao fim de cada step: `go test -short ./...`,
`golangci-lint run ./internal/... ./cmd/...`, `go vet`, build ok, commit atômico.
Integração (testcontainers) roda no pre-push.
