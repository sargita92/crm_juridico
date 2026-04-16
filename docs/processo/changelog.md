# Changelog

Registro histórico de entregas do projeto.

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
