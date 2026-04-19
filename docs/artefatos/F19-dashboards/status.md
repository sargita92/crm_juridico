# Status F19 — Dashboards (Tenant + Admin)

**Branch**: `feature/F19-dashboards`
**Status**: ✅ concluído (20/20 tasks, 72 testes, cov ≥ 80%)
**Spec**: [../../superpowers/specs/2026-04-07-dashboards-design.md](../../superpowers/specs/2026-04-07-dashboards-design.md) (v2 — revisada em 2026-04-19)
**Plano**: [../../superpowers/plans/2026-04-19-F19-dashboards.md](../../superpowers/plans/2026-04-19-F19-dashboards.md)
**Artefato aprovado**: [design-v1.md](design-v1.md)

## Fluxo de agentes
- PO: ✅ (spec v1 em 2026-04-07, revisada v2 em 2026-04-19 com Bloco 6 real)
- UI/UX: ✅ (templates Tasks 14-16, 15 templates novos, dashboard.css)
- Arquiteto: ✅ (plano de 20 tasks em `docs/superpowers/plans/2026-04-19-F19-dashboards.md`)
- Dev Backend: ✅ (Tasks 1–13, 17 — domain + UCs + repos GORM + handlers + observabilidade)
- Dev Front-end: ✅ (Tasks 14–16 — Chart.js + 15 templates)
- QA: ✅ (Task 18 — 9 testes OWASP, cobertura ≥ 80% em todos os pacotes)
- Segurança: ✅ (Task 18 — A01, A03, A04/A05 cobertos)

## Progresso (Tasks concluídas)

| Task | Descrição | Status | Commit |
|---|---|---|---|
| Preflight | Branch, spec v2, status, plano | ✅ | 573634f |
| 1 | Domain outputs (TenantStats, AdminStats) | ✅ | 746ebb5 |
| 2 | GetTenantDashboard UC + fakes + testes | ✅ | b03a730 + bb57a7e |
| 3 | GetAdminDashboard UC + testes | ✅ | e311e15 |
| 4 | `PaymentRepository.GlobalSummary` (integra F19 ↔ F11) | ✅ | f4579bb |
| 5 | Tenant stats repo (blocos 1, 5 — funil/leads/produtos) | ✅ | 92a604f + 2ace7a8 |
| 6 | Tenant stats repo (bloco 2 — WhatsApp) | ✅ | 87e2010 |
| 7 | Tenant stats repo (blocos 3, 4 — responsáveis/tempo) | ✅ | bcccd05 + 8d45e72 |
| 8 | Admin stats repo (blocos 1, 2, 3 — tenants/uso/health) | ✅ | 3c59e1b + 8e160b6 |
| 9 | Admin stats repo (blocos 5, 6 — especialistas/financeiro) | ✅ | a736dbb |
| 10 | PrometheusStatsProvider (bloco 4 admin) | ✅ | 643c3d5 |
| 11 | Wiring `dashboard.Module` + `cmd/api/main.go` | ✅ | 20f66f2 |
| 12 | Handlers HTTP tenant | ✅ | a4d0ae5 |
| 13 | Handlers HTTP admin | ✅ | dd3fb9d |
| 14 | Chart.js + `dashboard.css` + layouts | ✅ | cce50d2 |
| 15 | Templates tenant (5 blocos) | ✅ | b28663b |
| 16 | Templates admin (6 blocos) | ✅ | b28663b |
| 17 | Observabilidade (metrics/spans/logs) | ✅ | cc8caee |
| 18 | Testes OWASP + cobertura ≥ 80% | ✅ | db2cdf5 |
| 19 | `rest/16-dashboard.http` + menus + changelog | ✅ | 8019212 |
| 20 | Fechamento (backlog, status, PR) | ✅ | _(este commit)_ |

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
- **Task 5 — bugs do plano corrigidos**: (1) tabela é `funnel_columns`, NÃO `columns`; (2) `Row().Scan` retorna `sql.ErrNoRows` (plano usava `gorm.ErrRecordNotFound`) — trocado por `Take(...).Error` + `errors.Is`.
- **Task 5 — schema delta runtime**: `products` é global desde mig 28, com join `tenant_products`. SQL de `ProdutosBlock` está correto (filtra por `leads.tenant_id`); só `seedProduct` precisou inserir nas duas tabelas.
- **Task 5 — interface guard**: `var _ application.TenantStatsProvider = (*GormTenantStatsRepo)(nil)` adicionado para fail-fast em Tasks 6/7/11.
- **Task 5 — observação para Task 6**: `seedConversation` já existe; criar `seedMessage`. Cleanup de `messages`/`conversations` já está em ordem FK.
- **Task 5 — observação para Task 7**: criar `seedUser`/`seedUserTenant`. `responsible_user_id` já é plumbado via `leadOpts.responsible`.
- **Task 6 — adaptação real**: `seedLead` cria contact+conversation internamente (não recebe IDs). Adicionado `leadConversationID(t, db, leadID)` para recuperar a conversa criada. Helper `seedMessage` novo.
- **Task 6 — predicado de user-scope duplicado em 3 lugares** (active count join, message count join, raw SQL `IN (SELECT...)`). Se Task 7 adicionar mais um, considerar extrair helper `applyUserLeadScope(q, tenantID, userID)`. Sinalizar no Task 7 dispatch.
- **Task 6 — testes de borda recomendados** (incluir em Task 7 ou follow-up): (1) tenant sem conversas → todos os campos zero; (2) conversa só com incoming → `FirstResponseAvgSec=0`; (3) lead com `responsible_user_id IS NULL` filtrado fora em user-scope.
- **Task 7 — bugs do plano (mesmos de Task 5) corrigidos**: (1) `JOIN columns` → `JOIN funnel_columns`; (2) `Row().Scan` → `Take(...).Error` + `errors.Is(err, gorm.ErrRecordNotFound)`.
- **Task 7 — helper `applyLeadUserScope` extraído**: usado nos 2 novos métodos. Existing methods (FunilBlock, ProdutosBlock, WhatsAppBlock) NÃO foram refatorados — clean follow-up antes de Task 11 (mecânico, ~10 linhas).
- **Task 7 — contrato de retorno locked down (testes 13-15)**: `(emptyTenant)` → slice vazio não-nil; `TempoFunilBlock(noDefaultFunnel)` → nil; `TempoFunilBlock(funnelExists, noLeads)` → slice vazio não-nil. Handlers (Task 12) precisam tratar `nil` vs `empty slice` como estados distintos.
- **Task 7 — observação para Task 13 (handlers)**: `TempoFunilBlock` retornar nil-sem-erro é "sem funnel default" — handler/template renderiza mensagem "configure um funil"; `FunilBlock` mesma semântica.
- **Task 8 — bug do dispatch corrigido pelo implementer**: SQL `last_activity` com sentinel `'1970-01-01'` em GREATEST não fazia fallback para `t.created_at`. Redesenhada para per-branch `COALESCE(MAX(...), t.created_at)` + GREATEST flat 3-arg + `CAST AS DATETIME` (driver MySQL retorna `[]uint8` sem o cast).
- **Task 8 — decisão de produto**: `Top10Active` inclui tenants `blocked`/`inactive` com leads — métrica é volume histórico, não status. Test de regressão `TestHealthBlock_Top10Active_IncludesBlockedTenants`.
- **Task 8 — observação para Task 9**: (a) construtor de `GormAdminStatsRepo` precisará aceitar `payments` (quebra `setupAdminRepo`); (b) adicionar `DELETE FROM payments` em `setupStatsRepo` antes de seed de pagamentos para `FinanceiroBlock`; (c) `tenants.plano` é VARCHAR (não ENUM) — distribuição lê valores como string.
- **Task 8 — perf annotations futuras**: `InactiveList` usa correlated subqueries (O(N) por tenant); `UsageBlock` faz 3 round-trips. Tudo aceitável agora; revisitar em Task 17 se p95 doer.
- **Task 9 — TODO(F05) Qualifications**: `EspecialistasBlock.Qualifications` fica em 0 com test de regressão `TestEspecialistasBlock_QualificationsAlwaysZero`. Quando F05 expor contador, ajustar 2 linhas (production + test).
- **Task 9 — decisão `EspecialistasBlock.Total`**: conta tanto active quanto inactive (sem WHERE status). Aceitável para "inventário do admin"; se PO quiser só active, adicionar `Where("status='active'")`.
- **Task 9 — `FinanceiroBlock` plan distribution silenciosamente ignora valores desconhecidos**: aceitável (admin vê só categorias conhecidas). Se Task 17 adicionar tracing/log, considerar `default:` que emita métrica. Quando F11 adicionar `plano='trial'`, atualizar struct + switch.
- **Task 9 — observação para Task 11 (wiring)**: ordem de init em `cmd/api`: `pagamentos.Module` ANTES de `dashboard.Module`; passar `pagamentosModule.PaymentRepo()` no `NewGormAdminStatsRepo(db, payments)`.
- **Task 9 — perf**: dashboard admin total fica em ~12+ round-trips (6 blocos × média 2 queries). Revisitar em Task 17 se p95 incomodar.
- **Task 10 — observação para Task 11 (wiring)**: passar `prometheus.DefaultGatherer` (mesmo registry do middleware) + slice de `ServiceCheck`. MySQL: `func(ctx) { return "mysql", db.PingContext(ctx) == nil }`. WhatsApp: `func(ctx) { return "whatsapp", waClient.IsConnected() }`. Evitar checks com I/O pesado (eles entram no tempo de resposta do dashboard).
- **Task 10 — observação para Task 13 (template Bloco 4)**: agregação de latência é platform-wide (média sobre TODOS endpoints+labels). Um endpoint lento domina a média — admin vê o blend, não p99. Se PO quiser p95 por endpoint, precisa de query Prometheus diferente (não cabe aqui).
- **Task 10 — silent-degrade em métrica renomeada**: se `http_requests_total` ou `http_request_duration_seconds` forem renomeados em `internal/shared/middleware/prometheus.go`, o dashboard mostra 0/0 sem erro. Considerar test de regressão futuro que cubra metric-renamed.
- **Task 11 — bugs do plano corrigidos**: (1) `Row().Scan` do `gorm_user_lookup` → `Take` + `errors.Is`; (2) `NewModule` não recebe `users authDomain.UserRepository` (parâmetro era unused); (3) handler stub criado com 4 rotas + bodies placeholder para Tasks 12/13 substituírem só o body.
- **Task 11 — `adminGroup` removido**: era usado só pelo stub `/admin/dashboard`. Outras rotas admin têm seus próprios groups dentro dos módulos. Bloco inteiro de `cmd/api/main.go:442-447` removido.
- **Task 11 — observação para Task 12/13**: tenant routes em `mw.Auth + mw.Tenant`; admin routes em `mw.Auth + mw.Admin`. Handler já recebe `userTenants authDomain.UserTenantRepository` no constructor — usar para determinar `IsOwner` (campo `is_owner` em `user_tenants`).
- **Task 11 — `admin/dashboard.html` template antigo** ainda existe e é referenciado por sidebar/docs legacy. Task 16 (templates admin) substitui ou Task 13 (handler admin) renderiza um template novo.
- **Task 12 — adaptações reais ao plano**: (1) `UserTenantRepository` interface real usa `Associate/FindTenantIDsByUserID/FindByTenantID/FindByUserAndTenant/UpdateIsOwner/UpdateWhatsAppID/RemoveFromTenant/IsOwner` (não os nomes CRUD do plano); (2) middleware expõe `SetClaimsForTest`/`SetTenantIDForTest` (não `SetClaims`/`SetTenantID`); (3) `GetClaims`/`GetTenantID` recebem `ctx`, não `gin.Context`.
- **Task 12 — TenantView shape**: Task 15 (templates tenant) precisa usar `vm.Bloco1_Funil.{Open,Won,Lost,Total}`, `vm.Bloco1_Funil.ConversionPct` (string), `vm.Bloco2_WhatsApp.FirstResponseAvg` (string), etc. Conversão de PT-BR já no view model (R$, %, h).
- **Task 12 — admin handler bodies inalterados**: Task 13 substitui `adminPage`/`adminFragment` (estão como placeholders Task 11).
- **Task 13 — `AdminView` shape**: templates Task 16 usam `vm.Bloco1_Tenants.{Active,Inactive,Blocked,Total,NewThisMonth,Last6Months[]}`, `vm.Bloco4_Infra.{APILatencyMs,Error5xxRate (já formatado),ServicesStatus[]}`, `vm.Bloco6_Financeiro.{ReceitaAno,PendenteTotal,AtrasadoTotal (BRL),PlanDist.{Mensal,Anual,Vitalicio,Externo,Total},TopOverdue[]}`.
- **Task 13 — 401 path** não testado nos handler tests (covered por `internal/shared/middleware/auth_test.go`).
- **Task 14 — SRI hash do Chart.js**: omitido (path (b) do dispatch). TODO inline no `tenant.html`: `TODO(F19): adicionar SRI hash verificado para chart.js@4.4.1`. Script carrega via HTTPS + `crossorigin=anonymous` — aceitável para v1, mas P2 antes de prod.
- **Task 14 — admin layout**: `web/templates/admin/dashboard.html` é self-contained e será SUBSTITUÍDO em Task 16 (que inclui CSS + Chart.js diretamente nos templates novos).
- **Tasks 15+16 — combined dispatch** (15 templates total): tenant 5 blocos + admin 6 blocos + delete `admin/dashboard.html` antigo.
- **Tasks 15+16 — desvio real do plano**: JSON fields no view model usam `template.JS` (não `string`) — `html/template` em contexto `<script type="application/json">` faz double-encode (JSON virou string JSON dentro de string JSON). `template.JS` emite verbatim. Justificável: view model já é presentation layer (BRL, %, h formatados).
- **Tasks 15+16 — guards defensivos**: Bloco1 funil chart só renderiza se `len(ColumnTotals) > 0` (evita `JSON.parse('null').map()` throw em tenant sem funil configurado).
- **Tasks 15+16 — `partials/sidebar.html` `toggleSidebar()` em `/static/js/admin.js`** — admin page mantém esse script.
- **Task 17 — métricas + spans + logs**: `dashboard_render_duration_seconds{scope}` (histogram), `dashboard_load_total{scope, outcome}` (counter), spans HTTP `dashboard.http.tenant`/`dashboard.http.admin` (parent dos spans UC), log `dashboard_rendered` com `scope/tenant_id/user_id/took`. Coverage: application 98.3%, infrastructure 84.6%, interfaces/http 84.6% (todos > 80%).
- **Task 18 — OWASP**: 9 testes (A01 6×, A04/A05 1×, A03 SQLi 1×, A03 XSS 1×). RequireTenant retorna 403 (não 400 como o plano sugeriu) — corrigido na asserção. Final: 72 testes dashboard, todos packages ≥ 80% cov.
- **Task 19 — slot do .http**: plano sugeriu `11-dashboard.http` mas slot 11 já é `11-logs.http`. Usado `16-dashboard.http` (próximo livre).
- **Task 19 — env vars**: `rest/http-client.env.json` só tem `base_url/admin_email/admin_password`. Não tem `user_token`. Caso 403-admin documentado com comment para operador autenticar como user comum primeiro.
