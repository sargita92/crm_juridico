# Status F11 — Pagamentos (Admin)

**Branch**: `feature/F11-pagamentos`
**Spec**: [../../superpowers/specs/2026-04-19-F11-pagamentos-admin-design.md](../../superpowers/specs/2026-04-19-F11-pagamentos-admin-design.md)
**Plano**: [../../superpowers/plans/2026-04-19-F11-pagamentos-admin.md](../../superpowers/plans/2026-04-19-F11-pagamentos-admin.md)

## Fluxo de agentes
- PO: ✅ (brainstorming 2026-04-19 — 8 decisões consolidadas na spec)
- UI/UX: ✅ (decisões no spec §8; templates serão escritos nas Tasks 14-15)
- Arquiteto: ✅ (plano de 19 tasks em `docs/superpowers/plans/`)
- Dev Backend: 🟡 **em andamento — Tasks 1–7 concluídas, 8–19 pendentes**
- Dev Front-end: pendente
- QA: pendente
- Segurança: pendente

## Progresso (Tasks concluídas)

| Task | Descrição | Commit |
|---|---|---|
| Preflight | Branch, pasta de artefatos, dep `robfig/cron/v3` | `122b6ac` |
| 1 | Domain `Payment`, enums, erros e transições | `62dc45c` |
| 2 | `BrazilHolidayCalendar` (feriados + próximo dia útil) | `876c4d0` |
| 2-fix | Thread-safety do calendário (sync.RWMutex + teste `-race`) | `a6765ab` |
| 3 | `BillingConfig` do tenant (domínio) | `473c21c` |
| 4 | Migrations `054` (tenants billing) + `055` (payments) | `3401154` |
| 5 | `GormPaymentRepository` + 15 testes de integração | `2018d99` |
| 6 | Extensão do tenant (5 campos de cobrança) + `GormTenantBillingRepository` | `d85bc70` |
| 7 | UI do tenant: form com fieldset de cobrança + parsing no handler + UC estendido | (pending) |

**Cobertura atual**: 136 testes passando em `internal/tenant/...` (incluindo 4 novos handler tests e 4 UC tests para billing). `go build ./...` e `go vet` limpos.

## Tasks pendentes (retomar nesta ordem)

| Task | Descrição | Dependências |
|---|---|---|
| 8 | UCs mutação: RegisterManualPayment / MarkAsPaid / Cancel | 5 |
| 9 | UCs leitura: ListTenantPayments / ListAllPayments / FinancialSummary | 5, 6 |
| 10 | UCs do cron: GenerateRecurringPayments / RefreshOverdueStatuses | 6, 8 |
| 11 | `BillingScheduler` com `robfig/cron/v3` + métricas Prometheus | 10 |
| 12 | Wiring: `pagamentos.Module` + `cmd/api/main.go` + permissão `payments:view` | 8–11 |
| 13 | Admin HTTP: handler, rotas, view models, testes | 12 |
| 14 | Templates admin (listagem global, aba tenant, form, partials) + CSS | 13 |
| 15 | Portal tenant: middleware, handler, template read-only | 13, 14 |
| 16 | Observabilidade: traces OTel, dashboard Grafana, doc | 15 |
| 17 | Testes OWASP (401/403/404, SQLi, XSS) + cobertura ≥ 80% | 15 |
| 18 | `rest/pagamentos.http` + itens de menu admin/tenant | 13, 15 |
| 19 | Fechamento: backlog, changelog, artefato v1 | 18 |

## Decisões-chave (resumo para próxima sessão)

- Escopo: apenas Steps 1, 2 e 4 da spec original. Step 3 (gateway externo) → F11.1.
- Planos do tenant: `mensal` / `anual` / `vitalicio` / `externo`. Valor próprio por tenant (livre). `vitalicio` e `externo` não geram cobrança recorrente.
- Cron diário (`BILLING_CRON=0 3 * * *`, TZ `America/Sao_Paulo`) gera recorrentes faltantes + atualiza status atrasado.
- Carência: `BILLING_GRACE_DAYS=1` (env). Vencimento prorroga para próximo dia útil via `BrazilHolidayCalendar`.
- Auditoria: campos `paid_by_user_id`, `paid_at`, `cancelled_by_user_id`, `cancelled_at` na própria tabela `payments`.
- Valor sempre em **centavos** (`int64`) para evitar dep `shopspring/decimal`.
- Permissões do portal: owner vê por padrão; outros usuários precisam de `Permission(payments, view)`; admin pode ocultar via `exibir_pagamentos` no tenant. Vitalício/externo não veem menu.
- Indicador financeiro: badge (`em_dia` / `pendente` / `atrasado` / `sem_cobranca`) + 3 métricas (total pago no ano, pendente, atrasado).
- UI admin: menu global `/admin/pagamentos` + aba `/admin/tenants/:id/pagamentos`. Portal tenant: `/pagamentos` read-only.

## Como retomar

1. `git checkout feature/F11-pagamentos`
2. Confirmar estado: `git log --oneline main..HEAD` (esperar 8 commits)
3. Abrir [`docs/superpowers/plans/2026-04-19-F11-pagamentos-admin.md`](../../superpowers/plans/2026-04-19-F11-pagamentos-admin.md) e retomar a partir da **Task 7**
4. Seguir o fluxo subagent-driven-development (implementer → spec review → quality review) para cada task restante
5. Ao concluir Task 19, abrir PR e atualizar este status para `concluído`

## Ajustes ao plano durante a execução (para incorporar no futuro)

- **Task 2**: algoritmo de Páscoa no plano tinha `l % 30` (errado). Correto é `l % 7` (Meeus/Jones/Butcher). Já aplicado em `a6765ab`.
- **Task 2**: cache do calendário protegido por `sync.RWMutex` (double-check pattern). Já aplicado em `a6765ab`.
- **Task 5**: `ExistsRecorrente` precisa usar `competencia = DATE(?)` para contornar mismatch DATE/DATETIME quando DSN usa `loc=Local`. Já aplicado em `2018d99`.
- **Task 6**: remover struct tag `default:true` do GORM em `ExibirPagamentos` — Gorm trata `false` como "unset" e deixa o MySQL default sobrescrever. Mantido `not null`; default fica no nível do banco (migration 054). Já aplicado em `d85bc70`.
- **Task 6**: validação de `BillingConfig` continua centralizada em `pagamentos/domain`; o tenant expõe apenas o setter `SetBillingConfig` (sem acoplar `tenant/domain` a `pagamentos/domain`). Cuidado ao implementar Task 7 — o handler/UC de update do tenant é quem vai validar via `pagamentos/domain.NewBillingConfig(...)` antes de chamar `SetBillingConfig`.
- **Task 7**: fieldset de cobrança exibido apenas no modo edição (não no create) para evitar defaults parciais. Erros de validação (parse ou domínio) retornam 200 + form error (convenção HTMX existente), não 422 como sugerido no plano. Helpers `formatValor/uint8Or/dateOr` duplicados em `cmd/api/main.go` e `internal/shared/testhelper/mysql.go`.

## Observação

Se precisar repor permissões de leitura, lembre-se que `storage/files/` pode ter arquivos criados por `root` via Docker (não afeta o build; só o `go build ./...` recursivo).
