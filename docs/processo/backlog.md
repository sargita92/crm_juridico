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
| F04 | [CRUD de Especialistas](../features/F04-especialistas-crud.md) | backlog | F03 | alta |
| F05 | [Treinamento de Especialistas](../features/F05-especialistas-treinamento.md) | backlog | F04 | alta |

## Épico 3: WhatsApp e Funil de Vendas

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F06 | [Integração com WhatsApp](../features/F06-integracao-whatsapp.md) | backlog | F02 | alta |
| F07 | [Funis de Vendas (Kanban)](../features/F07-funis-kanban.md) | backlog | F06 | alta |

## Épico 4: Gestão de Equipe e Operação

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F08 | [Usuários e Permissões](../features/F08-usuarios-permissoes.md) | backlog | F07 | média |
| F09 | [Automações](../features/F09-automacoes.md) | backlog | F07 | média |
| F10 | [Produtos](../features/F10-produtos.md) | backlog | F07 | média |

## Épico 5: Gestão de Arquivos

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F14 | [Arquivos por Lead](../features/F14-arquivos.md) | backlog | F06, F07 | média |

## Épico 6: Admin — Financeiro e Operacional

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F11 | [Pagamentos](../features/F11-pagamentos-admin.md) | backlog | F03 | média |
| F12 | [Logs](../features/F12-logs-admin.md) | backlog | F01 | baixa |

## Épico 7: Marketing

| # | Feature | Status | Dependência | Prioridade |
|---|---------|--------|-------------|------------|
| F13 | [Landing Page](../features/F13-landing-page.md) | backlog | — | baixa |

---

## Ordem sugerida de execução

```text
Iteração 1: F01 (setup)
Iteração 2: F02 (auth + multitenancy)
Iteração 3: F03 (CRUD tenants)
Iteração 4: F04 (CRUD especialistas)
Iteração 5: F05 (treinamento especialistas)
Iteração 6: F06 (integração WhatsApp)
Iteração 7: F07 (funis/kanban)
Iteração 8: F08 (usuários e permissões)
Iteração 9: F09 (automações)
Iteração 10: F10 (produtos)
Iteração 11: F14 (arquivos por lead)
Iteração 12: F11 (pagamentos)
Iteração 13: F12 (logs)
Iteração 14: F13 (landing page)
```

## Notas
- cada feature é uma iteração/entrega independente
- features dentro do mesmo épico podem ser desenvolvidas em sequência
- F06 pode começar em paralelo com F04/F05 se houver capacidade
- F08, F09 e F10 podem ser paralelizadas após F07
