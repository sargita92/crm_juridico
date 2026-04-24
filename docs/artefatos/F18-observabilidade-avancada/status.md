# Status F18 — Observabilidade Avançada

**Branch**: `feature/F18-observabilidade-avancada`
**Status**: em andamento — Task 14 concluída em 2026-04-24 (14/23 concluídas, próxima: Task 15)
**Spec**: [`../../superpowers/specs/2026-04-24-F18-observabilidade-avancada-design.md`](../../superpowers/specs/2026-04-24-F18-observabilidade-avancada-design.md) (gitignored — local)
**Plano**: [`../../superpowers/plans/2026-04-24-F18-observabilidade-avancada.md`](../../superpowers/plans/2026-04-24-F18-observabilidade-avancada.md) (gitignored — local)
**Inventário**: [`inventario.md`](inventario.md)

## Fluxo de agentes

PO (inline) → UI/UX (não aplicável) → Arquiteto (inline) → QA (promtool + go test) → Dev Backend → Segurança (review final).

## Progresso (Tasks concluídas)

| Task | Descrição | Commit |
|------|-----------|--------|
| Preflight | Branch + artefatos + inventário | `f1556a2` ✅ |
| 1 | `observability.StartSpan` helper com testes | `4de9eeb` ✅ |
| 2 | `observability.LoggerFromContext` com testes | `dcc0c8c` ✅ |
| 3 | Registradores centrais (`metrics.go`) | `abe1a3e` ✅ |
| 4 | `InitTracer` suporta OTLP via env | `76e7bff` ✅ |
| 5 | Infra: tempo + alertmanager no compose | (commit anterior) ✅ |
| 6 | Spans em `automation` | (commit anterior) ✅ |
| 7 | Spans em `permission` + `auth` | (commit anterior) ✅ |
| 8 | Spans em `notification` | (commit anterior) ✅ |
| 9 | Spans em `funnel` | (commit anterior) ✅ |
| 10 | Spans em `whatsapp` | (commit anterior) ✅ |
| 11 | Span em `ai.usecase.respond` (motor IA) | (commit anterior) ✅ |
| 12 | Histograma `automation_execution_duration_seconds` | (commit anterior) ✅ |
| 13 | Histograma `permission_check_duration_seconds` | (commit anterior) ✅ |
| 14 | Histograma `specialist_response_duration_seconds` | (este commit) ✅ |
| 15 | `invites_total{outcome=expired}` | — |
| 16 | `load_balance_fallback_total` + scope load_balance | — |
| 17 | `notification_read_total{type}` | — |
| 18 | `automation_rate_limited_total{type}` | — |
| 19 | Dashboards: overview + whatsapp | — |
| 20 | Dashboards: leads-kanban + especialistas + equipe | — |
| 21 | `alerts.yml` + testes promtool no CI | — |
| 22 | 6 runbooks | — |
| 23 | Docs final + PR | — |

## Lições aprendidas (aplicar nas próximas tasks)

1. **Module path é `github.com/sasrgita/crm-juridico`** (não `crm_juridico`). Usar esse prefix em TODOS os imports.
2. **NÃO cachear `otel.Tracer(name)` em package-level `var tracer = ...`**. Quebra testes que usam `otel.SetTracerProvider(...)` para registrar um `tracetest.SpanRecorder`, porque a referência cacheada fica vinculada ao provider default (noop) na hora do `init`. A Task 1 tentou a otimização e teve que reverter. Chamar `otel.Tracer(tracerName)` dentro da função é correto — o OTel SDK já cacheia internamente.
3. **Pattern de teste OTel estabelecido**: helper `newInMemoryTracer(t)` em `internal/shared/observability/tracing_test.go` cria `tracetest.SpanRecorder` e registra como provider global. Reutilizar nos testes do mesmo package `observability_test` (já é reusado em `logging_test.go`).
4. **Spans existentes não renomear**: `internal/pagamentos/application/` usa `otel.Tracer("pagamentos").Start(ctx, "RegisterManualPayment")` (PascalCase). `internal/dashboard/application/` também. F18 só adiciona spans onde não existem; não altera nomenclatura dos existentes.
5. **Namespace Prometheus `crm`** para métricas transversais novas (Task 3 já fez). Módulos que historicamente usam sem prefixo (`pagamentos`, `whatsapp`) continuam sem prefixo para não quebrar dashboards existentes.
6. **HTTP middleware** (`internal/shared/middleware/prometheus.go`) usa `http_requests_total` e `http_request_duration_seconds` **SEM prefixo `crm_`**. Regras de alerta (Task 21) devem refletir isso.
7. **Descartar drive-by changes**: subagents podem rodar `goimports` e modificar arquivos fora do escopo da task (aconteceu em Task 3 com `metrics_registered_test.go`). Rodar `git status` depois de cada task e `git checkout --` no que não pertence.
8. **`go mod tidy` quebra com `storage/files` permission denied** (diretório criado pelo container Docker como root). Workaround: editar `go.mod` à mão para promover novas deps do bloco `// indirect` para o bloco direto, e validar com `go build ./cmd/... ./internal/...` em vez de `./...`. (Aplicado na Task 4 com `otlptrace`/`otlptracegrpc`.)

## Decisões-chave

- **Trace exporter**: Grafana Tempo via OTLP gRPC; fallback stdout quando `OTEL_EXPORTER_OTLP_ENDPOINT` vazio.
- **Retenção**: 15 dias métricas (Prometheus), 7 dias traces (Tempo).
- **Validação de alertas**: `promtool test rules` no CI — não depende de staging.
- **Alertmanager dev**: receiver `null` por padrão; Slack/email via env vars em prod.
- **Namespace Prometheus**: métricas novas transversais usam `Namespace: "crm"`; métricas de módulo respeitam o que já existe (pagamentos/whatsapp sem prefixo, notification/auth com prefixo).
- **Spans existentes não são renomeados**: `pagamentos` (PascalCase) e `dashboard` (PascalCase) ficam intocados para preservar rastros históricos.
- **Módulo "especialista" = `internal/ai/`**.

## Ajustes ao plano (descobertos no inventário)

1. Alertas HTTP usam `http_requests_total` / `http_request_duration_seconds_bucket` (sem prefixo `crm_`) — refletir na Task 21.
2. `WhatsAppDisconnected` e `SlowDatabase` dependem de métricas que NÃO existem hoje (gauge de sessão WhatsApp, histograma de query MySQL). Durante a Task 5/21 decidir entre criar essas métricas ou remover os respectivos alertas do escopo.
3. Task 11: apontar para `internal/ai/application/` em vez de `internal/specialist/`.

## Como retomar

1. `git checkout feature/F18-observabilidade-avancada`
2. Confirmar estado: `git log --oneline main..HEAD` — esperar 4 commits: preflight + Tasks 1, 2, 3.
3. Rodar `go test ./internal/shared/observability/... -v` — deve passar (sanity check).
4. Abrir o plano (gitignored, local): `docs/superpowers/plans/2026-04-24-F18-observabilidade-avancada.md`.
5. **Retomar na Task 4** (`InitTracer` suporta OTLP via env). O plano tem o conteúdo completo. Antes de começar, ler `internal/shared/observability/otel.go` atual para ver a assinatura exata de `InitTracer` (hoje é `InitTracer(serviceName string) (*sdktrace.TracerProvider, error)`).
6. Seguir fluxo subagent-driven-development para cada task (implementer → spec-reviewer → code-quality-reviewer). Para tasks muito pequenas (Task 4, 15, 17, 18) dá pra fazer spec review inline e pular o code quality review — como foi feito em Tasks 2 e 3.
7. Ao concluir cada task, atualizar este `status.md` (preencher coluna `Commit` + mover o marcador **← RETOMAR AQUI**) e commitar.
8. **Atenção às Lições aprendidas acima** — elas encaixam direto em várias tasks.

## Comando rápido para retomar

```bash
git checkout feature/F18-observabilidade-avancada && \
  git log --oneline main..HEAD && \
  go test ./internal/shared/observability/... -v | tail -10
```

Esperado: 5 commits listados; 9 testes PASS (2 `tracing`, 2 `logging`, 3 `metrics` + `metrics_registered`, 2 `otel`).
