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

## Épico 3: WhatsApp e Funil de Vendas

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F06 | [Integração com WhatsApp](../features/F06-integracao-whatsapp.md) | concluído | F02 | alta |
| F07 | [Funis de Vendas (Kanban)](../features/F07-funis-kanban.md) | em andamento | F06 | alta |

## Épico 4: Gestão de Equipe e Operação

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F08 | [Usuários e Permissões](../features/F08-usuarios-permissoes.md) | concluído | F07 | média |
| F09 | [Automações](../features/F09-automacoes.md) | em andamento | F07 | média |
| F10 | [Produtos](../features/F10-produtos.md) | em andamento | F07 | alta |

## Épico 5: Gestão de Arquivos

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F14 | [Arquivos por Lead](../features/F14-arquivos.md) | backlog | F06, F07 | média |

## Épico 6: Admin — Financeiro e Operacional

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F11 | [Pagamentos](../features/F11-pagamentos-admin.md) | backlog | F03 | média |
| F12 | [Logs](../features/F12-logs-admin.md) | backlog | F01 | baixa |

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
```

## Notas
- cada feature é uma iteração/entrega independente
- features dentro do mesmo épico podem ser desenvolvidas em sequência
- F06 pode começar em paralelo com F04/F05 se houver capacidade
- F08, F09 e F10 podem ser paralelizadas após F07
