# Status F25 — Filtro de Dashboard por Usuário (Tenant)

**Branch**: `feature/F25-dashboard-filtro-por-usuario`
**Status**: implementação concluída — testes verdes (unit + integração + OWASP); pendente verificação final + PR
**Plano**: [plan-v1.md](plan-v1.md)
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
- Dev Backend: domínio/aplicação/infra + handler (TDD) — concluído
- Dev Front-end: templates (seletor + fragmento) — concluído
- QA + Segurança: OWASP (escopo travado, isolamento de tenant) — concluído

## Progresso por step

| Step | Descrição | Status | Commit |
|------|-----------|--------|--------|
| 1 | Domínio `Operator` + `ViewUserID` no `TenantInput`; novo filtro no `Execute` (TDD) | concluído | `f95a48d` |
| 2 | Port `OperatorLister` + impl Gorm (JOIN user_tenants ⋈ users, só não-owners) (TDD) | concluído | `cef4181` |
| 3 | Handler: lista operadores, parse/validação de `?user`, view model | concluído | `a51fe8e` |
| 4 | Templates: `<select>` no header + indicador "vendo: X" no fragmento | concluído | `0cc704b` |
| 5 | OWASP + `rest/16-dashboard.http` + observabilidade (`viewed_user_id`) | concluído | _(este)_ |

## Decisões de produto

- Drill-down de **um operador por vez**, padrão **Consolidado (todos)**.
- Dropdown lista **só operadores (não-owners)**.
- Seleção **efêmera** (query param `?user=`).
- Seletor visível **apenas para owner** (admin de plataforma = owner).

## Próximo passo

Verificação final (build, testes -short, cobertura ≥80%, lint, integração com
testcontainers, smoke test no app) e abrir PR para `main`.
