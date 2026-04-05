# F02 - Autenticação e Multitenancy

## Objetivo
Implementar autenticação de usuários e isolamento multitenant com banco de dados único.

## Pré-requisitos
- F01 (setup inicial)

## Steps

### Step 1: Domínio de tenant
- [ ] criar entidade Tenant (id, nome, tipo PF/PJ, documento, status, motivo_bloqueio, created_at, updated_at)
- [ ] criar repositório de Tenant (interface + implementação Gorm)
- [ ] criar migration para tabela tenants
- [ ] testes unitários do domínio
- [ ] testes de integração do repositório

### Step 2: Domínio de usuário
- [ ] criar entidade User (id, nome, email, senha_hash, tenant_id, role, status, created_at, updated_at)
- [ ] criar repositório de User
- [ ] criar migration para tabela users
- [ ] implementar hash de senha (bcrypt)
- [ ] testes unitários e de integração

### Step 3: Autenticação
- [ ] criar caso de uso de login (email + senha)
- [ ] implementar geração e validação de JWT
- [ ] criar middleware de autenticação Gin
- [ ] criar handler HTTP de login
- [ ] testes do caso de uso e do handler

### Step 4: Middleware de tenant
- [ ] criar middleware que extrai tenant_id do contexto do usuário autenticado
- [ ] propagar tenant_id via context.Context
- [ ] implementar scoping automático de queries por tenant_id
- [ ] testes de isolamento entre tenants

### Step 5: Tela de seleção de tenant
- [ ] criar caso de uso para listar tenants do usuário
- [ ] criar handler e endpoint
- [ ] criar template HTMX para seleção de tenant
- [ ] se usuário tem apenas 1 tenant → redirecionar direto
- [ ] se tem mais de 1 → exibir tela de seleção
- [ ] admin vê todos os tenants

### Step 6: Telas de login
- [ ] criar template de login (HTMX)
- [ ] criar template de seleção de tenant (HTMX)
- [ ] fluxo completo: login → seleção de tenant (se necessário) → dashboard

## Critérios de aceite
- usuário consegue fazer login
- JWT válido é gerado e validado
- queries são isoladas por tenant automaticamente
- admin vê todos os tenants
- usuário com 1 tenant vai direto, com vários seleciona
- cobertura >= 80%
