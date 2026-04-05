# F12 - Logs (Admin)

## Objetivo
Implementar a área de logs centralizada no painel administrativo.

## Pré-requisitos
- F01 (setup inicial com Zap)

## Steps

### Step 1: Domínio de logs
- [ ] criar entidade AuditLog (id, tenant_id, user_id, acao, entidade, entidade_id, detalhes, ip, created_at)
- [ ] migration
- [ ] testes unitários

### Step 2: Middleware de auditoria
- [ ] middleware que registra ações relevantes automaticamente
- [ ] ações: login, CRUD de entidades, bloqueio/desbloqueio, alteração de permissão
- [ ] propagar contexto do usuário e tenant
- [ ] testes

### Step 3: Casos de uso
- [ ] listar logs (com filtros: tenant, usuário, ação, período)
- [ ] buscar log por ID
- [ ] exportar logs (CSV)
- [ ] testes

### Step 4: Telas admin (HTMX)
- [ ] template de listagem de logs com filtros
- [ ] detalhe do log
- [ ] paginação
- [ ] botão de exportar

## Critérios de aceite
- ações relevantes são registradas automaticamente
- admin consegue consultar e filtrar logs
- exportação funciona
- cobertura >= 80%
