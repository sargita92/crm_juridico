# Status F18 — Observabilidade Avançada

**Branch**: `feature/F18-observabilidade-avancada`
**Status**: **concluído em 2026-04-24** — 23/23 tasks entregues.
**Spec**: [`../../superpowers/specs/2026-04-24-F18-observabilidade-avancada-design.md`](../../superpowers/specs/2026-04-24-F18-observabilidade-avancada-design.md) (gitignored — local)
**Plano**: [`../../superpowers/plans/2026-04-24-F18-observabilidade-avancada.md`](../../superpowers/plans/2026-04-24-F18-observabilidade-avancada.md) (gitignored — local)
**Inventário**: [`inventario.md`](inventario.md)

## Fluxo de agentes

PO (inline) → UI/UX (não aplicável) → Arquiteto (inline) → QA (promtool + go test) → Dev Backend → Segurança (review final).

## Progresso

| Task | Descrição | Commit |
|------|-----------|--------|
| Preflight | Branch + artefatos + inventário | `f1556a2` ✅ |
| 1 | `observability.StartSpan` helper com testes | `4de9eeb` ✅ |
| 2 | `observability.LoggerFromContext` com testes | `dcc0c8c` ✅ |
| 3 | Registradores centrais (`metrics.go`) | `abe1a3e` ✅ |
| 4 | `InitTracer` suporta OTLP via env | `76e7bff` ✅ |
| 5 | Infra: tempo + alertmanager no compose | `cd9ff19` ✅ |
| 6 | Spans em `automation` | `6e2d6bc` ✅ |
| 7 | Spans em `permission` + `auth` | `3d2c922` ✅ |
| 8 | Spans em `notification` | `266f6d0` ✅ |
| 9 | Spans em `funnel` | `c55d920` ✅ |
| 10 | Spans em `whatsapp` | `c63dbe6` ✅ |
| 11 | Span em `ai.usecase.respond` (motor IA) | `2009d14` ✅ |
| 12 | Histograma `automation_execution_duration_seconds` | `8a88f68` ✅ |
| 13 | Histograma `permission_check_duration_seconds` | `eef80ed` ✅ |
| 14 | Histograma `specialist_response_duration_seconds` | `4a04015` ✅ |
| 15 | `invites_total{outcome=expired}` | `0b27f59` ✅ |
| 16 | `load_balance_fallback_total` + scope load_balance | `e8b4cdc` ✅ |
| 17 | `notification_read_total{type}` | `50e5654` ✅ |
| 18 | `automation_rate_limited_total{type}` | `9dc6881` ✅ |
| 19 | Dashboards: overview + whatsapp | `99caa23` ✅ |
| 20 | Dashboards: leads-kanban + especialistas + equipe | `4767afb` ✅ |
| 21 | `alerts.yml` (4 regras) + testes promtool + CI | `c7c124c` ✅ |
| 22 | 4 runbooks + README | `558422b` ✅ |
| 23 | Docs final + PR | (este commit) ✅ |

Além dos commits de feature acima, há um commit de chore `802d526` que move `metrics_registered_test.go` para `package observability_test` (quebrou um ciclo de import pré-existente introduzido ao adicionar spans ao módulo funnel na Task 9).

## Ajustes de escopo (documentados)

1. **Alertas `WhatsAppDisconnected` e `SlowDatabase` removidos** do `alerts.yml`. Dependiam de métricas inexistentes hoje (`crm_whatsapp_sessions_active`, `crm_db_query_duration_seconds`). Criar essas métricas ficou fora do escopo de F18 — abrir ticket de follow-up para gauge de sessão WhatsApp e histograma de latência de query Gorm. Como consequência, os runbooks `whatsapp-desconectado.md` e `banco-lento.md` também não foram criados (4 runbooks entregues em vez de 6).
2. **`automation_rate_limited_total` sem call site**: não há rate limiter no módulo `automation` hoje. O counter foi registrado (aparece em `/metrics` com valor zero) para que dashboards e alertas possam referenciá-lo desde já. Será incrementado quando o rate limiter for introduzido.
3. **CRUD de automação sem span**: os métodos `CRUDUseCase.{Create,Update,Delete,…}` ficaram sem span, pois tracing de CRUD direto tem baixo valor. Spans foram priorizados nos caminhos operacionais (engine + executors).

## Lições aprendidas (aplicar em features futuras)

1. **Module path é `github.com/sasrgita/crm-juridico`** (não `crm_juridico`). Usar esse prefix em TODOS os imports.
2. **NÃO cachear `otel.Tracer(name)` em package-level `var tracer = …`**. Quebra testes que usam `otel.SetTracerProvider(…)` para registrar um `tracetest.SpanRecorder`, porque a referência cacheada fica vinculada ao provider default (noop) na hora do `init`. Chamar `otel.Tracer(tracerName)` dentro da função é correto — o OTel SDK já cacheia internamente.
3. **Pattern de teste OTel estabelecido**: `tracetest.NewSpanRecorder()` + `sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))` + `otel.SetTracerProvider(tp)`. Reutilizar o helper `newInMemoryTracer(t)` quando no mesmo package de teste; caso contrário, replicar inline (5 linhas).
4. **Spans existentes não renomear**: `internal/pagamentos/` e `internal/dashboard/` usam PascalCase (`RegisterManualPayment`, `GetTenantDashboard`). F18 só adiciona spans novos; não altera nomenclatura histórica.
5. **Namespace Prometheus**: métricas novas transversais usam `Namespace: "crm"`. Módulos que historicamente publicam sem prefixo (`pagamentos`, `whatsapp`, HTTP middleware) continuam sem prefixo para não quebrar dashboards existentes.
6. **HTTP middleware** (`internal/shared/middleware/prometheus.go`) usa `http_requests_total` e `http_request_duration_seconds` **SEM prefixo `crm_`**. Regras de alerta refletem isso.
7. **Descartar drive-by changes**: subagents podem rodar `goimports` e modificar arquivos fora do escopo da task. Rodar `git status` depois de cada task e `git checkout --` no que não pertence.
8. **`go mod tidy` quebra com `storage/files` permission denied** (diretório criado pelo container Docker como root). Workaround: editar `go.mod` à mão para promover novas deps do bloco `// indirect` para o bloco direto, e validar com `go build ./cmd/... ./internal/...` em vez de `./...`. (Aplicado na Task 4 com `otlptrace`/`otlptracegrpc`.)
9. **Spans em executors com histograma observado no `defer`**: padrão é `func (…) Execute(…) (err error) { … defer func(){ outcome := "success"; if err != nil { outcome = "error" }; hist.WithLabelValues(type, outcome).Observe(…) }() }`. Named return `err` é necessário para o defer ver o erro final.
10. **Ciclo de imports**: adicionar import de um package de módulo (`internal/funnel/application`) em `internal/shared/observability` via teste cria ciclo. Testes que consomem instrumentação cross-module devem ficar em `package <pkg>_test` (black-box) para evitar o ciclo.

## Decisões-chave (recap)

- **Trace exporter**: Grafana Tempo via OTLP gRPC; fallback stdout quando `OTEL_EXPORTER_OTLP_ENDPOINT` vazio.
- **Retenção**: 15 dias métricas (Prometheus), 7 dias traces (Tempo).
- **Validação de alertas**: `promtool test rules` no CI (`.github/workflows/ci.yml`) — não depende de staging.
- **Alertmanager dev**: receiver `null`; Slack/email via env vars em prod.
- **Spans existentes**: pagamentos e dashboard mantidos com a nomenclatura histórica (PascalCase).
- **Módulo "especialista" = `internal/ai/`**.

## Comandos úteis

```bash
# Confirmar estado da branch
git log --oneline main..HEAD

# Testes do pacote de observabilidade
go test ./internal/shared/observability/... -v

# Validar regras de alerta localmente
docker run --rm -v "$(pwd)/infra/prometheus:/etc/prometheus" \
  prom/prometheus:v2.53.0 promtool test rules /etc/prometheus/alerts_test.yml

# Smoke do stack completo
docker compose -f docker-compose.dev.yml.dist up -d
curl -sf http://localhost:3200/ready          # tempo
curl -sf http://localhost:9093/-/ready        # alertmanager
curl -sf http://localhost:9190/-/ready        # prometheus
```
