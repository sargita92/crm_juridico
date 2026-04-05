# F03 - CRUD de Tenants (Admin)

## Objetivo
Permitir que o admin crie, liste, edite e desative tenants pelo painel administrativo.

## Pré-requisitos
- F02 (autenticação e multitenancy)

## Steps

### Step 1: Casos de uso
- [x] criar caso de uso: criar tenant
- [x] criar caso de uso: listar tenants (com filtros e paginação)
- [x] criar caso de uso: buscar tenant por ID
- [x] criar caso de uso: editar tenant
- [x] criar caso de uso: desativar tenant
- [x] testes unitários de cada caso de uso

### Step 2: Handlers HTTP
- [x] POST /admin/tenants (criar)
- [x] GET /admin/tenants (listar)
- [x] GET /admin/tenants/:id (detalhe)
- [x] PUT /admin/tenants/:id (editar)
- [x] DELETE /admin/tenants/:id (desativar)
- [x] middleware de autorização (somente admin)
- [x] testes de integração dos handlers

### Step 3: Bloqueio/Desbloqueio
- [x] criar caso de uso: bloquear tenant (com motivo obrigatório)
- [x] criar caso de uso: desbloquear tenant (com motivo obrigatório)
- [x] POST /admin/tenants/:id/block
- [x] POST /admin/tenants/:id/unblock
- [ ] registrar histórico de bloqueios/desbloqueios (pendente — ver nota abaixo)
- [x] tenant bloqueado não consegue acessar a plataforma
- [x] testes

### Step 4: Telas admin (HTMX)
- [x] template de listagem de tenants (com busca e filtros)
- [x] template de formulário de criação/edição
- [x] template de detalhe do tenant
- [x] modal/form de bloqueio/desbloqueio com campo de motivo
- [x] interações via HTMX (sem reload de página)

### Extras implementados (fora da spec original)
- [x] GET /admin/login — login dedicado para admin
- [x] POST /admin/login — login com redirect para /admin/dashboard
- [x] GET /admin — redirect para login ou dashboard
- [x] GET /admin/dashboard — dashboard admin com sidebar
- [x] layout admin com sidebar reutilizável
- [x] otimização de testes (shared container, 3m25s → 18s)

## Critérios de aceite
- [x] admin consegue criar, listar, editar e desativar tenants
- [x] bloqueio/desbloqueio funciona com motivo registrado
- [x] tenant bloqueado não acessa o sistema
- [x] interface intuitiva e responsiva
- [x] cobertura >= 80% (86.1%)

## Nota: histórico de bloqueios
O item "registrar histórico de bloqueios/desbloqueios" não foi implementado nesta iteração. Atualmente apenas o motivo mais recente é armazenado no campo `block_reason` da tabela `tenants`. Para implementar o histórico completo será necessário:
- migration para tabela `tenant_block_history` (id, tenant_id, action, reason, created_by, created_at)
- entidade de domínio, repositório e extensão nos use cases de block/unblock

Pode ser adicionado como melhoria incremental sem impacto no que já está entregue.
