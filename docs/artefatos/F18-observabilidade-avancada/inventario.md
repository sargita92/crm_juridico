# F18 — Inventário de observabilidade existente

**Data**: 2026-04-24
**Origem**: `grep -rnE 'promauto\.New|prometheus\.(Counter|Histogram|Gauge)Opts|otel\.Tracer|tracer\.Start'`

## Métricas já registradas

### Com `Namespace: "crm"` (seguem o padrão desejado)
- `internal/notification/infrastructure/metrics.go` — `crm_notifications_delivered_total`, `crm_notifications_sse_events_emitted_total`, gauge de conexões SSE
- `internal/auth/infrastructure/metrics.go` — counters + histograma com prefix `crm_`

### Sem Namespace (nomes diretos, sem prefixo `crm_`)
- `internal/shared/middleware/prometheus.go`:
  - **`http_requests_total{method,path,status}`** ← usado pelos alertas F18
  - **`http_request_duration_seconds{method,path,status}`** (bucket)
- `internal/pagamentos/infrastructure/metrics.go` — `CronRunsTotal`, `RecorrentesGeradosTotal`, etc. (F11)
- `internal/whatsapp/application/metrics.go` — `messages_received_total`, `messages_sent_total`, `message_processing_duration_seconds`
- `internal/ai/application/metrics.go` — vários counters + histograma do motor IA (F16)
- `internal/automation/infrastructure/metrics.go` — counters F09
- `internal/permission/infrastructure/metrics.go` — counters F08
- `internal/files/infrastructure/metrics.go` — counters F14
- `internal/dashboard/infrastructure/metrics.go` — histograma + counter F19

### Implicação para as regras de alerta (Task 21)
**Ajustar** os nomes de métrica nas regras:

| Plano original (ajustar) | Nome real no código |
|---|---|
| `crm_http_requests_total` | `http_requests_total` |
| `crm_http_request_duration_seconds_bucket` | `http_request_duration_seconds_bucket` |
| `crm_whatsapp_sessions_active` | **NÃO EXISTE** — investigar; atualmente o código tem apenas counters de mensagens. Gauge de sessão ativa precisa ser criado no Step 2/3 ou deixado como out-of-scope |
| `crm_db_query_duration_seconds_bucket` | **NÃO EXISTE** — investigar; pode precisar adicionar via GORM callbacks ou remover essa regra |
| `crm_specialist_responses_total` | Verificar nomes exatos em `internal/ai/application/metrics.go` |
| `crm_automation_executions_total` | Verificar em `internal/automation/infrastructure/metrics.go` |

## Spans já existentes

### Com tracer nomeado por módulo (padrão próximo ao desejado)
- `pagamentos/application/tracer.go`: `otel.Tracer("pagamentos")`, spans em PascalCase (`RegisterManualPayment`, `CancelPayment`, `MarkPaymentAsPaid`, `RefreshOverdueStatuses`, `GetTenantFinancialSummary`, `GenerateRecurringPayments`)
- `pagamentos/infrastructure/billing_scheduler.go`: tracer `pagamentos/billing_scheduler`, span `billing_cron_run`
- `dashboard/application/tracer.go`: `otel.Tracer("dashboard")`, spans `GetAdminDashboard`, `GetTenantDashboard`
- `dashboard/interfaces/http/tracer.go`: `otel.Tracer("dashboard.http")`
- `auth/infrastructure/load_balance_picker.go`: tracer `crm.load_balance`, span `load_balance.pick`

### No padrão `<module>.<action>` em handlers HTTP
- `notification/interfaces/http/*`: `notification.stream.emit`, `notification.badge`, `notification.dropdown`, `notification.page.list`, `notification.page.render`
- `auth/interfaces/http/page_handler.go`: `auth.page.users`, `auth.page.users_table`, `auth.page.invite_modal`, `auth.page.create_invite`, `auth.page.user_permissions_modal`, `auth.page.set_user_permissions`, `auth.page.user_whatsapp_modal`, `auth.page.set_user_whatsapp`, `auth.page.redirect_users`
- `auth/module_handlers.go`: `auth.users.list`, `auth.users.remove`, `auth.users.set_whatsapp`
- `files/interfaces/http/handler.go`: `files.list_page`, `files.list_fragment`, `files.preview`, `files.download`, `files.thumbnail`, `files.lead_summary`
- `permission/interfaces/http/handler.go`: `permission.list`

### Módulos que NÃO têm spans em camada de aplicação (alvo do F18)
- `internal/automation/application/*` — nenhum span de UC encontrado
- `internal/auth/application/invite_*.go` — sem spans (só handlers HTTP têm)
- `internal/auth/application/manage_load_balance.go` — sem span em UC
- `internal/permission/application/*` — sem spans em UC (só handler)
- `internal/notification/application/*` — sem spans em UC (só handlers)
- `internal/funnel/application/*` — verificar
- `internal/whatsapp/application/*` — verificar
- `internal/ai/application/*` — verificar (motor IA)

### Implicação para tasks 6-11

- **Não renomear spans existentes**: `pagamentos` (PascalCase) e `dashboard` (PascalCase) continuam como estão. Renomear quebra rastros históricos e não agrega.
- **Novos spans em UCs seguem `<module>.usecase.<action>`** conforme o plano.
- **Módulo especialista é `internal/ai/`** (não `internal/specialist/` nem `internal/especialista/`). Task 11 deve apontar para `internal/ai/application/`.

## Estrutura `internal/` relevante

```
internal/
  ai/              application/, infrastructure/, domain/, interfaces/http/
  auth/            application/, infrastructure/, domain/, interfaces/http/
  automation/      application/, infrastructure/, domain/, interfaces/http/
  dashboard/       application/, infrastructure/, domain/, interfaces/http/
  files/           application/, infrastructure/, domain/, interfaces/http/
  funnel/          application/, infrastructure/, domain/, interfaces/http/
  notification/    application/, infrastructure/, domain/, interfaces/http/
  pagamentos/      application/, infrastructure/, domain/, interfaces/http/
  permission/      application/, infrastructure/, domain/, interfaces/http/
  shared/          middleware/, observability/, testhelper/, eventbus/
  whatsapp/        application/, infrastructure/, domain/, interfaces/http/
```

## Decisões finais para o plano (aplicar antes/durante Task 21)

1. Regras de alerta usam nomes REAIS: `http_requests_total`, `http_request_duration_seconds_bucket` — sem prefixo `crm_`.
2. `WhatsAppDisconnected` precisa de uma métrica nova de sessão (gauge `whatsapp_session_active` ou similar). Criar isso no Step 2/3 OU remover o alerta do escopo F18 e tratar em follow-up.
3. `SlowDatabase` precisa de histograma de query. Criar via callback GORM OU remover do escopo.
4. Os nomes exatos de `specialist_responses_total` e `automation_executions_total` precisam ser validados em `internal/ai/application/metrics.go` e `internal/automation/infrastructure/metrics.go` — ler esses arquivos antes de escrever as regras.
5. Spans já existentes em `pagamentos` e `dashboard` (PascalCase) ficam intocados.
6. Module "especialista" = **`internal/ai/`**.
