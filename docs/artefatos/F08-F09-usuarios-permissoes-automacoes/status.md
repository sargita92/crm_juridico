# F08 + F09 — Status

## Status: Em andamento

## Planos concluídos

### Plan 1: EventBus Shared + Permission Module (concluído)
- [x] EventBus promovido de whatsapp para shared/events (5 tipos de evento)
- [x] Permission module DDD completo (grupos, permissões union grupo+individual, perfis de visualização, group-funnel)
- [x] RequirePermission middleware
- [x] 16 endpoints HTTP
- [x] 51 tests, 86.8% coverage
- [x] 5 migrations (000035-000039)

### Plan 2: Auth Extensions + Funnel Extensions + Notification Module (concluído)
- [x] Auth: InviteToken (convite por link), LoadBalanceConfig, UserTenant.IsOwner/WhatsAppID
- [x] Auth: Gestão de usuários (listar, remover, WhatsApp)
- [x] Auth: auth module.go criado (antes era wiring direto no main.go)
- [x] Funnel: Lead.ResponsibleUserID, AssignLead endpoint
- [x] Funnel: Eventos lead-created/lead-moved publicados no EventBus
- [x] Notification: módulo DDD completo (6 tipos, SSE stream + polling REST, preferências)
- [x] OwnerChecker funcional com query real no DB
- [x] 6 migrations (000040-000045)
- [x] 167 tests totais, coverage > 80% em todos os módulos

## Pendente

### Plan 3: Automation Module (a fazer)
- [ ] Entidade Automation (7 tipos)
- [ ] AutomationEngine (event-driven, sync/async híbrido)
- [ ] ExpirationTicker (goroutine, 5min)
- [ ] ExecutionLog + RateLimitCounter
- [ ] 7 endpoints HTTP
- [ ] Telas HTMX (automações, permissões, notificações)

### Itens complementares (a fazer)
- [ ] Load balance integration (conectar à criação de leads)
- [ ] Telas HTMX para gestão de grupos e permissões
- [ ] Telas HTMX para convites e gestão de usuários
- [ ] Telas HTMX para notificações (badge, toast, painel)
- [ ] Telas HTMX para automações
- [ ] Testes OWASP nos novos endpoints (401/403, isolamento de tenant)
- [ ] Observabilidade: métricas Prometheus + traces nos novos endpoints
- [ ] Arquivos .http em rest/ para os novos endpoints
