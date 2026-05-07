# Changelog

Registro histórico de entregas do projeto.

---

## [2026-05-07] F21 — Saneamento Técnico (Faxina Mecânica)

Rodada one-shot que zera dívida técnica mecânica e estabelece o processo recorrente de manutenção. Foco em automação, não em refactor.

**Highlights**
- `make audit` orquestra 6 sub-alvos: `audit-vuln`, `audit-vet`, `audit-lint`, `audit-cover`, `audit-deps`, `audit-todos`.
- `.golangci.yml` configurado (errcheck, govet, ineffassign, staticcheck, unused, gosimple, misspell, goimports). `govet.shadow` desabilitado por escolha de estilo (codebase usa `err :=` em escopos internos).
- `scripts/audit-vuln.sh` filtra IDs aceitas em `.audit-accepted-vulns.txt` (com justificativa e revisão trimestral).
- `scripts/audit-cover.sh` em **dois modos**: `-short` (dev, informativo, exclui pacotes que dependem de testcontainers) e `AUDIT_INTEGRATION=1` (CI, enforce com global ≥ 85% e pacote ≥ 80%).
- `.github/workflows/ci.yml` ganhou 7 jobs: `build`, `vet`, `lint`, `vuln`, `test-short`, `test-integration`, `coverage`.
- Issue template `.github/ISSUE_TEMPLATE/manutencao-trimestral.md` com checklist alinhado a `docs/processo/manutencao-tecnica.md`.

**Resultados**
- Vulnerabilidades: **14 → 0** não-aceitas. Toolchain Go 1.26.1 → 1.26.3 (resolveu 11 stdlib); `golang.org/x/net` v0.52 → v0.53 (resolveu GO-2026-4918); 2 vulns `docker/docker` aceitas com justificativa (transitivas via testcontainers, sem fix upstream, sem call path produtivo).
- Lint: **199 → 0** issues. Auto-fix em misspell (79) e goimports (57); manual em staticcheck `nil` context (17), unused (10), errcheck (2), ineffassign (1).
- Testes: **1911 → 1921** verdes em `-short`. Resolveu regressão herdada do redesign UI (commit `af3b174`) em `internal/audit/interfaces/http` (10 testes).
- Bug de TestMain pré-existente: 20 pacotes detectavam `-short` por igualdade exata de `os.Args`, mas `go test -short` propaga `-test.short=true`. Corrigido com `strings.HasPrefix`. Testes de integração agora pulam corretamente em `-short`.
- Deps diretas atualizadas: `go-sql-driver/mysql`, `mattn/go-sqlite3`, `docker/go-connections`, `testcontainers-go` (v0.41 → v0.42, com ajuste de assinatura `wait.ForSQL`), `whatsmeow`, `zap`.

**Decisões consolidadas**
- Updates major ficam fora do escopo (viram features dedicadas se necessário).
- Refactors arquiteturais ficam fora do escopo (viram F23+).
- `govet.shadow` desabilitado: padrão idiomatico do codebase.
- Vulnerabilidades aceitas vão em `.audit-accepted-vulns.txt` com justificativa e são revisadas trimestralmente.
- Cobertura tem duas metas (global ≥ 85%, pacote ≥ 80%) para evitar média enganosa.
- Pacotes wiring/main isentos da meta (`cmd/...`, `internal/shared/module`).

**Artefatos**
- PO/UIUX/Arquiteto: `docs/artefatos/F21-saneamento-tecnico/`
- Relatório inicial: `docs/manutencao/audit-inicial-2026-05-07.md`
- Relatório final: `docs/manutencao/audit-final-2026-05-07.md`
- Processo recorrente: `docs/processo/manutencao-tecnica.md`

---

## [2026-04-24] F12 — Logs (Admin)

Auditoria centralizada de ações admin e de segurança: login (sucesso e falha), CRUD de tenants, CRUD de usuários admin, bloqueio/desbloqueio de tenant e alteração de permissões. Consulta com filtros (tenant, usuário, ação, período) + paginação + detalhe. Rota `/admin/logs` restrita ao admin global; usuários não-admin recebem 404 genérico (sem revelar a existência da rota). Sem exportação CSV no MVP; retenção ilimitada.

**Highlights**
- Novo módulo `internal/audit/` (DDD + Clean Architecture): `domain` (entidade `AuditLog`, enum `Action` com 14 ações, value object `Filter`, erros), `application` (`RegisterAuditLogUseCase`, `ListAuditLogsUseCase`, `GetAuditLogUseCase`, `Publisher` com `NoopPublisher` e default que engole erro com WARN, helper `BuildDiff`), `infrastructure` (`GormAuditLogRepository` com 4 índices, métricas Prometheus, adapters `tenant_lister` e `admin_user_lister`), `interfaces/http` (`Handler` para fragmentos HTMX, `PageHandler` para full page, OWASP suite).
- Migration `000056_create_audit_logs` com índices `idx_audit_created_at`, `idx_audit_tenant_created`, `idx_audit_user_created`, `idx_audit_action_created`. JSON metadata. Imutável (sem update/delete).
- Captura via `AuditPublisher` injetável em casos de uso (não middleware HTTP) — eventos auditáveis são de domínio. Falha de auditoria **não** quebra a operação de negócio (WARN + métrica `status="error"`).
- Integração com 3 módulos: `auth` (`login.success`/`login.failure`/`logout` + claim `Email` no JWT), `tenant` (5 actions com diff em update e motivo em block/unblock), `auth.ManageUsers` + `permission` (5 actions de `user_admin.*` + `permission.changed` quando alvo é admin).
- Sanitização: chaves proibidas (`password`, `password_hash`, `token`, `secret`, `authorization`, `hash`) removidas de `Metadata` no domínio (`IsForbiddenMetadataKey`) e ignoradas em `BuildDiff`. Diff também ignora `updated_at`.
- Middleware `AdminOr404` + `AdminPageAuth`: não autenticado → 302 redirect HTML para `/admin/login?return=...`; autenticado mas não-admin → 404 genérico (mesma página de id inexistente, sem vazar existência — OWASP A01).
- Filtros via querystring com **clamp** de `page_size` (default 10, max 100), rejeição 400 em `action` fora do enum, rejeição 400 em `from > to`. Botão "Voltar" no detalhe preserva filtros via `?return=...`.
- UI HTMX consistente com painel admin: tabela em desktop, cartões em mobile, date pickers nativos, escape automático via `html/template`. Sem JavaScript custom.
- 2 métricas Prometheus: `crm_audit_logs_registered_total{action,status}` (counter) e `crm_audit_logs_list_duration_seconds` (histogram). Spans OTel `audit.list`/`audit.get`. Smoke test em `internal/shared/observability/metrics_registered_test.go`.

**Decisões de produto**
- Escopo reduzido a admin/segurança no MVP (sem CRUD operacional de tenant — leads, kanban, produtos, automações, arquivos ficam fora).
- Retenção ilimitada (sem expurgo no MVP).
- Sem exportação CSV (cortado por YAGNI — F18 já cobre observabilidade operacional via Grafana).
- Logs imutáveis (sem botão editar/excluir nas telas).
- `permission.changed` só para alvo admin no MVP (`tenant_id` sempre NULL para esta ação).
- `access.denied` (auditoria de tentativa de acesso negado) ficou fora do MVP — pode ser reativado em revisão de Segurança futura.

**Cobertura final** (todos ≥ 80%):
- `internal/audit/domain`: 100%
- `internal/audit/application`: 91.9%
- `internal/audit/infrastructure`: ~87%
- `internal/audit/interfaces/http`: 89.4%
- `internal/shared/middleware/admin_or_404.go`: 100%

**Entregáveis**
- 10 steps, 10 commits atômicos na branch `feature/F12-logs-admin` (`654448b` → `a132f9b`).
- Artefatos: `docs/artefatos/F12-logs-admin/{po-stories,uiux-wireframes,arquiteto-design,qa-cenarios}/v1.md` + `status.md`.
- `rest/11-logs.http` cobre listagem (sem filtro / com filtro / paginação), detalhe e cenários OWASP (sem token → 302, id inexistente → 404, token tenant → 404 genérico).

---

## [2026-04-24] F18 — Observabilidade Avançada

Fecha o gap de instrumentação deixado por F08/F09: spans em camada de aplicação, histogramas de duração, dashboards Grafana novos, regras de alerta com testes promtool no CI e runbooks operacionais.

**Highlights**
- Novos helpers em `internal/shared/observability/`: `StartSpan(ctx, name, attrs...)`, `LoggerFromContext(ctx, base)` e `InitTracer` com suporte a OTLP gRPC via `OTEL_EXPORTER_OTLP_ENDPOINT` (fallback stdout em dev).
- Spans em 7 módulos — `automation` (engine + 6 executors), `permission`, `auth`, `notification`, `funnel`, `whatsapp`, `ai.usecase.respond` — todos com atributos `tenant.id`/`lead.id`/`user.id` quando aplicável.
- 3 histogramas novos: `crm_automation_execution_duration_seconds{type,outcome}`, `crm_permission_check_duration_seconds{scope}`, `crm_specialist_response_duration_seconds{outcome}`.
- 4 counters novos: `crm_load_balance_fallback_total{reason}`, `crm_notifications_read_total{type}`, `crm_automation_rate_limited_total{type}` (registrado sem call site) e novo outcome `expired` em `invites_total`, mais scope `load_balance` em `permission_changes_total`.
- Infra: Tempo e Alertmanager adicionados ao `docker-compose.dev.yml.dist`. Retenção 15d (Prometheus) / 7d (Tempo). Datasource Tempo provisionado no Grafana.
- 5 dashboards em `infra/grafana/dashboards/` — overview (atualizado), whatsapp, leads-kanban, especialistas, equipe.
- 4 regras em `infra/prometheus/alerts.yml` — `HighHTTPErrorRate`, `HighHTTPLatency`, `SpecialistFailing`, `AutomationFailing` — com testes `promtool test rules` rodando no CI (`.github/workflows/ci.yml`).
- 4 runbooks em `docs/operacoes/runbooks/` + README índice.
- Cobertura de `internal/shared/observability`: **86.2%**.

**Decisões técnicas**
- Trace exporter: Grafana Tempo via OTLP gRPC; stdout quando `OTEL_EXPORTER_OTLP_ENDPOINT` vazio.
- Alertmanager dev: receiver `null`; Slack/email via `ALERTMANAGER_SLACK_URL`/`ALERTMANAGER_EMAIL_TO` em prod.
- Namespace Prometheus: métricas transversais novas com prefixo `crm_`; middleware HTTP e módulos legados (`pagamentos`, `whatsapp`) mantêm o que já publicam.
- Spans pré-existentes em `pagamentos` e `dashboard` não foram renomeados (preservam rastros históricos).
- Nomenclatura dos spans novos: `<module>.<usecase|engine|executor>.<action>` em snake_case.

**Fora do escopo (follow-up)**
- Alertas `WhatsAppDisconnected` e `SlowDatabase` removidos — dependem de métricas inexistentes (gauge de sessão WhatsApp e histograma de query Gorm). Tickets separados para criar essas métricas.
- `automation_rate_limited_total` está registrado mas não incrementado — módulo `automation` ainda não tem rate limiter.
- Validação ponta-a-ponta do stack Tempo + Alertmanager no compose é smoke manual; CI valida apenas as regras via promtool.

**Entregáveis**
- 23 tasks do plano, 26 commits na branch `feature/F18-observabilidade-avancada`.
- Build/vet limpos; `go test ./internal/...` verde (66 pacotes).

---

## [2026-04-19] F19 — Dashboards (Tenant + Admin)

CRM ganha dois dashboards: tenant (5 blocos: funil, WhatsApp, responsáveis, tempo no funil, produtos) com filtro automático por usuário responsável quando o user é comum; admin (6 blocos: tenants, uso, health, infraestrutura, especialistas, financeiro com dados reais de F11). Chart.js via CDN, HTMX para refresh manual via fragmento, métricas Prometheus de duração e load_total, tracing OTel.

**Highlights**
- Novo módulo `internal/dashboard` (DDD + Clean Architecture, 5 packages, 72 testes incluindo 9 OWASP).
- Cobertura: application 98.3%, infrastructure 84.6%, interfaces/http 84.6% (todos ≥ 80%).
- `PaymentRepository.GlobalSummary` adicionado em F11 (consumo cruzado F11 → F19).
- Templates self-contained com Chart.js (sem SRI por enquanto — TODO P2).
- Métricas Prometheus: `dashboard_render_duration_seconds{scope}`, `dashboard_load_total{scope, outcome}`.
- Spans OTel: `dashboard.http.{tenant,admin}` (parent dos spans dos UCs).

**Decisões de produto**
- Bloco 6 admin (Financeiro) consome dados reais de F11 — opção B aprovada em 2026-04-19.
- Usuário comum vê **todos** os 5 blocos do tenant filtrados por `responsible_user_id` (não apenas o Bloco 3).
- `Top10Active` admin inclui tenants `blocked`/`inactive` com leads históricos (decisão: métrica é volume, não status operacional).
- `Qualifications` no Bloco 5 admin fica em 0 com TODO(F05) até o módulo de qualificações expor o contador.

**Entregáveis**
- Sidebar do tenant ganha item "Dashboard" como primeiro link.
- `rest/16-dashboard.http` cobre páginas + fragmentos HTMX + casos OWASP (401 sem auth, 403 user comum em /admin, SQLi/XSS em querystring).

---

## [2026-04-19] F11 — Pagamentos (Admin)

Controle financeiro multi-tenant com cron diário de recorrentes, UI admin e portal tenant read-only. Step 3 da spec original (gateway externo Stripe/Mercado Pago) deferido para F11.1.

- **Domínio novo `internal/pagamentos/`** (DDD completo): `domain` (entidade `Payment` com enums `PaymentType`/`PaymentStatus`, `BillingConfig` do tenant com enum `Plan {mensal|anual|vitalicio|externo}`, `BrazilHolidayCalendar` thread-safe com cache por ano + Páscoa Meeus/Jones/Butcher, ports `PaymentRepository`/`TenantBillingRepository`/`HolidayCalendar`), `application` (UCs `RegisterManualPayment`/`MarkPaymentAsPaid`/`CancelPayment`, `ListTenantPayments`/`ListAllPayments`/`GetTenantFinancialSummary`, `GenerateRecurringPayments`/`RefreshOverdueStatuses` + helpers `Clock`/`IDGenerator`), `infrastructure` (`GormPaymentRepository` + `GormTenantBillingRepository` lendo direto de `tenants` sem acoplar ao pacote `tenant`, `BillingScheduler` com `robfig/cron/v3`), `interfaces/http` (admin + portal tenant).
- **Migrations**: `054_alter_tenants_billing` (plano, valor_cobranca_cents BIGINT, dia_vencimento TINYINT, data_inicio_cobranca DATE, exibir_pagamentos BOOL) e `055_create_payments` (tabela payments com tipo/status/competência/vencimento + FK tenants + auditoria `paid_by_user_id`/`paid_at`/`cancelled_*` + unique `(tenant_id, tipo=recorrente, competencia)`).
- **Extensão tenant**: `Tenant` ganha `SetBillingConfig` (sem acoplar `tenant/domain` a `pagamentos/domain`); UC `UpdateTenant` valida combinações via `pagamentos/domain.NewBillingConfig` antes de persistir. Form de edição do tenant com fieldset "Cobrança" e JS de toggle (habilita valor/dia/data apenas para mensal/anual).
- **Cron**: `BillingScheduler` com `BILLING_CRON` (default `0 3 * * *`), TZ `BILLING_TZ` (default `America/Sao_Paulo`), carência `BILLING_GRACE_DAYS` (default 1) em dias úteis. Gera competências faltantes respeitando `DataInicioCobranca` (skip futuro) e prorroga vencimento para próximo dia útil via calendário BR. Idempotente via `ExistsRecorrente(tenant, comp)` com `competencia = DATE(?)`. Fix de DATE-roundtrip: tenant `toModel` normaliza `DataInicioCobranca` para local-midnight preservando Y/M/D (evita duplicar competências por shift TZ com DSN `loc=Local`).
- **UI admin**: menu global `/admin/pagamentos` com filtros + aba `/admin/tenants/:id/pagamentos` com resumo financeiro (badge `em_dia`/`pendente`/`atrasado`/`sem_cobranca` + total pago no ano + pendente + atrasado). Modal HTMX para lançamento avulso; botões pagar/cancelar via `hx-post` com swap `outerHTML` da linha. Templates em `web/templates/pagamentos/` + partials + CSS em `web/static/css/main.css`.
- **Portal tenant**: rota `/pagamentos` read-only com middleware `PortalAccessChecker` que valida (1) plano cobrável + `exibir_pagamentos=true` (senão 404) e (2) permissão via `permission.Resolver` — que inclui owner bypass + admin bypass + `payments:view`. Tenants vitalício/externo e com flag desabilitada não veem o menu nem acessam o endpoint.
- **Permissões**: novo resource `payments` com ação `view` em `ValidPermissions`.
- **Observabilidade**: 7 métricas Prometheus (`pagamentos_cron_runs_total{status}`, `pagamentos_cron_duration_seconds`, `pagamentos_recorrentes_gerados_total`, `pagamentos_atualizados_atrasado_total`, `pagamentos_marcados_pago_total`, `pagamentos_lancados_avulso_total`, `pagamentos_cancelados_total`); traces OTel no tracer `pagamentos` em todos UCs + `billing_cron_run` com `request_id` de correlação; dashboard Grafana `infra/grafana/dashboards/pagamentos.json`.
- **Testes**: 152 testes em `internal/pagamentos/...` — domain 98.2%, application 92.4%, infrastructure 82.7% (integração MySQL via testcontainers), interfaces/http 80.8% (inclui 14 testes OWASP: 401/403/404, isolamento tenant, SQLi em status/data/tenant_id e XSS em descricao).
- **Wiring**: `cmd/api/main.go` instancia o módulo, conecta o `permissionMod.Resolver` tardiamente (quebra ciclo), inicia o scheduler e para no shutdown graceful. `rest/10-pagamentos.http` com todos os endpoints + exemplos de erros. Sidebars admin e tenant ganham item "Pagamentos".
- **Loader de templates**: `web/templates/**/*.html` agora carregado via múltiplos globs (1, 2 e 3 níveis) porque `filepath.Glob` do Go não suporta `**` recursivo. Aplica-se a `cmd/api/main.go` e `internal/shared/testhelper/mysql.go`.

---

## [2026-04-19] F14 — Arquivos por Lead

Captura automática de mídia do WhatsApp (imagem/documento/áudio/vídeo/sticker) com aba dedicada `/tenant/files` e integração com o detalhe do lead.

- **Módulo novo `internal/files/`** (DDD completo): `domain` (entidade `File`, `MediaType`/`Direction` enums, `SanitizeFileName`, `DetectMediaType`, ports `FileRepository`/`Storage`/`LeadLookup`), `infrastructure` (`GormFileRepository` com LIKE seguro + `LocalDiskStorage` com resolução confinada à raiz e escrita atômica via temp+rename), `application` (`StoreFileUseCase`, `ListFilesUseCase`, `GetFileUseCase`, `DownloadFileUseCase`, `LeadFilesSummaryUseCase`, `WhatsAppFileAdapter`), `interfaces/http` (Handler + 6 endpoints).
- **Integração whatsapp**: `whatsmeow_provider` estendido para detectar `ImageMessage|DocumentMessage|AudioMessage|VideoMessage|StickerMessage` e baixar bytes via `client.Download`; `MessageType` enum cresce para `image/document/audio/video/sticker/other` (content opcional para mídia); `ReceiveMessageUseCase.SetFileStorer` persiste `Message` + invoca `FileStorer.StoreInbound` (best-effort — falha de storage não derruba ingestão).
- **Integração funnel**: `FileLeadLookupAdapter` resolve `lead_id` a partir da conversa respeitando tenant; `lead_drawer.html` ganha seção "Arquivos (N)" carregada via HTMX lazy-load com os 6 mais recentes + link "Ver todos".
- **UI/HTMX**: aba "Arquivos" na sidebar do tenant; página `/tenant/files` com filtros (busca, tipo, período — presets `today`/`7d`/`30d`/`custom`, `lead_id` pré-filtro do detalhe do lead) em um único `<form>` com debounce e `hx-push-url`; drawer lateral para preview (imagem inline, player `<audio>`, ícone+metadados para outros); lightbox full-screen; `files.js` com 3 funções mínimas (`openFileDrawer`, `closeFileDrawer`, `openLightbox` + ESC handler).
- **Segurança**: download com `Content-Disposition: attachment; filename*=UTF-8''<escaped>` + `X-Content-Type-Options: nosniff` + `Cache-Control: private`; thumbnail **só** para `media_type=image` (404 para outros — evita polyglot inline); cross-tenant retorna 404 (não 403, para não vazar existência); LIKE com `ESCAPE '\\'` + `escapeLike` (wildcards `%`/`_` do usuário tratados como literais); storage key sempre UUID; `resolve()` recusa chaves com `..`/absolute.
- **Permissão**: novo resource `files:view` em `ValidPermissions` + UI admin (`groupPermActions` e `groupPermAvailable`); middleware `requirePerm("files", "view")` em todas as rotas.
- **Observabilidade**: métricas `crm_files_stored_total{media_type,direction}`, `crm_files_downloads_total{media_type}`, `crm_files_stored_bytes_total`; OTel spans `files.*`; logs estruturados de download (tenant_id, user_id, file_id, media_type) e falha de storage (tenant_id, conversation_id, message_id, size, mime).
- **Config**: `FILES_STORAGE_ROOT` (default `storage/files`) e `FILES_MAX_BYTES` (default 50 MB) via Viper.
- **Migrations**: `000052_create_files_table` (índices por tenant, lead, conversation, `tenant+created_at`, `tenant+media_type`) + `000053_extend_message_types` (ALTER `messages.type` para aceitar os novos enums).
- **Outbound media**: `StoreOutbound` implementado e coberto por testes unitários, mas ainda não exercitado — aguardando F06 estender `SendMessage` para mídia. Não bloqueia F14.
- **Cobertura final**: domain 100%, application 94.2%, infrastructure 91%, interfaces/http 86.9%. Suite total: 1562 curtos + 504 integração (incl. testcontainers-mysql).
- **Artefatos**: `docs/artefatos/F14-arquivos/{po-stories,uiux-wireframes,arquiteto-design,qa-cenarios,qa-validacao,seguranca-review}/v1.md` + `status.md`. `rest/12-arquivos.http` reescrito com rotas reais e casos negativos OWASP.
- **Follow-ups (não bloqueantes)**: retenção/cleanup, quota por tenant, tabela de auditoria de downloads, thumbnail com resize server-side.

---

## [2026-04-19] F10 — Fechamento de Produtos

- **Cobertura HTTP elevada** de 32.7% → 93.7% em `internal/product/interfaces/http` via `handler_flow_test.go`: happy-path + fallbacks para priority inválida, listers com erro, repo error, toggle/link/unlink/priority/associate/disassociate (admin + tenant).
- **Sub-item "especialista vinculado a produto"** reclassificado: já entregue em F16 (`SpecialistRouter` + tabela `specialist_products`), portanto marcado como concluído no spec.
- Feature marcada `concluído` no backlog e no spec; cobertura final: domain 100%, application 88.8%, http 93.7%.

---

## [2026-04-18] F09 Step 8 — Telas de Notificações (HTMX)

Fecha o loop de UX do Step 4.1: todo lead atribuído via load balance agora é visto pelo responsável em tempo real através de toast + badge + dropdown + página dedicada.

- **Sino flutuante** (`position: fixed` canto superior direito) em 10 páginas do tenant via `partials/notification_bell.html` + `partials/tenant_head.html` compartilhados. CSS dedicado em `web/static/css/notification.css`.
- **Dropdown** com 10 últimas + botão "marcar todas" + link "Ver todas" → página dedicada `/tenant/notifications` com tabs "Não lidas" / "Todas" e paginação.
- **Toast em tempo real** via SSE: `/notifications/stream` agora emite HTML fragment (toast + badge OOB swap) ao invés de JSON. HTMX ext `htmx-ext-sse@2.2.2` consome via `sse-swap="notification"` com `hx-swap="beforeend"`. `c.SSEvent` usa `gin-contrib/sse` que formata corretamente dados multi-linha (cada `\n` vira `\ndata:`).
- **Deep-link** `lead_assigned` → `/tenant/leads?open=<lead_id>`: kanban handler valida ownership via `GetLeadDetailUseCase` e carrega o drawer existente via HTMX (`hx-trigger="load"`). Cross-tenant e lead inexistente são ignorados silenciosamente (sem 404 pra evitar timing oracle).
- **PageHandler** novo em `internal/notification/interfaces/http/page_handler.go` com 4 rotas HTML (`/tenant/notifications`, `/list`, `/dropdown`, `/badge`). Late-binding do `ToastRenderer` via `Module.SetRenderer()` resolve o ciclo module ↔ template-parse (módulo construído antes de `setupRouter` parsear templates).
- **Observabilidade**: métricas `crm_notifications_delivered_total{type}`, `crm_notifications_sse_active_streams`, `crm_notifications_sse_events_emitted_total{outcome}` em `internal/notification/infrastructure/metrics.go`. Spans OTel `notification.page.render|list|dropdown|badge` + `notification.stream.emit`.
- **Cobertura**: `internal/notification/interfaces/http` em ~49% (foco nos handlers novos); `page_handler.go` individual tem funções em 53-100%. OWASP tests cobrem 401 nas 4 rotas + isolamento por user e tenant.
- **Regressão UX aceita**: 3 páginas (funnel_detail, funnel_form, group_detail) tinham `<title>` dinâmico (ex.: "Funil — {{.Funnel.Name}}") — simplificados para estático ("Funil"/"Grupo") porque o `tenant_head.html` recebe só `Title` via `dict`. O `<h1>` continua dinâmico.
- **Fora de escopo**: preferências de notificação (canal WhatsApp sem emissor), emissores dos 4 tipos ainda inativos (`lead_moved`, `lead_handoff`, `lead_qualified`, `rate_limit_reached`), som, admin area, observabilidade transversal dos demais módulos.
- Artefatos: `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/design-step8-notification-screens.md` + `plan-step8-notification-screens.md`; `rest/notifications.http`.

---

## [2026-04-18] F08 Step 4/4.1 — Load Balance integrado ao fluxo de criação de lead

Todo lead criado (HTTP manual, WhatsApp ou IA) passa a nascer com `ResponsibleUserID` preenchido automaticamente. Fecha os dois itens abertos da F08: distribuição automática entre membros do grupo responsável (Step 4) e atribuição do responsável humano no momento da criação (Step 4.1).

- **Porta `ResponsiblePicker`** (`internal/funnel/domain`) + implementação `LoadBalancePicker` (`internal/auth/infrastructure`) com 3 algoritmos:
  - `round_robin` (persistência de `LastIndex`, ordem determinística por sort lexicográfico dos membros)
  - `least_load` (contagem de leads ativos `status=open` no tenant, tiebreak determinístico)
  - `random` (uniforme via `crypto/rand`)
- **Cascata de fallback**: sem grupo / sem config ativa / config inativa / sem membros / erro de infra → owner do tenant. Lead nunca nasce órfão.
- **Regra de unicidade** ("1 grupo ativo com load balance por funil/coluna") validada proativamente em `ManageLoadBalanceUseCase.SetByGroup` via novo port `GroupColumnOverlapChecker` + adapter `permission/infrastructure`; erro `ErrActiveLoadBalanceOverlap` (mapeável a 409 no HTTP boundary).
- **Evento `EventLeadResponsibleAssigned`** publicado junto ao `EventLeadCreated` (que ganhou `responsible_user_id` no payload); `notification.Module` assina globalmente via novo `GlobalEventBus.SubscribeAll()` e cria notificação in-app `lead_assigned` para o responsável.
- **Observabilidade**: métricas `crm_lead_responsible_picker_total{algorithm,outcome}` e `_duration_seconds{algorithm}`; span OTel `load_balance.pick`; logs com níveis diferenciados (Info para fallback benigno, Warn para infra-error, Error com preservação do erro original).
- **Defesa**: `errors.Is(authdomain.ErrUserNotFound)` no filtro de membros ativos; nil-guard com `ErrPickerNotConfigured` no `CreateLeadUseCase` para falha explícita em wiring incorreto; compile-time assertion que o gorm lead repo implementa `LeadLoadCounter`.
- **Cobertura**: `funnel/application` 83.6%, `auth/application` 81.6%, funções do picker 78–100%. Suite total ~1330 testes passando em `-short`; tree integrado 1717+ com integration tests.
- **Wiring**: late-binding setter (`funnelMod.SetResponsiblePicker`) para resolver ciclo de construção auth → permission → funnel; mesma estratégia do `SetLoadBalanceOverlapChecker`.
- **Follow-ups (não bloqueantes)**: race do `LastIndex` em round-robin sob alta concorrência; índice composto `(tenant_id, status, responsible_user_id)` em `leads` para hot path do `least_load`; auditoria de cobertura de `auth/infrastructure` (41.5% pacote-total devido a gorm repos rodados só em integration).
- Artefatos: `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/design-step4-load-balance-integration.md` + `plan-step4-load-balance-integration.md`; arquivos .http atualizados em `rest/06-funis-kanban.http` e `rest/team.http`.

---

## [2026-04-18] F07 — Funis de Vendas (Kanban) — fechamento

Feature encerrada com validação QA + review de segurança (ambos aprovados):

- **QA validação (66 cenários)**: todos cobertos por testes automatizados.
  Gaps preenchidos: CT-33 (refresh de `column_entered_at` no `Lead.MoveTo`),
  CT-64 (JWT tampering — assinatura adulterada/corrompida retorna 401),
  CT-65/CT-66 (audit logs de tentativa cross-tenant e de movimentação de
  lead, validados via `zaptest/observer`).
- **Audit logging opcional** adicionado em `MoveLeadUseCase` e
  `GetLeadDetailUseCase` (`SetAuditLogger`). Emite `INFO "lead moved"`
  com tenant_id, lead_id, funnel_id, from/to column_id no sucesso, e
  `WARN "cross-tenant lead access denied"` em isolamento quebrado.
- **Security review (OWASP Top 10)**: APROVADO sem achados alto/crítico.
  Recomendações nice-to-have (rate limiting global, user_id no audit)
  ficaram para follow-up.
- **Cobertura por pacote** (todos ≥ 80%):
  - domain 89.7% · application 83.5% · infrastructure 88.6% ·
    interfaces/http 86.3%
  - infrastructure saltou de 3.5% → 88.6% com 48 novos testes de
    integração (testcontainers-go) + adapters.
  - interfaces/http saltou de 15.6% → 86.3% com 36 novos testes de
    handler cobrindo os 17 endpoints.
- **Suite total**: 1703 testes passando.
- Artefatos: `docs/artefatos/F07-funis-kanban/qa-validacao/v1.md` e
  `seguranca-review/v1.md`; backlog atualizado para `concluído`.

---

## [2026-04-18] F08 Step 6 — Telas de Equipe (HTMX)

- Novo item "Equipe" no sidebar do tenant (visível com `users:read` OU `groups:manage`)
- Rota `/tenant/team` com 2 abas: **Usuários** e **Grupos**
- Aba Usuários: lista + convites pendentes, modal de convite (link copiável),
  modal de permissões individuais (override sobre as herdadas do grupo),
  modal de WhatsApp ID, remoção (bloqueada para owner)
- Aba Grupos: lista + criação; cada grupo abre um detail com 5 seções:
  - 👥 Membros (adicionar/remover, seleciona entre usuários do tenant)
  - 🔐 Permissões (matriz resource × action)
  - 🎯 Funis atribuídos (toggle de funis do tenant)
  - 👁️ Perfis de visualização (colunas visíveis por funil)
  - ⚖️ Load Balance (algoritmo + toggle ativo)
- Backend de load balance concluído: `ManageLoadBalanceUseCase`, campo `active`
  (migration 000051), endpoints `GET/PUT /tenant/groups/:id/load-balance`
- Cross-module wiring via `AttachPermissionDeps` no auth module + accessors
  (`ListGroupsUseCase`, `ManagePermissionsUseCase`) no permission module
- 19 tests no `auth.PageHandler` (91% cobertura), 37 tests no
  `permission.PageHandler` (≥85% por função), 103 tests OWASP (401/403/tenant
  isolation), 1569 tests totais no repositório
- Artefatos: design spec + plano em
  `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/`,
  arquivo `rest/team.http` para testes manuais

---

## [2026-04-18] F17 — AI Playground e robustez do pipeline

Ferramenta interna de desenvolvimento para exercitar o motor de IA sem
envolver o WhatsApp real, mais robustez colhida no caminho:

- **Playground UI** (`/tenant/ai/playground`): sidebar de contatos + chat
  com polling de 2s, botão de reset que zera state e volta o lead à
  coluna inicial
- **Scoring-driven routing**: `ConversationEngine` consulta o `ScoringConfig`
  do especialista para qualificar/desqualificar leads automaticamente
  (veto do LLM > `TargetColumnID` explícito > threshold atingido > flow
  completo sem threshold); `ScoringConfigFinder` opcional por DI
- **WhatsApp DM-only**: filtra mensagens que não são 1:1 (groups/status
  broadcast) e usa `ToNonAD()` no JID do sender para evitar rejeição no
  send; AI processing passou a usar `context.WithoutCancel` para não
  cancelar ao final do HTTP request
- **Hardening cross-cutting**:
  - Lookup produto → funil agora é tenant-scoped (regressão:
    `TestCreateLead_ProductRoutesToOtherTenantFunnel_IsIgnored`)
  - `Lead.ProductID`/`ResponsibleUserID` e `LeadMovement.FromColumnID`
    persistem como `NULL` quando vazios (antes: empty string violava FK)
- Scripts `create-playground-lead.sh` / `test-playground.sh` e fixture
  `escritorio-teste.sql` para seed rápido

---

## [2026-04-16] F15 — Internal Tool Registry para Especialistas IA

- Domain: `ToolDefinition`, `ToolCall`, `ToolResult`, `ToolCategory`, `ParameterDef`, `Tool` interface
- `AIRequest`/`AIResponse` estendidos com `Tools`, `ToolResults`, `ToolCalls`
- `ToolRegistry` e `ToolResolver` (filtragem por especialista + steps com `ForcedTools`/`RestrictedTools`)
- Tool calling loop no `ConversationEngine` (max iterations, timeout, truncate)
- OpenAI provider com function calling nativo (provider-agnostic por design)
- 10 tools em 3 categorias:
  - Consulta: `search_leads`, `get_lead_detail`, `get_conversation_history`, `list_products`, `get_pipeline`
  - CRM: `move_lead`, `create_note`, `update_score`
  - Automação: `trigger_automation`, `switch_specialist`
- Entidade `SpecialistTool` + tabela `specialist_tools` (migration 49) + campos `forced_tools`/`restricted_tools` no step (migration 50)
- Admin UI HTMX para associação tool↔especialista em `/admin/specialists/:id/tools`
- 4 métricas Prometheus (`tool_calls_total`, `tool_call_duration_seconds`, `tool_loop_iterations`, `tool_result_truncated_total`)
- Limites configuráveis via env (`AI_TOOL_LOOP_MAX_ITERATIONS`, `AI_TOOL_EXECUTION_TIMEOUT_SECONDS`, `AI_TOOL_RESULT_MAX_LENGTH`, `AI_TOOL_CALL_MAX_PER_ITERATION`)
- Segurança: `tenantID` sempre do contexto (nunca do LLM), validação de args em todas as tools, erros retornam `ToolResult{IsError: true}` (não crasham conversa)
- 1404 testes passando, coverage core (domain+application) 92%
- Artefatos: design spec (`docs/features/F15-internal-tool-registry-design.md`), plano (`F15-internal-tool-registry-plan.md`)

---

## [2026-04-05] F02 — Autenticação e Multitenancy

- Entidade Tenant (PF/PJ, status, bloqueio) com repositório GORM
- Entidade User com bcrypt, relação N:N com Tenant via user_tenants
- Login com JWT (HS256, expiração configurável), cookie HttpOnly + SameSite Lax
- Middleware Auth (cookie/Bearer), middleware RequireTenant, TenantScope GORM
- Tela de login (HTMX, toggle senha, loading state, erro genérico)
- Tela de seleção de tenant (cards PF/PJ, admin vê todos)
- Dashboard placeholder
- 3 migrations reversíveis (tenants, users, user_tenants)
- 83 testes, cobertura F02 87.6%
- Segurança: 3 vulnerabilidades encontradas e corrigidas (err.Error() exposto, cookie Secure, SameSite)
- Artefatos: stories, wireframes, design técnico, cenários QA, validação QA, review segurança
