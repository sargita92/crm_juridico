# Status F19 — Dashboards (Tenant + Admin)

**Branch**: `feature/F19-dashboards`
**Status**: 🟡 em andamento (Task 4/20 concluída)
**Spec**: [../../superpowers/specs/2026-04-07-dashboards-design.md](../../superpowers/specs/2026-04-07-dashboards-design.md) (v2 — revisada em 2026-04-19)
**Plano**: [../../superpowers/plans/2026-04-19-F19-dashboards.md](../../superpowers/plans/2026-04-19-F19-dashboards.md)
**Artefato aprovado**: [design-v1.md](design-v1.md)

## Fluxo de agentes
- PO: ✅ (spec v1 em 2026-04-07, revisada v2 em 2026-04-19 com Bloco 6 real)
- UI/UX: 🟡 pendente — templates nos Steps 14-16 do plano
- Arquiteto: ✅ (plano de 20 tasks em `docs/superpowers/plans/2026-04-19-F19-dashboards.md`)
- Dev Backend: 🟡 pendente (Tasks 1–13, 17)
- Dev Front-end: 🟡 pendente (Tasks 14–16)
- QA: 🟡 pendente (Task 18)
- Segurança: 🟡 pendente (Task 18)

## Progresso (Tasks concluídas)

| Task | Descrição | Status | Commit |
|---|---|---|---|
| Preflight | Branch, spec v2, status, plano | ✅ | 573634f |
| 1 | Domain outputs (TenantStats, AdminStats) | ✅ | 746ebb5 |
| 2 | GetTenantDashboard UC + fakes + testes | ✅ | b03a730 + bb57a7e |
| 3 | GetAdminDashboard UC + testes | ✅ | e311e15 |
| 4 | `PaymentRepository.GlobalSummary` (integra F19 ↔ F11) | ✅ | f4579bb |
| 5 | Tenant stats repo (blocos 1, 5 — funil/leads/produtos) | ⬜ | — |
| 6 | Tenant stats repo (bloco 2 — WhatsApp) | ⬜ | — |
| 7 | Tenant stats repo (blocos 3, 4 — responsáveis/tempo) | ⬜ | — |
| 8 | Admin stats repo (blocos 1, 2, 3 — tenants/uso/health) | ⬜ | — |
| 9 | Admin stats repo (blocos 5, 6 — especialistas/financeiro) | ⬜ | — |
| 10 | PrometheusStatsProvider (bloco 4 admin) | ⬜ | — |
| 11 | Wiring `dashboard.Module` + `cmd/api/main.go` | ⬜ | — |
| 12 | Handlers HTTP tenant | ⬜ | — |
| 13 | Handlers HTTP admin | ⬜ | — |
| 14 | Chart.js + `dashboard.css` + layouts | ⬜ | — |
| 15 | Templates tenant (5 blocos) | ⬜ | — |
| 16 | Templates admin (6 blocos) | ⬜ | — |
| 17 | Observabilidade (metrics/spans/logs) | ⬜ | — |
| 18 | Testes OWASP + cobertura ≥ 80% | ⬜ | — |
| 19 | `rest/11-dashboard.http` + menus + changelog | ⬜ | — |
| 20 | Fechamento (backlog, status, PR) | ⬜ | — |

## Decisões-chave
- **Escopo**: todos os 8 Steps do feature file F19 estão no plano. Nada foi postergado.
- **Bloco 6 admin**: consome dados reais de F11 via `PaymentRepository.GlobalSummary` (opção B aprovada em 2026-04-19). Inclui: receita do ano, total pendente, total atrasado, distribuição por plano (doughnut), top 10 tenants atrasados (bar horizontal).
- **Usuário comum do tenant**: filtra **todos** os blocos tenant por `responsible_user_id` (não apenas o Bloco 3). Owner/admin do tenant vê dados consolidados.
- **Admin de plataforma** acessando `/dashboard` (contexto de tenant, caso raro) é tratado como owner para efeito do dashboard.
- **Bloco 4 admin (Infra)**: lê `prometheus.DefaultGatherer` (latência e taxa 5xx) + health checks injetáveis (MySQL via `db.Ping`; WhatsApp opcional via callback).
- **Valores monetários**: sempre em centavos no domínio; formatação pt-BR (`R$ 1.234,56`) no view model.
- **Chart.js via CDN** (zero dependência Go), com SRI. Nenhum bundler/build JS.
- **Refresh manual**: botão "Atualizar" usa `hx-get` no fragmento `/dashboard/content` (ou `/admin/dashboard/content`). Sem auto-refresh.
- **Nova dependência cruzada**: `pagamentos.Module.PaymentRepo()` getter (exposto em Task 4) é consumido por `dashboard.NewModule` no wiring (Task 11).

## Como retomar (instruções completas)

1. **Checkout da branch**:
   ```bash
   git checkout feature/F19-dashboards
   ```

2. **Confirmar estado**:
   ```bash
   git log --oneline main..HEAD    # esperar commits incrementais por task
   git status                       # working tree limpa
   ```

3. **Abrir o plano e identificar próxima task não marcada**:
   - Arquivo: [`docs/superpowers/plans/2026-04-19-F19-dashboards.md`](../../superpowers/plans/2026-04-19-F19-dashboards.md)
   - Procurar a primeira caixinha `- [ ]` não marcada; é a próxima ação.
   - Atualizar este `status.md` ao concluir cada task (marcar status + colar hash do commit na coluna `Commit`).

4. **Fluxo recomendado por task** (usar `superpowers:subagent-driven-development`):
   - Spawn subagent `implementer` → executa a task seguindo os Steps do plano (TDD).
   - Spawn subagent `spec-reviewer` → confere se a task respeitou o plano.
   - Spawn subagent `quality-reviewer` → confere estilo/cobertura/segurança.
   - Ao passar as 2 reviews, commitar.

5. **Testes ao longo do caminho**:
   - Unitários de aplicação: `go test ./internal/dashboard/application/... -v`
   - Integração de repositórios: `go test ./internal/dashboard/infrastructure/... -v`
   - Handlers: `go test ./internal/dashboard/interfaces/http/... -v`
   - Full suite ao final: `go test ./... -count=1`

6. **Na última task (20 — fechamento)**:
   - Atualizar `docs/processo/backlog.md` → F19 como `concluído`.
   - Atualizar `docs/features/F19-dashboards.md` → marcar todos os checkboxes e status `concluído`.
   - Rodar `go test ./... -count=1 && go vet ./... && go build ./...` e confirmar tudo verde.
   - Abrir PR: `gh pr create --title "F19: dashboards (tenant + admin)" --body "$(cat docs/artefatos/F19-dashboards/status.md)"`.

## Ajustes ao plano durante a execução

_(Preencher conforme forem aparecendo divergências entre o plano e a realidade do código — tabelas/colunas com nomes diferentes, helpers que já existem, etc. Este é o log de "lições aprendidas" para referência futura.)_

- **Task 1 — naming dos sentinel errors**: o plano usa `ErrTenantRequired` / `ErrUserRequired`, mas o restante do projeto (funnel, pagamentos) usa `ErrTenantIDRequired` / `ErrUserIDRequired`. Mantido como no plano para não desviar; se incomodar em Task 2 (UC retorna esses erros), avaliar rename antes do wiring.
- **Task 1 — observação para Task 11/12 (infra)**: `Infrastructure.APILatencyMs` zero-value não distingue "Prometheus indisponível" de "sem requests". Considerar flag `PrometheusUp` ou tratar no template.
- **Task 1 — observação para Task 2 (UC tenant)**: lembrar de popular `ScopeIsUser`, `CurrentUserName`, `ActiveFunnelName` em `TenantStats` e cobrir com teste explícito.
- **Task 2 — follow-up (commit `bb57a7e`)**: code review pediu cobertura adicional (fan-out do filtro em todos os 5 blocos, `ErrUserRequired`, propagação de erro por bloco). Production code intocado. Mesmo padrão deve ser aplicado em Task 3 (admin UC) — fake captura por método + tabela de erros.
- **Task 2 — observação para Task 17 (obs)**: `users.UserName` falha silenciosa por design (header degrada graceful). Adicionar `slog.WarnContext` quando Task 17 chegar para não cegar debug em prod.
- **Task 2 — observação para Task 17 (obs)**: providers retornam erro sem `fmt.Errorf("...%w", err)` (segue convenção pagamentos). Quando Task 17 entrar, adicionar log estruturado com `block=funil|whatsapp|...` para identificar bloco que falhou em prod sem quebrar a convenção sentinel-unwrapped.
- **Task 3 — cobertura proativa**: padrão fan-out + propagação de erro replicado (table-driven com 6 sub-tests). Coverage da application 98.3% (1.7% restante = `SystemClock.Now`, intencionalmente sem teste).
- **Task 3 — observação para Task 8/9 (admin SQL providers)**: cada bloco admin é potencial agregação cross-tenant. Cuidado com N+1 — fold "active/inactive/blocked" em um único `GROUP BY status`.
- **Task 3 — observação para Task 10**: receiver value de `fakeInfraProvider` é só do teste; `PrometheusStatsProvider` real provavelmente precisa pointer (cliente Prometheus + lazy init).
- **Task 3 — observação para Task 12/13 (handlers)**: admin dashboard tende a ser endpoint mais lento; setar timeout generoso (5-10s). Se p95 doer, refatorar `Execute` para `errgroup.WithContext` (6 chamadas em paralelo).
- **Task 4 — desvios reais do plano**: helpers `newRepoEnv`/`insertTenant`/`mustCreate`/`ptrTime` não existiam — usado `setupPaymentRepo` + novo `seedTenantWithName`. `Status`/`Tipo` são tipados (`PaymentStatus`/`PaymentType`), exige constantes (`domain.StatusPago` etc.) ao invés de literais string. IDs são UUIDs (não `"t1"`/`"t2"`).
- **Task 4 — efeito colateral**: extender interface `PaymentRepository` forçou stub (3 linhas) em 2 mocks existentes (`application/mocks_test.go`, `infrastructure/billing_scheduler_test.go`). Inevitável em Go.
- **Task 4 — testes de borda recomendados antes de Task 9** (consumer real): (1) DB vazio → `&GlobalSummary{}` sem panic; (2) `data_pagamento = 31/dez/ano-1` NÃO conta no `TotalPagoAnoCents` (timezone-prone); (3) tenant com mix de status → `TopOverdue` só soma `atrasado`. Adicionar quando Task 9 entrar.
- **Task 4 — observação para Task 9**: `GlobalSummary` não cobre distribuição por plano (`Mensal/Anual/Vitalicio/Externo`). Task 9 precisa de query separada em `tenant_billings.plan` (ou estender `GlobalSummary` se ficar pesado).
- **Task 4 — sem transação entre 2 queries**: dashboard tolera eventual consistency. Considerar comment one-liner em `GlobalSummary` se incomodar.
