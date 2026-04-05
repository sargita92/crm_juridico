# CRM Jurídico — Instruções do Projeto

## Contexto

CRM integrado com WhatsApp para o ambiente jurídico (extensível a outras áreas). Multitenant, banco único, backend em Go, frontend em HTMX.

## Referências obrigatórias

Antes de implementar qualquer feature, consultar:

- `docs/engenharia/principios.md` — princípios de engenharia
- `docs/engenharia/stack.md` — stack técnica
- `docs/engenharia/arquitetura.md` — diretrizes de arquitetura e estrutura de pastas
- `docs/engenharia/testes.md` — estratégia de testes
- `docs/engenharia/banco-migrations.md` — banco e migrations
- `docs/engenharia/docker-ambientes.md` — Docker e ambientes
- `docs/engenharia/observabilidade.md` — logs, métricas, traces e dashboards
- `docs/processo/fluxo-entrega.md` — fluxo de entrega por agentes
- `docs/processo/definition-of-done.md` — DoD, checklist e fluxo Git
- `docs/processo/backlog.md` — backlog com features e ordem de execução
- `docs/agentes/` — prompts dos agentes (PO, UI/UX, Arquiteto, QA, Dev Backend, Dev Front-end, Segurança)

## Regras invioláveis

1. **TDD**: escrever teste antes de implementar
2. **Cobertura >= 80%**: se cair abaixo, corrigir antes de prosseguir
3. **DDD + Clean Architecture**: domínio isolado, handlers finos, sem vazamento de infraestrutura
4. **Branch por feature**: sem branch e PR aberta, entrega não está concluída
5. **DoD completa**: testes passando + cobertura + build ok + containers ok
6. **UI/UX**: toda interface deve ser simples, intuitiva e bonita
7. **HTMX primeiro**: evitar JavaScript customizado quando HTMX resolver
8. **Agentes**: seguir o fluxo PO → UI/UX → Arquiteto → QA → Dev Backend → Dev Front-end → QA → Segurança
9. **Feature em andamento**: manter `docs/processo/feature-em-andamento.md` atualizado durante o desenvolvimento e limpar ao concluir
10. **Observabilidade**: todo endpoint com métricas, logs com contexto (request_id, tenant_id, user_id), traces end-to-end
11. **WhatsApp via whatsmeow**: usar whatsmeow em dev/testes; interface de provider abstrai para trocar por Meta Business API no futuro
12. **Testes OWASP**: todo endpoint deve ter testes de acesso não autorizado (401/403), isolamento de tenant e anti-injection
13. **Arquivos .http**: manter `rest/` atualizado com novos endpoints a cada feature entregue

## Stack resumida

Go, Gin, Gorm, MySQL, golang-migrate, testcontainers-go, Zap, Prometheus, OpenTelemetry, godotenv, Viper, whatsmeow, HTMX, Go html/template, Docker, Grafana, Air

## Estrutura de pastas

```text
cmd/
  api/
internal/
  <feature>/
    domain/
    application/
    infrastructure/
    interfaces/http/
  shared/
pkg/
web/
  templates/
    layouts/
    partials/
    <feature>/
  static/
    css/
    js/
rest/                            # arquivos .http (JetBrains HTTP Client) para testes manuais
docs/
  engenharia/
  processo/
  agentes/
  produto/
  features/
```
