# Backlog

## Legenda de status
- `backlog` — ainda não iniciado
- `em andamento` — feature em desenvolvimento
- `concluído` — entregue e validado

---

## Épico 1: Fundação

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F01 | [Setup Inicial](../features/F01-setup-inicial.md) | concluído | — | alta |
| F02 | [Autenticação e Multitenancy](../features/F02-autenticacao-multitenancy.md) | concluído | F01 | alta |

## Épico 2: Admin — Tenants e Especialistas

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F03 | [CRUD de Tenants](../features/F03-crud-tenants-admin.md) | concluído | F02 | alta |
| F04 | [CRUD de Especialistas](../features/F04-especialistas-crud.md) | concluído | F03 | alta |
| F05 | [Treinamento de Especialistas](../features/F05-especialistas-treinamento.md) | concluído | F04 | alta |
| F27 | [Variáveis de escritório no prompt do especialista](../features/F27-variaveis-de-escritorio-no-prompt.md) | backlog | F04, F05 | alta |

## Épico 3: WhatsApp e Funil de Vendas

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F06 | [Integração com WhatsApp](../features/F06-integracao-whatsapp.md) | concluído | F02 | alta |
| F07 | [Funis de Vendas (Kanban)](../features/F07-funis-kanban.md) | concluído | F06 | alta |
| F20 | [WhatsApp Business API (Meta) — Provider de Produção](../features/F20-whatsapp-meta-provider.md) | backlog | F06 | alta |
| F22 | [WhatsApp Meta — Onboarding e Billing Avançado](../features/F22-whatsapp-meta-onboarding-billing.md) | backlog | F20, F11 | média |
| F28 | [Sinal de não-lido nas notas da conversa](../features/F28-notas-sinal-de-nao-lido.md) | backlog | F06, F07 | média |

## Épico 4: Gestão de Equipe e Operação

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F08 | [Usuários e Permissões](../features/F08-usuarios-permissoes.md) | concluído | F07 | média |
| F09 | [Automações](../features/F09-automacoes.md) | concluído | F07 | média |
| F10 | [Produtos](../features/F10-produtos.md) | concluído | F07 | alta |

## Épico 5: Gestão de Arquivos

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F14 | [Arquivos por Lead](../features/F14-arquivos.md) | concluído | F06, F07 | média |

## Épico 6: Admin — Financeiro e Operacional

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F11 | [Pagamentos](../features/F11-pagamentos-admin.md) | concluído | F03 | média |
| F12 | [Logs](../features/F12-logs-admin.md) | concluído | F01 | baixa |
| F18 | [Observabilidade Avançada](../features/F18-observabilidade-avancada.md) | concluído | F08, F09 | média |

## Épico 7: IA e MCP

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F15 | [MCP Interno para Especialistas](../features/F15-mcp-interno-especialistas.md) | concluído | F05 | média |
| F16 | [Motor de IA dos Especialistas](../features/F16-motor-ia-especialistas.md) | concluído | F05, F06, F07, F10 | alta |

## Épico 8: Marketing

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F13 | [Landing Page](../features/F13-landing-page.md) | concluído | — | baixa |

## Épico 9: Ferramentas de desenvolvimento

Suporte interno, não faz parte do produto final.

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F17 | [Fluxo de teste manual (/reset)](../features/F17-fluxo-teste-manual.md) | concluído | F16 | baixa |
| F17 | [AI Playground](../features/F17-ai-playground.md) | concluído | F16 | baixa |

## Épico 10: Dashboards e Analytics

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F19 | [Dashboards (Admin + Tenant)](../features/F19-dashboards.md) | concluído | F06, F07, F08, F10 | média |
| F25 | [Filtro de Dashboard por Usuário (Tenant)](../features/F25-dashboard-filtro-por-usuario.md) | backlog | F19, F08 | média |

## Épico 11: Manutenção e Qualidade Técnica

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F21 | [Saneamento Técnico (one-shot)](../features/F21-saneamento-tecnico.md) | concluído | F01 | alta |
| F26 | [Bug: gargalo intermitente de banco (delays até ~19s)](../features/F26-gargalo-banco.md) | em andamento | — | alta |

> Após F21, manutenção contínua segue o processo recorrente em [manutencao-tecnica.md](manutencao-tecnica.md) (não vai ao backlog).
>
> F26 (reportado 2026-05-25): em alguns momentos o banco gargala e o tempo de resposta chega a ~19s. Suspeita inicial: AI Playground (F17), mas a **causa-raiz não está confirmada**. Primeiro passo é instrumentar/profilar (queries lentas, N+1, pool de conexões, locks) para localizar o gargalo antes de propor a correção.

## Épico 12: Qualificação Avançada

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F23 | [Qualificação Multi-Destino (faixa cinzenta + cross-sell)](../features/F23-qualificacao-multi-destino.md) | concluído | F16, F07, F10 | alta |

---

## Ordem sugerida de execução

```text
Iteração 1: F01 (setup)
Iteração 2: F02 (auth + multitenancy)
Iteração 3: F03 (CRUD tenants)
Iteração 4: F04 (CRUD especialistas)
Iteração 5: F05 (treinamento especialistas)
Iteração 6: F06 (integração WhatsApp — steps 1-4)
Iteração 7: F07 (funis/kanban)
Iteração 8: F10 (produtos)
Iteração 9: F16 (motor de IA — especialista responde no WhatsApp)
Iteração 10: F08 (usuários e permissões)
Iteração 11: F09 (automações)
Iteração 12: F14 (arquivos por lead)
Iteração 13: F15 (MCP interno para especialistas)
Iteração 14: F11 (pagamentos)
Iteração 15: F12 (logs)
Iteração 16: F13 (landing page)
Iteração 17: F19 (dashboards admin + tenant)
Iteração 18: F23 (qualificação multi-destino)
```

## Notas
- cada feature é uma iteração/entrega independente
- features dentro do mesmo épico podem ser desenvolvidas em sequência
- F06 pode começar em paralelo com F04/F05 se houver capacidade
- F08, F09 e F10 podem ser paralelizadas após F07
