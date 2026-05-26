# Status F25 — Filtro de Dashboard por Usuário (Tenant)

**Branch**: `feature/F25-dashboard-filtro-por-usuario`
**Status**: em andamento — design aprovado, implementação iniciada
**Doc da feature**: [../../features/F25-dashboard-filtro-por-usuario.md](../../features/F25-dashboard-filtro-por-usuario.md)
**Design**: [design-v1.md](design-v1.md)

## Resumo

Expor ao owner do escritório um seletor de operador no dashboard tenant
(drill-down de um por vez; padrão "Consolidado"). Reutiliza a fiação de filtro por
usuário que já existe no use case `GetTenantDashboard`. Abordagem escolhida:
`ViewUserID *string` explícito no `TenantInput`. Sem migration nova.

## Fluxo de agentes

- PO: doc da feature (relato + critérios) — concluído
- UI/UX + Arquiteto: design consolidado (seletor, HTMX, sincronia do badge) — concluído
- Dev Backend: domínio/aplicação/infra + handler (TDD) — pendente
- Dev Front-end: templates (seletor + fragmento) — pendente
- QA + Segurança: OWASP (escopo travado, isolamento de tenant) — pendente

## Progresso por step

| Step | Descrição | Status |
|------|-----------|--------|
| 1 | Domínio `Operator` + `ViewUserID` no `TenantInput`; novo filtro no `Execute` (TDD) | pendente |
| 2 | Port `OperatorLister` + impl Gorm (JOIN user_tenants ⋈ users, só não-owners) (TDD) | pendente |
| 3 | Handler: lista operadores, parse/validação de `?user`, view model | pendente |
| 4 | Templates: `<select>` no header + indicador "vendo: X" no fragmento | pendente |
| 5 | OWASP + `rest/dashboard.http` + observabilidade (`viewed_user_id`) | pendente |

## Decisões de produto

- Drill-down de **um operador por vez**, padrão **Consolidado (todos)**.
- Dropdown lista **só operadores (não-owners)**.
- Seleção **efêmera** (query param `?user=`).
- Seletor visível **apenas para owner** (admin de plataforma = owner).

## Próximo passo

Gerar o plano de implementação (writing-plans) e executar por steps com TDD,
começando pelo domínio/aplicação.
