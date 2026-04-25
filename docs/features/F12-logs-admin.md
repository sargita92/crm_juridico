# F12 - Logs (Admin)

## Objetivo
Implementar a área de logs centralizada no painel administrativo.

## Pré-requisitos
- F01 (setup inicial com Zap)

## Steps

### Step 1: Domínio de logs
- [x] criar entidade AuditLog (id, tenant_id, user_id, acao, entidade, entidade_id, detalhes, ip, created_at)
- [x] migration
- [x] testes unitários

### Step 2: Middleware de auditoria
- [x] middleware que registra ações relevantes automaticamente
- [x] ações: login, CRUD de entidades, bloqueio/desbloqueio, alteração de permissão
- [x] propagar contexto do usuário e tenant
- [x] testes

### Step 3: Casos de uso
- [x] listar logs (com filtros: tenant, usuário, ação, período)
- [x] buscar log por ID
- [x] exportar logs (CSV)
- [x] testes

### Step 4: Telas admin (HTMX)
- [x] template de listagem de logs com filtros
- [x] detalhe do log
- [x] paginação
- [x] botão de exportar

## Critérios de aceite
- ações relevantes são registradas automaticamente
- admin consegue consultar e filtrar logs
- exportação funciona
- cobertura >= 80%
