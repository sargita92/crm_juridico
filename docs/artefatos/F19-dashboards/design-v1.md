# Feature: Dashboards (Tenant + Admin) — Design Spec

**Data**: 2026-04-07
**Status**: Aprovado
**Prioridade**: 3 (terceira a implementar)

## Contexto

Ambos os dashboards (tenant e admin) sao stubs com mensagem de boas-vindas. Esta feature os transforma em paineis com graficos e resumos relevantes usando Chart.js.

## Regras de Negocio

### Dashboard Tenant
- **Owner/Admin do tenant**: ve dados consolidados de todo o tenant
- **Usuario comum**: mesmos 5 blocos, filtrados por `responsible_user_id = user_id` (apenas seus dados)
- Dados carregam ao abrir a pagina e ao clicar no botao "Atualizar"
- Periodo fixo: todos os dados disponiveis (sem seletor de periodo)

### Dashboard Admin
- Visivel apenas para usuarios com role "admin" (gestores da plataforma)
- Dados agregados de todos os tenants
- Mesma mecanica: carrega ao abrir + botao atualizar

## Modulo

Novo modulo `internal/dashboard/` seguindo Clean Architecture:

```
internal/dashboard/
├── domain/
│   ├── tenant_stats.go       # Structs de output do tenant
│   └── admin_stats.go        # Structs de output do admin
├── application/
│   ├── get_tenant_dashboard.go  # Use case tenant
│   └── get_admin_dashboard.go   # Use case admin
├── infrastructure/
│   ├── gorm_tenant_stats_repo.go   # Queries agregadas tenant
│   ├── gorm_admin_stats_repo.go    # Queries agregadas admin
│   └── prometheus_stats_provider.go # Metricas de infra
└── interfaces/http/
    ├── handler.go             # Handlers HTMX
    └── routes.go              # Rotas
```

## Dashboard Tenant — 5 Blocos

### Bloco 1: Funil/Leads
- Total de leads por status: open, won, lost (contadores)
- Leads por coluna do funil ativo (bar chart horizontal)
- Taxa de conversao: won / (won + lost) (doughnut chart)
- Leads novos hoje e na semana (contadores)

### Bloco 2: WhatsApp
- Mensagens recebidas vs enviadas (bar chart comparativo)
- Conversas ativas (contador)
- Tempo medio de primeira resposta (contador)

### Bloco 3: Performance por Responsavel
- Leads por responsavel (bar chart)
- Taxa de conversao por responsavel (bar chart agrupado)
- *Usuario comum*: ve apenas seus proprios dados neste bloco

### Bloco 4: Tempo no Funil
- Tempo medio por coluna (bar chart horizontal)
- Leads parados ha mais de 7 dias (lista com contador de alerta)

### Bloco 5: Produtos
- Leads por produto (doughnut chart)
- Conversao por produto (bar chart)

## Dashboard Admin — 6 Blocos

### Bloco 1: Tenants
- Total por status: ativos / inativos / bloqueados (doughnut chart)
- Novos tenants no mes (contador)
- Crescimento mes a mes, ultimos 6 meses (line chart)

### Bloco 2: Uso da Plataforma
- Total de leads no sistema (contador)
- Total de mensagens WhatsApp (contador)
- Conversas ativas agregadas (contador)

### Bloco 3: Health por Tenant
- Top 10 tenants mais ativos por leads criados (bar chart horizontal)
- Tenants inativos ha mais de 30 dias (lista com alerta)

### Bloco 4: Infraestrutura
- Latencia media da API (contador, via Prometheus)
- Taxa de erros 5xx (contador, via Prometheus)
- Status dos servicos (indicadores verde/vermelho)

### Bloco 5: Especialistas/IA
- Total de agentes cadastrados (contador)
- Qualificacoes realizadas (contador)
- Agentes por tenant (bar chart)

### Bloco 6: Financeiro/Billing (revisado 2026-04-19 — dados reais de F11)
- Contadores do ano corrente: receita (pago), total pendente, total atrasado — agregados de todos os tenants
- Distribuição de tenants por plano (doughnut): mensal / anual / vitalício / externo
- Top 10 tenants com maior valor atrasado (bar horizontal)
- Fonte: `internal/pagamentos` (PaymentRepository + BillingConfig)

## UI — Layout Comum

- Grid responsivo: 2 colunas em desktop, 1 coluna em mobile
- Cada bloco e um card com titulo, subtitulo, e conteudo (grafico ou contadores)
- Botao "Atualizar" no topo da pagina: `hx-get` recarrega o fragmento inteiro do dashboard
- Chart.js via CDN incluido no layout
- Graficos inicializados via `<script>` inline apos cada bloco ou via evento `htmx:afterSwap`
- Contadores exibidos em cards menores com icone + valor + label

## Rotas

```
GET /dashboard              → Tenant dashboard (handler existente, substituir stub)
GET /dashboard/content      → Fragmento HTMX do dashboard tenant (para refresh)
GET /admin/dashboard        → Admin dashboard (handler existente, substituir stub)
GET /admin/dashboard/content → Fragmento HTMX do dashboard admin (para refresh)
```

## Dependencias

- Chart.js via CDN (nenhuma dependencia Go)
- Repositorios existentes: lead, funnel, column, conversation, message, tenant, user
- Prometheus client (ja configurado) para metricas de infraestrutura no admin

## Testes

- Unitarios: use cases com dados mockados, validacao de calculos (taxas, medias)
- Integracao: handlers retornam HTML valido com dados corretos
- OWASP: usuario comum so ve seus dados, isolamento de tenant, admin dashboard inacessivel para usuarios normais
