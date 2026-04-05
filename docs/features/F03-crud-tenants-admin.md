# F03 - CRUD de Tenants (Admin)

## Objetivo
Permitir que o admin crie, liste, edite e desative tenants pelo painel administrativo.

## Pré-requisitos
- F02 (autenticação e multitenancy)

## Steps

### Step 1: Casos de uso
- [ ] criar caso de uso: criar tenant
- [ ] criar caso de uso: listar tenants (com filtros e paginação)
- [ ] criar caso de uso: buscar tenant por ID
- [ ] criar caso de uso: editar tenant
- [ ] criar caso de uso: desativar tenant
- [ ] testes unitários de cada caso de uso

### Step 2: Handlers HTTP
- [ ] POST /admin/tenants (criar)
- [ ] GET /admin/tenants (listar)
- [ ] GET /admin/tenants/:id (detalhe)
- [ ] PUT /admin/tenants/:id (editar)
- [ ] DELETE /admin/tenants/:id (desativar)
- [ ] middleware de autorização (somente admin)
- [ ] testes de integração dos handlers

### Step 3: Bloqueio/Desbloqueio
- [ ] criar caso de uso: bloquear tenant (com motivo obrigatório)
- [ ] criar caso de uso: desbloquear tenant (com motivo obrigatório)
- [ ] POST /admin/tenants/:id/block
- [ ] POST /admin/tenants/:id/unblock
- [ ] registrar histórico de bloqueios/desbloqueios
- [ ] tenant bloqueado não consegue acessar a plataforma
- [ ] testes

### Step 4: Telas admin (HTMX)
- [ ] template de listagem de tenants (com busca e filtros)
- [ ] template de formulário de criação/edição
- [ ] template de detalhe do tenant
- [ ] modal/form de bloqueio/desbloqueio com campo de motivo
- [ ] interações via HTMX (sem reload de página)

## Critérios de aceite
- admin consegue criar, listar, editar e desativar tenants
- bloqueio/desbloqueio funciona com motivo registrado
- tenant bloqueado não acessa o sistema
- interface intuitiva e responsiva
- cobertura >= 80%
