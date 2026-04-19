# Observabilidade

## Visão geral

Observabilidade bem projetada para ter visão completa do sistema e diagnosticar problemas rapidamente. Baseada nos três pilares: logs, métricas e traces.

## Pilares

### Logs (Zap)

- logging estruturado em JSON
- níveis: debug, info, warn, error
- contexto obrigatório em todo log: `tenant_id`, `user_id`, `request_id`
- nunca logar dados sensíveis (senhas, tokens, documentos pessoais)
- logs vão para stdout (container-friendly)
- formato consistente em toda a aplicação

Campos padrão por log:
```json
{
  "level": "info",
  "timestamp": "2026-04-05T10:30:00Z",
  "request_id": "uuid",
  "tenant_id": "uuid",
  "user_id": "uuid",
  "module": "whatsapp",
  "message": "mensagem recebida",
  "duration_ms": 45
}
```

### Métricas (Prometheus)

Métricas obrigatórias:

| Métrica | Tipo | Descrição |
|---------|------|-----------|
| `http_requests_total` | Counter | Total de requests HTTP (por rota, método, status) |
| `http_request_duration_seconds` | Histogram | Latência de requests HTTP |
| `whatsapp_messages_received_total` | Counter | Mensagens recebidas do WhatsApp |
| `whatsapp_messages_sent_total` | Counter | Mensagens enviadas pelo WhatsApp |
| `whatsapp_message_processing_duration_seconds` | Histogram | Tempo de processamento de mensagem |
| `leads_created_total` | Counter | Leads criados (por funil) |
| `leads_moved_total` | Counter | Leads movidos entre colunas |
| `crm_automation_executions_total` | Counter | Automações executadas (labels: `type` ∈ 6 executors, `outcome` ∈ success/error) |
| `crm_permission_changes_total` | Counter | Alterações de escopo de permissão (labels: `scope` ∈ group/user/funnel/view_profile, `action` = updated) |
| `crm_auth_invites_total` | Counter | Ciclo de vida de convites (label: `outcome` ∈ sent/accepted/revoked) |
| `crm_files_stored_total` | Counter | Arquivos capturados do WhatsApp (labels: `media_type` ∈ image/document/audio/video/other, `direction` ∈ inbound/outbound) |
| `crm_files_downloads_total` | Counter | Downloads bem-sucedidos (label: `media_type`) |
| `crm_files_stored_bytes_total` | Counter | Bytes totais capturados desde o boot (para monitorar crescimento do storage) |
| `specialist_responses_total` | Counter | Respostas de especialistas IA |
| `specialist_response_duration_seconds` | Histogram | Latência de resposta do especialista |
| `crm_ai_reset_commands_total` | Counter | Conversas reiniciadas via `/reset` (labels: tenant_id, specialist_id, source=command\|playground) |
| `crm_ai_playground_messages_total` | Counter | Mensagens injetadas pelo playground dev (label: tenant_id) |
| `active_whatsapp_sessions` | Gauge | Sessões WhatsApp ativas |
| `db_query_duration_seconds` | Histogram | Latência de queries ao banco |

Endpoint: `GET /metrics` (Prometheus format)

### Traces (OpenTelemetry)

- tracing distribuído para rastrear requests end-to-end
- span por: handler HTTP → caso de uso → repositório → provider externo
- propagação de trace context entre componentes
- trace_id e span_id incluídos nos logs (correlação)
- útil para diagnosticar latência e gargalos

Spans obrigatórios:
- request HTTP (entry point)
- caso de uso (business logic)
- query ao banco
- chamada ao WhatsApp (envio de mensagem)
- chamada ao especialista IA

## Dashboards (Grafana)

Dashboards sugeridos:

| Dashboard | Conteúdo |
|-----------|----------|
| Overview | requests/s, latência p50/p95/p99, taxa de erro |
| WhatsApp | mensagens recebidas/enviadas, sessões ativas, erros |
| Leads/Kanban | leads criados, movimentações, automações executadas |
| Especialistas | respostas, latência, erros |
| Infraestrutura | uso de CPU, memória, conexões ao banco |

## Alertas sugeridos

| Alerta | Condição |
|--------|----------|
| Alta taxa de erro HTTP | > 5% de requests com status 5xx em 5 min |
| Latência alta | p95 > 2s em 5 min |
| WhatsApp desconectado | sessão inativa por > 2 min |
| Fila de mensagens crescendo | mensagens pendentes > threshold |
| Banco lento | p95 query > 500ms |
| Especialista falhando | taxa de erro > 10% |

## Implementação

### Middleware de instrumentação

- middleware Gin que registra automaticamente:
  - métricas HTTP (requests, latência, status)
  - trace span para cada request
  - request_id no contexto
- injetar tenant_id e user_id nos logs após autenticação

### Health e readiness

- `GET /health` — healthcheck básico (app está viva)
- `GET /ready` — readiness (app + banco + WhatsApp prontos)
- `GET /metrics` — métricas Prometheus

### Docker compose

- Prometheus para coleta de métricas
- Grafana para dashboards
- configuração pronta no `docker-compose.dev.yml.dist`

## Regras

- todo endpoint HTTP deve ter métricas de latência e contagem
- todo log deve ter request_id, tenant_id e user_id quando disponíveis
- nunca logar dados sensíveis
- traces devem cobrir o caminho completo (HTTP → use case → repo → external)
- dashboards devem existir antes de ir para produção
- alertas devem existir para cenários críticos

## F11 — Pagamentos (Admin)

**Métricas Prometheus** (`/metrics`):

- `pagamentos_cron_runs_total{status="success|error"}` — contadores das execuções do cron diário
- `pagamentos_cron_duration_seconds` — histograma de duração das execuções
- `pagamentos_recorrentes_gerados_total` — total de lançamentos recorrentes criados pelo cron
- `pagamentos_atualizados_atrasado_total` — lançamentos que transitaram para status `atrasado`
- `pagamentos_marcados_pago_total` — cliques em "marcar como pago" (admin)
- `pagamentos_lancados_avulso_total` — lançamentos avulsos criados
- `pagamentos_cancelados_total` — lançamentos cancelados

**Traces OpenTelemetry** (tracer `pagamentos`):

- `RegisterManualPayment`, `MarkPaymentAsPaid`, `CancelPayment` com atributos `tenant_id`, `payment_id`, `user_id`, `valor_cents`
- `GenerateRecurringPayments`, `RefreshOverdueStatuses` com `today`, `created`, `updated`
- `GetTenantFinancialSummary` com `tenant_id`
- `billing_cron_run` (tracer `pagamentos/billing_scheduler`) com `request_id`, `generated`, `overdue`

**Logs estruturados**: o scheduler gera um `request_id = "cron-billing-<uuid>"` por execução e o usa como chave de correlação em todos os logs daquela iteração (generate + refresh).

**Dashboard**: `infra/grafana/dashboards/pagamentos.json` (UID `pagamentos-f11`). Painéis de cron health/duração, contadores diários e latência p95 das rotas admin/portal.

**Alertas sugeridos** (não configurados em código — definir no Grafana/Alertmanager conforme SLA):

- `pagamentos_cron_runs_total{status="error"}` > 0 em janelas de 10m (cron diário falhou)
- `histogram_quantile(0.95, rate(pagamentos_cron_duration_seconds_bucket[1h]))` > 60s (cron demorando demais)

## F19 — Dashboards

**Métricas Prometheus** (`/metrics`):

- `dashboard_render_duration_seconds{scope="tenant|admin"}` — histograma (default buckets) com o tempo total do handler, do request até o HTML final.
- `dashboard_load_total{scope="tenant|admin", outcome="success|error"}` — contador de carregamentos por escopo e desfecho.

**Logs estruturados (zap)**:

- `dashboard_rendered` — emitido após sucesso. Campos: `scope`, `tenant_id` (apenas tenant), `user_id`, `scope_is_user` (apenas tenant; `true` quando o caso de uso aplicou recorte por usuário), `took` (duração).
- `dashboard tenant: check owner` / `dashboard tenant: execute` / `dashboard admin: execute` — emitidos em erros, com `tenant_id` (quando aplicável) e `user_id`.

**Traces OpenTelemetry**:

- HTTP boundary (tracer `dashboard.http`): `dashboard.http.tenant`, `dashboard.http.admin` — span pai, instrumentado no handler.
- Use cases (tracer `dashboard`): `GetTenantDashboard`, `GetAdminDashboard` — criados nas Tasks 2/3 e ficam como filhos do span HTTP. O span do tenant carrega atributos `tenant_id` e `is_owner`.

**Dashboards Grafana sugeridos**:

- Latência p50/p95/p99 por escopo: `histogram_quantile(0.95, sum(rate(dashboard_render_duration_seconds_bucket[5m])) by (le, scope))`.
- Taxa de erro por escopo: `sum(rate(dashboard_load_total{outcome="error"}[5m])) by (scope) / sum(rate(dashboard_load_total[5m])) by (scope)`.
- Volume de carregamentos: `sum(rate(dashboard_load_total[5m])) by (scope)`.
