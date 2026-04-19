# F19 - Dashboards (Admin + Tenant)

## Objetivo
Transformar os stubs atuais de `/dashboard` (tenant) e `/admin/dashboard` em painéis com gráficos e resumos relevantes, usando Chart.js via HTMX. Entregar visão consolidada de operação para donos de tenant, visão individual para usuários comuns e visão agregada da plataforma para admins.

## Pré-requisitos
- F06 (integração WhatsApp) — fonte de dados de mensagens/conversas
- F07 (funis/kanban) — fonte de dados de leads, colunas, conversão
- F08 (usuários e permissões) — perfis (owner/admin do tenant vs usuário comum) e isolamento
- F10 (produtos) — agregações por produto

## Status: concluído (2026-04-19)

## Design aprovado
Ver [docs/artefatos/F19-dashboards/design-v1.md](../artefatos/F19-dashboards/design-v1.md) (aprovado em 2026-04-07).

## Regras de negócio
- **Tenant — owner/admin**: dados consolidados de todo o tenant
- **Tenant — usuário comum**: mesmos blocos, filtrados por `responsible_user_id = user_id`
- **Admin**: visível apenas para role `admin` (plataforma); dados agregados de todos os tenants
- Dados carregam ao abrir a página e ao clicar em "Atualizar"
- Período fixo: todos os dados disponíveis (sem seletor de período nesta entrega)

## Steps

### Step 1: Módulo `internal/dashboard` (domínio + use cases)
- [x] structs de output tenant (`TenantStats`) e admin (`AdminStats`)
- [x] use case `GetTenantDashboard` com filtro por `responsible_user_id` quando usuário comum
- [x] use case `GetAdminDashboard` (guard por role admin)
- [x] testes unitários com dados mockados (taxas, médias)

### Step 2: Repositórios e providers
- [x] `gorm_tenant_stats_repo.go` — queries agregadas por tenant (leads, funil, WhatsApp, produtos, responsáveis, tempo no funil)
- [x] `gorm_admin_stats_repo.go` — queries agregadas da plataforma (tenants, uso, top ativos, especialistas)
- [x] `prometheus_stats_provider.go` — métricas de infra (latência, erro 5xx, status de serviços)
- [x] testes de integração com testcontainers-go

### Step 3: Handlers HTMX e rotas
- [x] `GET /dashboard` — substitui stub; renderiza layout + fragmento
- [x] `GET /dashboard/content` — fragmento para refresh (botão Atualizar)
- [x] `GET /admin/dashboard` — substitui stub; guard por role admin
- [x] `GET /admin/dashboard/content` — fragmento admin
- [x] registrar rotas em `cmd/api`
- [x] testes de handler (HTML válido, dados corretos)

### Step 4: UI — Dashboard Tenant (5 blocos)
- [x] Bloco 1 — Funil/Leads: totais por status (open/won/lost), leads por coluna (bar), conversão (doughnut), novos hoje/semana
- [x] Bloco 2 — WhatsApp: mensagens in/out (bar), conversas ativas, tempo médio de primeira resposta
- [x] Bloco 3 — Performance por responsável: leads por responsável (bar), conversão por responsável; usuário comum vê só os próprios
- [x] Bloco 4 — Tempo no funil: tempo médio por coluna (bar horizontal), leads parados > 7 dias (lista + alerta)
- [x] Bloco 5 — Produtos: leads por produto (doughnut), conversão por produto (bar)

### Step 5: UI — Dashboard Admin (6 blocos)
- [x] Bloco 1 — Tenants: ativos/inativos/bloqueados (doughnut), novos no mês, crescimento 6 meses (line)
- [x] Bloco 2 — Uso da plataforma: total de leads, total de mensagens WhatsApp, conversas ativas
- [x] Bloco 3 — Health por tenant: top 10 mais ativos (bar horizontal), inativos > 30 dias (lista + alerta)
- [x] Bloco 4 — Infraestrutura: latência média, taxa de 5xx, status de serviços (Prometheus)
- [x] Bloco 5 — Especialistas/IA: total de agentes, qualificações, agentes por tenant
- [x] Bloco 6 — Financeiro/Billing (dados reais de F11): receita do ano, total pendente, total atrasado (contadores), tenants por plano (doughnut), top 10 tenants atrasados (bar horizontal)

### Step 6: Frontend comum
- [x] Chart.js via CDN no layout (admin e tenant)
- [x] grid responsivo: 2 colunas desktop, 1 coluna mobile
- [x] cards de contadores (ícone + valor + label)
- [x] inicialização de gráficos via evento `htmx:afterSwap`
- [x] botão "Atualizar" com `hx-get` no fragmento de conteúdo

### Step 7: Segurança e observabilidade
- [x] testes OWASP: usuário comum só vê seus dados; isolamento de tenant; admin dashboard inacessível para não-admin
- [x] métricas: `dashboard_render_duration_seconds{scope=tenant|admin}`, `dashboard_load_total{scope,outcome}`
- [x] logs com `request_id`, `tenant_id`, `user_id`
- [x] spans OTel nos use cases

### Step 8: Arquivos `.http`, documentação e changelog
- [x] adicionar em `rest/` fluxos para `/dashboard` e `/admin/dashboard` (inclui `/content`)
- [x] atualizar `docs/engenharia/observabilidade.md` com novas métricas
- [x] entrada em `docs/processo/changelog.md`

## Decisões técnicas
- Módulo `internal/dashboard/` com DDD + Clean Architecture
- Chart.js via CDN (zero dependência Go nova)
- Sem seletor de período nesta entrega — "todos os dados disponíveis"
- Refresh manual via HTMX (sem auto-refresh nesta entrega)
- Reutiliza repositórios existentes (lead, funnel, column, conversation, message, tenant, user, product, specialist)

## Critérios de aceite
- [x] tenant dashboard exibe os 5 blocos com dados reais
- [x] admin dashboard exibe os 6 blocos com dados reais (Financeiro **com dados reais de F11** — opção B aprovada em 2026-04-19)
- [x] usuário comum vê apenas seus próprios dados nos blocos 3
- [x] admin dashboard inacessível para não-admin (403)
- [ ] tempo de render p95 < 500ms em ambiente de dev com massa de dados típica _(não medido — métricas `dashboard_render_duration_seconds` instaladas; medir após deploy de staging)_
- [x] cobertura >= 80% no módulo `dashboard`
- [x] build + containers OK
- [x] arquivos `.http` em `rest/` atualizados
