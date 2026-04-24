# Status F18 — Observabilidade Avançada

**Branch**: `feature/F18-observabilidade-avancada`
**Status**: em andamento
**Spec**: [`../../superpowers/specs/2026-04-24-F18-observabilidade-avancada-design.md`](../../superpowers/specs/2026-04-24-F18-observabilidade-avancada-design.md) (gitignored — local)
**Plano**: [`../../superpowers/plans/2026-04-24-F18-observabilidade-avancada.md`](../../superpowers/plans/2026-04-24-F18-observabilidade-avancada.md) (gitignored — local)
**Inventário**: [`inventario.md`](inventario.md)

## Fluxo de agentes

PO (inline) → UI/UX (não aplicável) → Arquiteto (inline) → QA (promtool + go test) → Dev Backend → Segurança (review final).

## Progresso (Tasks concluídas)

| Task | Descrição | Commit |
|------|-----------|--------|
| Preflight | Branch + artefatos + inventário | — |
| 1 | `observability.StartSpan` helper com testes | — |
| 2 | `observability.LoggerFromContext` com testes | — |
| 3 | Registradores centrais (`metrics.go`) | — |
| 4 | `InitTracer` suporta OTLP via env | — |
| 5 | Infra: tempo + alertmanager no compose | — |
| 6 | Spans em `automation` | — |
| 7 | Spans em `permission` + `auth` | — |
| 8 | Spans em `notification` | — |
| 9 | Spans em `funnel` | — |
| 10 | Spans em `whatsapp` | — |
| 11 | Span em `ai.usecase.respond` (motor IA) | — |
| 12 | Histograma `automation_execution_duration_seconds` | — |
| 13 | Histograma `permission_check_duration_seconds` | — |
| 14 | Histograma `specialist_response_duration_seconds` | — |
| 15 | `invites_total{outcome=expired}` | — |
| 16 | `load_balance_fallback_total` + scope load_balance | — |
| 17 | `notification_read_total{type}` | — |
| 18 | `automation_rate_limited_total{type}` | — |
| 19 | Dashboards: overview + whatsapp | — |
| 20 | Dashboards: leads-kanban + especialistas + equipe | — |
| 21 | `alerts.yml` + testes promtool no CI | — |
| 22 | 6 runbooks | — |
| 23 | Docs final + PR | — |

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
2. Abrir o plano (gitignored, local): `docs/superpowers/plans/2026-04-24-F18-observabilidade-avancada.md`
3. Identificar a primeira task `- [ ]` não marcada
4. Executar via subagent-driven-development (implementer → spec-reviewer → code-quality-reviewer)
5. Ao concluir cada task, atualizar este `status.md` e commitar
