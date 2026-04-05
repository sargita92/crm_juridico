# F11 - Pagamentos (Admin)

## Objetivo
Implementar o controle de pagamentos por tenant no painel administrativo.

## Pré-requisitos
- F03 (CRUD tenants)

## Steps

### Step 1: Domínio de pagamentos
- [ ] criar entidade Payment (id, tenant_id, valor, tipo, status, referencia, observacao, data_pagamento, created_at)
- [ ] tipos: automatico, manual
- [ ] status: pendente, pago, atrasado, cancelado
- [ ] migration
- [ ] testes unitários

### Step 2: Casos de uso
- [ ] registrar pagamento manual
- [ ] listar pagamentos de um tenant
- [ ] alterar status de pagamento
- [ ] consultar situação financeira do tenant
- [ ] testes

### Step 3: Integração com gateway (opcional/futuro)
- [ ] definir interface de gateway de pagamento
- [ ] implementar provider (ex: Stripe, Mercado Pago)
- [ ] webhook para receber confirmação de pagamento
- [ ] atualizar status automaticamente
- [ ] testes

### Step 4: Telas admin (HTMX)
- [ ] template de listagem de pagamentos por tenant
- [ ] formulário de registro de pagamento manual
- [ ] indicador de situação financeira no detalhe do tenant
- [ ] filtros por status e período

## Critérios de aceite
- admin consegue registrar e consultar pagamentos
- pagamento manual funciona
- histórico de pagamentos visível
- cobertura >= 80%
