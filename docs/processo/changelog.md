# Changelog

Registro histórico de entregas do projeto.

---

## [2026-04-18] F07 — Funis de Vendas (Kanban) — fechamento

Feature encerrada com validação QA + review de segurança (ambos aprovados):

- **QA validação (66 cenários)**: todos cobertos por testes automatizados.
  Gaps preenchidos: CT-33 (refresh de `column_entered_at` no `Lead.MoveTo`),
  CT-64 (JWT tampering — assinatura adulterada/corrompida retorna 401),
  CT-65/CT-66 (audit logs de tentativa cross-tenant e de movimentação de
  lead, validados via `zaptest/observer`).
- **Audit logging opcional** adicionado em `MoveLeadUseCase` e
  `GetLeadDetailUseCase` (`SetAuditLogger`). Emite `INFO "lead moved"`
  com tenant_id, lead_id, funnel_id, from/to column_id no sucesso, e
  `WARN "cross-tenant lead access denied"` em isolamento quebrado.
- **Security review (OWASP Top 10)**: APROVADO sem achados alto/crítico.
  Recomendações nice-to-have (rate limiting global, user_id no audit)
  ficaram para follow-up.
- **Cobertura por pacote** (todos ≥ 80%):
  - domain 89.7% · application 83.5% · infrastructure 88.6% ·
    interfaces/http 86.3%
  - infrastructure saltou de 3.5% → 88.6% com 48 novos testes de
    integração (testcontainers-go) + adapters.
  - interfaces/http saltou de 15.6% → 86.3% com 36 novos testes de
    handler cobrindo os 17 endpoints.
- **Suite total**: 1703 testes passando.
- Artefatos: `docs/artefatos/F07-funis-kanban/qa-validacao/v1.md` e
  `seguranca-review/v1.md`; backlog atualizado para `concluído`.

---

## [2026-04-18] F08 Step 6 — Telas de Equipe (HTMX)

- Novo item "Equipe" no sidebar do tenant (visível com `users:read` OU `groups:manage`)
- Rota `/tenant/team` com 2 abas: **Usuários** e **Grupos**
- Aba Usuários: lista + convites pendentes, modal de convite (link copiável),
  modal de permissões individuais (override sobre as herdadas do grupo),
  modal de WhatsApp ID, remoção (bloqueada para owner)
- Aba Grupos: lista + criação; cada grupo abre um detail com 5 seções:
  - 👥 Membros (adicionar/remover, seleciona entre usuários do tenant)
  - 🔐 Permissões (matriz resource × action)
  - 🎯 Funis atribuídos (toggle de funis do tenant)
  - 👁️ Perfis de visualização (colunas visíveis por funil)
  - ⚖️ Load Balance (algoritmo + toggle ativo)
- Backend de load balance concluído: `ManageLoadBalanceUseCase`, campo `active`
  (migration 000051), endpoints `GET/PUT /tenant/groups/:id/load-balance`
- Cross-module wiring via `AttachPermissionDeps` no auth module + accessors
  (`ListGroupsUseCase`, `ManagePermissionsUseCase`) no permission module
- 19 tests no `auth.PageHandler` (91% cobertura), 37 tests no
  `permission.PageHandler` (≥85% por função), 103 tests OWASP (401/403/tenant
  isolation), 1569 tests totais no repositório
- Artefatos: design spec + plano em
  `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/`,
  arquivo `rest/team.http` para testes manuais

---

## [2026-04-18] F17 — AI Playground e robustez do pipeline

Ferramenta interna de desenvolvimento para exercitar o motor de IA sem
envolver o WhatsApp real, mais robustez colhida no caminho:

- **Playground UI** (`/tenant/ai/playground`): sidebar de contatos + chat
  com polling de 2s, botão de reset que zera state e volta o lead à
  coluna inicial
- **Scoring-driven routing**: `ConversationEngine` consulta o `ScoringConfig`
  do especialista para qualificar/desqualificar leads automaticamente
  (veto do LLM > `TargetColumnID` explícito > threshold atingido > flow
  completo sem threshold); `ScoringConfigFinder` opcional por DI
- **WhatsApp DM-only**: filtra mensagens que não são 1:1 (groups/status
  broadcast) e usa `ToNonAD()` no JID do sender para evitar rejeição no
  send; AI processing passou a usar `context.WithoutCancel` para não
  cancelar ao final do HTTP request
- **Hardening cross-cutting**:
  - Lookup produto → funil agora é tenant-scoped (regressão:
    `TestCreateLead_ProductRoutesToOtherTenantFunnel_IsIgnored`)
  - `Lead.ProductID`/`ResponsibleUserID` e `LeadMovement.FromColumnID`
    persistem como `NULL` quando vazios (antes: empty string violava FK)
- Scripts `create-playground-lead.sh` / `test-playground.sh` e fixture
  `escritorio-teste.sql` para seed rápido

---

## [2026-04-16] F15 — Internal Tool Registry para Especialistas IA

- Domain: `ToolDefinition`, `ToolCall`, `ToolResult`, `ToolCategory`, `ParameterDef`, `Tool` interface
- `AIRequest`/`AIResponse` estendidos com `Tools`, `ToolResults`, `ToolCalls`
- `ToolRegistry` e `ToolResolver` (filtragem por especialista + steps com `ForcedTools`/`RestrictedTools`)
- Tool calling loop no `ConversationEngine` (max iterations, timeout, truncate)
- OpenAI provider com function calling nativo (provider-agnostic por design)
- 10 tools em 3 categorias:
  - Consulta: `search_leads`, `get_lead_detail`, `get_conversation_history`, `list_products`, `get_pipeline`
  - CRM: `move_lead`, `create_note`, `update_score`
  - Automação: `trigger_automation`, `switch_specialist`
- Entidade `SpecialistTool` + tabela `specialist_tools` (migration 49) + campos `forced_tools`/`restricted_tools` no step (migration 50)
- Admin UI HTMX para associação tool↔especialista em `/admin/specialists/:id/tools`
- 4 métricas Prometheus (`tool_calls_total`, `tool_call_duration_seconds`, `tool_loop_iterations`, `tool_result_truncated_total`)
- Limites configuráveis via env (`AI_TOOL_LOOP_MAX_ITERATIONS`, `AI_TOOL_EXECUTION_TIMEOUT_SECONDS`, `AI_TOOL_RESULT_MAX_LENGTH`, `AI_TOOL_CALL_MAX_PER_ITERATION`)
- Segurança: `tenantID` sempre do contexto (nunca do LLM), validação de args em todas as tools, erros retornam `ToolResult{IsError: true}` (não crasham conversa)
- 1404 testes passando, coverage core (domain+application) 92%
- Artefatos: design spec (`docs/features/F15-internal-tool-registry-design.md`), plano (`F15-internal-tool-registry-plan.md`)

---

## [2026-04-05] F02 — Autenticação e Multitenancy

- Entidade Tenant (PF/PJ, status, bloqueio) com repositório GORM
- Entidade User com bcrypt, relação N:N com Tenant via user_tenants
- Login com JWT (HS256, expiração configurável), cookie HttpOnly + SameSite Lax
- Middleware Auth (cookie/Bearer), middleware RequireTenant, TenantScope GORM
- Tela de login (HTMX, toggle senha, loading state, erro genérico)
- Tela de seleção de tenant (cards PF/PJ, admin vê todos)
- Dashboard placeholder
- 3 migrations reversíveis (tenants, users, user_tenants)
- 83 testes, cobertura F02 87.6%
- Segurança: 3 vulnerabilidades encontradas e corrigidas (err.Error() exposto, cookie Secure, SameSite)
- Artefatos: stories, wireframes, design técnico, cenários QA, validação QA, review segurança
