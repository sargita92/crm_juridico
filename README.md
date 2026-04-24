# CRM Jurídico

CRM multitenant integrado com WhatsApp, voltado ao ambiente jurídico e extensível a outras áreas. Backend em Go com renderização server-side via HTMX.

## Visão geral

- **Admin**: gestão de tenants, especialistas (agentes IA treináveis), produtos, pagamentos e dashboards consolidados.
- **Tenant**: atendimento via WhatsApp, funis em kanban, automações, usuários/permissões, arquivos por lead e dashboard operacional.
- **Multitenancy**: banco único com isolamento lógico por `tenant_id`.
- **WhatsApp**: `whatsmeow` em desenvolvimento; interface de provider permite migrar para Meta Business API em produção.

## Stack

**Backend**: Go 1.26, Gin, Gorm, MySQL, golang-migrate, Viper, Zap, JWT, whatsmeow
**Frontend**: HTMX + `html/template` server-side, CSS próprio, Chart.js em dashboards
**Observabilidade**: Prometheus, OpenTelemetry, Grafana
**Testes**: stretchr/testify, testcontainers-go
**Infra**: Docker, Air (hot reload)

Detalhes em [docs/engenharia/stack.md](docs/engenharia/stack.md).

## Arquitetura

DDD + Clean Architecture, organizado por feature em `internal/<feature>/`:

```text
cmd/api/                     # entry point mínimo
internal/
  <feature>/
    domain/                  # entidades, regras de negócio, interfaces de repo
    application/             # casos de uso
    infrastructure/          # repositórios Gorm, clients externos
    interfaces/http/         # handlers, rotas
  shared/                    # middleware, config, database, módulo
migrations/                  # golang-migrate versionado
web/templates/               # layouts, partials e templates por feature
web/static/                  # css, js
rest/                        # arquivos .http para testes manuais
docs/                        # princípios, stack, arquitetura, features, processos
```

Cada feature expõe um `Module` que cuida do próprio wire-up; `main.go` apenas registra os módulos. Detalhes em [docs/engenharia/arquitetura.md](docs/engenharia/arquitetura.md) e [docs/engenharia/principios.md](docs/engenharia/principios.md).

## Pré-requisitos

- Go 1.26+
- Docker e Docker Compose
- MySQL 8 (via container)

## Setup

```bash
cp .env.dist .env
cp docker-compose.dev.yml.dist docker-compose.dev.yml   # se houver
docker compose -f docker-compose.dev.yml up -d
```

A aplicação sobe em `http://localhost:8533` com hot reload via Air. Migrations executam automaticamente no startup.

### Reset do banco + fixtures

```bash
./scripts/refresh.sh
```

**Credenciais padrão após refresh:**
- URL: `http://localhost:8533/login`
- Email: `admin@teste.com`
- Senha: `admin123`

Mais detalhes em [docs/engenharia/docker-ambientes.md](docs/engenharia/docker-ambientes.md).

## Testes

```bash
go test ./...                              # todos os testes
go test -cover ./...                       # com cobertura
go test -coverprofile=coverage.out ./...   # arquivo de cobertura
```

- Cobertura mínima exigida: **80%**
- Testes de integração usam `testcontainers-go` com MySQL real
- Todo endpoint tem testes OWASP: 401/403, isolamento de tenant, anti-injection

Estratégia completa em [docs/engenharia/testes.md](docs/engenharia/testes.md).

## Observabilidade

- **Logs**: Zap estruturado com `request_id`, `tenant_id`, `user_id`, `trace_id`, `span_id` (via `observability.LoggerFromContext`).
- **Métricas**: Prometheus em `:9190/metrics` — histogramas de duração de negócio (`crm_automation_execution_duration_seconds`, `crm_permission_check_duration_seconds`, `crm_specialist_response_duration_seconds`) + counters de eventos (`invites_total`, `crm_notifications_read_total`, `crm_load_balance_fallback_total`, …).
- **Traces**: OpenTelemetry end-to-end com exporter configurável via `OTEL_EXPORTER_OTLP_ENDPOINT` (Tempo em dev, stdout quando vazio).
- **Tempo** em `:3200` (OTLP gRPC em `:4317`) — retenção 7d.
- **Dashboards**: Grafana em `:3100` — `overview`, `whatsapp`, `leads-kanban`, `especialistas`, `equipe`, `pagamentos`.
- **Alertmanager** em `:9093` — 4 regras em `infra/prometheus/alerts.yml` validadas no CI com `promtool test rules`.
- **Runbooks** operacionais em [docs/operacoes/runbooks/](docs/operacoes/runbooks/README.md).

Detalhes em [docs/engenharia/observabilidade.md](docs/engenharia/observabilidade.md).

## Features entregues

| # | Feature | Épico |
|---|---------|-------|
| F01 | [Setup Inicial](docs/features/F01-setup-inicial.md) | Fundação |
| F02 | [Autenticação e Multitenancy](docs/features/F02-autenticacao-multitenancy.md) | Fundação |
| F03 | [CRUD de Tenants](docs/features/F03-crud-tenants-admin.md) | Admin |
| F04 | [CRUD de Especialistas](docs/features/F04-especialistas-crud.md) | Admin |
| F05 | [Treinamento de Especialistas](docs/features/F05-especialistas-treinamento.md) | Admin |
| F06 | [Integração com WhatsApp](docs/features/F06-integracao-whatsapp.md) | WhatsApp/Funil |
| F07 | [Funis de Vendas (Kanban)](docs/features/F07-funis-kanban.md) | WhatsApp/Funil |
| F08 | [Usuários e Permissões](docs/features/F08-usuarios-permissoes.md) | Tenant |
| F09 | [Automações](docs/features/F09-automacoes.md) | Tenant |
| F10 | [Produtos](docs/features/F10-produtos.md) | Tenant |
| F11 | [Pagamentos](docs/features/F11-pagamentos-admin.md) | Admin |
| F13 | [Landing Page](docs/features/F13-landing-page.md) | Público |
| F14 | [Arquivos por Lead](docs/features/F14-arquivos.md) | Tenant |
| F15 | [MCP Interno](docs/features/F15-mcp-interno-especialistas.md) | IA |
| F16 | [Motor de IA dos Especialistas](docs/features/F16-motor-ia-especialistas.md) | IA |
| F17 | [Fluxo de Teste Manual + Playground](docs/features/F17-fluxo-teste-manual.md) | IA |
| F18 | [Observabilidade Avançada](docs/features/F18-observabilidade-avancada.md) | Plataforma |
| F19 | [Dashboards (Admin + Tenant)](docs/features/F19-dashboards.md) | Analytics |

Backlog completo e status atualizado em [docs/processo/backlog.md](docs/processo/backlog.md).

## Fluxo de entrega

Trabalho por feature seguindo PO → UI/UX → Arquiteto → QA → Dev Backend → Dev Front-end → QA → Segurança. Cada step é implementado com TDD, validado isoladamente e comitado antes do próximo.

- [Princípios de engenharia](docs/engenharia/principios.md)
- [Definition of Done](docs/processo/definition-of-done.md)
- [Fluxo de entrega](docs/processo/fluxo-entrega.md)
- [Agentes](docs/agentes/)

## Regras invioláveis

Ver [CLAUDE.md](CLAUDE.md) para as regras completas. Resumo:

1. Step-by-step — feature nunca entregue de uma vez
2. TDD obrigatório, cobertura ≥ 80%
3. DDD + Clean Architecture
4. Branch por feature + PR
5. DoD completa antes de merge
6. HTMX primeiro — evitar JS customizado
7. Observabilidade em todo endpoint
8. Testes OWASP em todo endpoint
9. Arquivos `.http` em `rest/` atualizados a cada feature

## Estrutura de pastas

```text
cmd/api/                     # main
internal/<feature>/          # módulos por feature (domain/application/infrastructure/interfaces)
internal/shared/             # middleware, config, database, module
migrations/                  # golang-migrate
web/templates/, web/static/  # HTMX + templates + css/js
rest/                        # .http (JetBrains HTTP Client)
scripts/                     # refresh.sh e utilitários
infra/                       # prometheus, grafana
fixture/                     # seeds para dev
docs/                        # engenharia, processo, features, agentes, artefatos
```
