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

## F18 — Observabilidade Avançada (estado final)

### Helpers `internal/shared/observability/`

- `StartSpan(ctx, name, attrs...) (ctx, span)`: cria spans na camada de aplicação. Nomenclatura: `<module>.<usecase|engine|executor>.<action>` (ex.: `automation.engine.on_lead_event`, `permission.usecase.check`, `ai.usecase.respond`). Sempre com `defer span.End()`.
- `LoggerFromContext(ctx, base *zap.Logger) *zap.Logger`: retorna o logger base com `trace_id`/`span_id` injetados quando há span ativo no contexto. Usar no lugar de `uc.logger` dentro do caso de uso.
- `InitTracer(serviceName)`: registra o TracerProvider global. Seleciona exporter via `OTEL_EXPORTER_OTLP_ENDPOINT` — OTLP gRPC quando setado (insecure, aponta para Tempo), stdout pretty quando vazio (dev local sem Tempo).

### Registradores centrais (`metrics.go`)

- `PermissionCheckDuration` (histograma, label `scope`): observado pelo middleware `RequirePermission`.
- `SpecialistResponseDuration` (histograma, label `outcome`): observado no motor IA (`ai.usecase.respond`).

Prefixo Prometheus: métricas novas transversais usam `Namespace: "crm"`. Módulos antigos (`pagamentos`, `whatsapp`, HTTP middleware) mantêm o prefixo/convenção histórica para não quebrar dashboards existentes.

### Spans adicionados (Tasks 6–11)

| Módulo | Spans |
|--------|-------|
| automation | `automation.engine.on_lead_event`, `automation.engine.trigger_by_id`, `automation.executor.<type>` (6 tipos) |
| permission | `permission.usecase.<action>` (resolver, CRUD de grupos/perfis/funnels) |
| auth | `auth.usecase.<action>` (login, invites, tenants, load balance) |
| notification | `notification.usecase.<action>` (create/list/count_unread/mark_read/mark_all_read/preferences) |
| funnel | `funnel.usecase.<action>` (move_lead, CRUD de funis/colunas/leads, kanban) |
| whatsapp | `whatsapp.usecase.<action>` (connect, disconnect, status, list/send/receive) |
| ai | `ai.usecase.respond` (com atributos `ai.provider`/`ai.model`) |

Spans existentes em `pagamentos/` e `dashboard/` (PascalCase) **não foram renomeados** — preservam rastros históricos.

### Métricas novas de negócio (Tasks 12–18)

- `crm_automation_execution_duration_seconds{type,outcome}` — histograma dos 6 executors (buckets 0.01–10s).
- `crm_automation_rate_limited_total{type}` — counter (registrado; sem call site ainda, aguardando rate limiter).
- `crm_load_balance_fallback_total{reason}` — counter em `load_balance_picker.fallbackToOwner`.
- `crm_permission_changes_total{scope="load_balance"}` — novo valor de label em `SetByGroup`.
- `invites_total{outcome="expired"}` — incrementado em `AcceptInvite` quando `Validate()` retorna `ErrInviteTokenExpired`.
- `crm_notifications_read_total{type}` — label `type` = `single` (MarkRead) ou `all` (MarkAllRead), 1 incremento por chamada.

### Infra (docker-compose.dev.yml.dist)

- `tempo` (grafana/tempo:2.5.0): porta 3200 HTTP, 4317 OTLP gRPC, 4318 OTLP HTTP. Retenção 7 dias.
- `alertmanager` (prom/alertmanager:v0.27.0): porta 9093. Receiver `null` em dev; Slack/email via env em prod (`ALERTMANAGER_SLACK_URL`, `ALERTMANAGER_EMAIL_TO`).
- `prometheus`: retenção 15d (`--storage.tsdb.retention.time=15d`), carrega `alerts.yml` e aponta `alertmanager:9093`.
- `grafana`: datasource Tempo provisionado (`infra/grafana/provisioning/datasources/tempo.yml`).

### Dashboards (`infra/grafana/dashboards/`)

| Dashboard | UID | Foco |
|-----------|-----|------|
| overview | `crm-overview` | HTTP requests/s, p95/p99, 5xx ratio, top endpoints |
| whatsapp | `crm-whatsapp` | mensagens in/out, latência de processamento |
| leads-kanban | `crm-leads-kanban` | automações por tipo, rate-limited, taxa de erro |
| especialistas | `crm-especialistas` | respostas/min, p95/p99, taxa de erro |
| equipe | `crm-equipe` | convites por outcome, permissões por scope, load balance fallbacks |
| pagamentos | `pagamentos-f11` | (F11 pré-existente) |
| banco | `crm-banco-f26` | pool de conexões (in use/idle/open/max), esperas e duração de espera, saturação, p95/p99 do especialista |

### Alertas (`infra/prometheus/alerts.yml`) e runbooks

Regras validadas via `promtool test rules` em CI (`.github/workflows/ci.yml`):

| Alerta | Severidade | Runbook |
|--------|------------|---------|
| `HighHTTPErrorRate` | critical | [http-5xx-alto](../operacoes/runbooks/http-5xx-alto.md) |
| `HighHTTPLatency` | warning | [http-latencia-alta](../operacoes/runbooks/http-latencia-alta.md) |
| `SpecialistFailing` | critical | [especialista-falhando](../operacoes/runbooks/especialista-falhando.md) |
| `AutomationFailing` | warning | [automacao-falhando](../operacoes/runbooks/automacao-falhando.md) |

`WhatsAppDisconnected` e `SlowDatabase` estavam previstos no escopo original mas foram descartados porque as métricas-fonte (`crm_whatsapp_sessions_active`, `crm_db_query_duration_seconds`) não existem — follow-up separado.

## F26 — Instrumentação de banco (gargalo intermitente)

Para diagnosticar gargalos intermitentes de banco sem chutar a correção, foram
adicionados quatro instrumentos. Nenhum altera comportamento de negócio.

### Métricas do pool de conexões (`/metrics`)

`internal/shared/observability/RegisterDBStats` registra o `DBStatsCollector` do
`client_golang`, expondo `go_sql_*` (label `db_name`):

| Métrica | Uso |
|---------|-----|
| `go_sql_open_connections` / `go_sql_in_use_connections` / `go_sql_idle_connections` | ocupação do pool |
| `go_sql_max_open_connections` | teto do pool (hoje 25) |
| `go_sql_wait_count_total` | **nº de vezes que um request esperou por conexão** |
| `go_sql_wait_duration_seconds_total` | **tempo total aguardando conexão** |

PromQL úteis:

```promql
# Saturação do pool (0..1) — perto de 1 = exaustão iminente
max(go_sql_in_use_connections / go_sql_max_open_connections)

# Esperas por conexão por segundo (deveria ser ~0; >0 sustentado = gargalo)
rate(go_sql_wait_count_total[5m])

# Tempo médio aguardando uma conexão (s)
rate(go_sql_wait_duration_seconds_total[5m]) / rate(go_sql_wait_count_total[5m])
```

### Slow-query log (Gorm + zap)

`DB_SLOW_QUERY_THRESHOLD_MS` (default 200; `0` desativa). Queries acima do limiar
são logadas em `Warn` ("slow query") com SQL truncado, duração, linhas e contexto
(`request_id`, `tenant_id`). Erros de query (exceto `ErrRecordNotFound`) viram `Error`.

### Tracing por query (spans `gorm.*`)

`database.EnableQueryTracing` emite um span OTel por operação (`gorm.Query`,
`gorm.Create`, …), filho do span do request, com `db.statement` (SQL com
placeholders, **sem valores** → sem PII), `db.sql.table` e `db.rows_affected`.
Permite ver, no trace, qual query consome o tempo.

### pprof (`PPROF_ENABLED`, default false)

`/debug/pprof/*` atrás de auth + admin, para investigar gargalos fora do banco
(CPU, alocação, goroutines). Ver `rest/00-health.http`.

### Dashboard

| Dashboard | UID | Foco |
|-----------|-----|------|
| banco | `crm-banco-f26` | pool (in use/idle/open/max), esperas/duração de espera, saturação, p95/p99 do especialista |

> Com `go_sql_wait_*` agora existente, o alerta `SlowDatabase` (antes descartado
> por falta de métrica-fonte) pode ser reavaliado após a causa-raiz da F26 ser
> confirmada.

### Padrões para novas features

1. **Spans**: toda função pública de use case que recebe `context.Context` deve abrir um span com `observability.StartSpan`. Nome: `<module>.usecase.<action>` em snake_case. Atributos: `tenant.id`, `user.id`, entidade.id quando disponíveis no input.
2. **Logger**: dentro do use case, obter via `observability.LoggerFromContext(ctx, uc.logger)` para que os logs carreguem `trace_id`/`span_id`.
3. **Histograma de duração**: operação de negócio relevante (execute, respond, check, query externa) deve ter um histograma `<dominio>_<action>_duration_seconds{outcome}` com observação via `defer` para capturar também o caminho de erro.
4. **Counter de evento**: eventos discretos (sent/accepted/expired/blocked/…) viram counter `<dominio>_<evento>_total{label}` com labels de cardinalidade baixa.
5. **Alerta novo**: regra em `alerts.yml` com `runbook_url` apontando para `docs/operacoes/runbooks/<nome>.md` + teste em `alerts_test.yml` com par disparar/não-disparar.
