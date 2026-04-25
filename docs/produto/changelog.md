# Changelog

Histórico de features entregues, em ordem cronológica reversa.

## 2026-04-24 — F12 - Logs (Admin) - concluído

Auditoria centralizada de ações admin e de segurança: login (sucesso e falha), CRUD de tenants, CRUD de usuários admin, bloqueio/desbloqueio de tenant e alteração de permissões. Consulta com filtros (tenant, usuário, ação, período) + paginação + detalhe. Rota `/admin/logs` restrita ao admin global; usuários não-admin recebem 404 genérico (sem revelar a existência da rota). Sem exportação CSV no MVP; retenção ilimitada.

**Steps entregues:**
- Step 1: domínio (entidade `AuditLog`, enum `Action`, value object `AuditLogFilter`, erros).
- Step 2: migration `000056_create_audit_logs` + repositório Gorm + integration tests.
- Step 3: caso de uso `RegisterAuditLog` + `AuditPublisher` + helper `BuildDiff`.
- Step 4: casos de uso `ListAuditLogs` e `GetAuditLog`.
- Step 5: integração com `auth` (login/logout/falha + email no claim).
- Step 6: integração com `tenants` (criação, edição, bloqueio/desbloqueio, desativação).
- Step 7: integração com `usuários admin` + alteração de permissões.
- Step 8: handlers HTTP + templates HTMX + middleware 404 admin.
- Step 9: observabilidade (smoke test + registro de métricas).
- Step 10: REST `.http` + changelog + backlog + status.

**Observabilidade:** 2 métricas Prometheus
- `crm_audit_logs_registered_total` — counter de logs registrados (label `action`).
- `crm_audit_logs_list_duration_seconds` — histograma de duração da listagem.
