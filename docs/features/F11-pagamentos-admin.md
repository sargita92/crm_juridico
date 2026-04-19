# F11 - Pagamentos (Admin)

## Objetivo
Implementar o controle de pagamentos por tenant no painel administrativo.

## Pré-requisitos
- F03 (CRUD tenants)

## Steps

### Step 1: Domínio de pagamentos
- [x] criar entidade Payment (id, tenant_id, valor_cents, tipo, status, descricao, competencia, data_vencimento, data_pagamento, paid_by_user_id, cancelled_*, observacao, created_at)
- [x] tipos: recorrente, avulso
- [x] status: pendente, pago, atrasado, cancelado
- [x] migration
- [x] testes unitários

### Step 2: Casos de uso
- [x] registrar pagamento manual (avulso)
- [x] listar pagamentos de um tenant e listar todos (admin)
- [x] marcar como pago / cancelar com motivo
- [x] consultar situação financeira do tenant (badge + 3 métricas)
- [x] geração recorrente via cron diário + atualização de status atrasado
- [x] testes (`internal/pagamentos/application`: 92.4%)

### Step 3: Integração com gateway (opcional/futuro) — DEFERIDO para F11.1
Razão: nesta entrega o cliente quis manter pagamentos manuais. Quando houver decisão de gateway (Stripe vs Mercado Pago), criar feature F11.1 com webhook + provider concreto.

- [ ] definir interface de gateway de pagamento
- [ ] implementar provider (ex: Stripe, Mercado Pago)
- [ ] webhook para receber confirmação de pagamento
- [ ] atualizar status automaticamente
- [ ] testes

### Step 4: Telas admin (HTMX) + portal tenant
- [x] template de listagem global e por tenant
- [x] formulário modal de registro de pagamento manual
- [x] indicador de situação financeira (badge + pago/pendente/atrasado)
- [x] filtros por tenant, status e período
- [x] portal tenant read-only (`/pagamentos`) com middleware de permissão
- [x] sidebars admin e tenant com item "Pagamentos"
- [x] CSS em `web/static/css/main.css`

## Critérios de aceite
- [x] admin consegue registrar e consultar pagamentos
- [x] pagamento manual funciona (+ recorrente via cron)
- [x] histórico de pagamentos visível (admin + portal)
- [x] cobertura >= 80% em todos os pacotes (`internal/pagamentos/*`)

## Entrega

**Release**: 2026-04-19 na branch `feature/F11-pagamentos` (19 commits). Artefato versionado em [`docs/artefatos/F11-pagamentos/v1.md`](../artefatos/F11-pagamentos/v1.md). Entrada no [changelog](../processo/changelog.md).
