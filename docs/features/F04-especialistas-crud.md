# F04 - CRUD de Especialistas (Admin)

## Objetivo
Permitir que o admin crie, edite, liste e exclua especialistas (agentes de IA) e os associe a tenants.

## Pré-requisitos
- F03 (CRUD tenants)

## Status: concluído

## Steps

### Step 1: Domínio de especialista
- [x] criar entidade Specialist (id, nome, descricao, prompt, status, created_at, updated_at)
- [x] criar entidade SpecialistTenant (specialist_id, tenant_id) para associação N:N
- [x] criar repositório de Specialist
- [x] migration para tabelas specialists e specialist_tenants
- [x] testes unitários do domínio
- [x] testes de integração do repositório

### Step 2: Casos de uso básicos
- [x] criar especialista
- [x] listar especialistas (com filtros)
- [x] buscar especialista por ID
- [x] editar especialista
- [x] excluir especialista
- [x] testes unitários

### Step 3: Associação com tenants
- [x] caso de uso: associar especialista a tenant(s)
- [x] caso de uso: desassociar especialista de tenant
- [x] caso de uso: listar tenants de um especialista
- [x] caso de uso: listar especialistas de um tenant
- [x] testes

### Step 4: Edição de prompt
- [x] campo de texto rico para editar prompt do especialista
- [x] validação básica (não vazio, tamanho máximo)
- [x] histórico de alterações do prompt (opcional, recomendado) — adiado para F05
- [x] testes

### Step 5: Telas admin (HTMX)
- [x] template de listagem de especialistas
- [x] template de formulário de criação/edição
- [x] template de detalhe do especialista
- [x] seção de associação com tenants (multi-select)
- [x] editor de prompt
- [x] interações via HTMX

## Critérios de aceite
- [x] admin consegue criar, listar, editar e excluir especialistas
- [x] associação com tenants funciona (N:N)
- [x] prompt é editável pela interface
- [x] cobertura >= 80% (87.8%)

## Entrega
- **Branch**: main
- **Commit**: feat(F04): CRUD de especialistas com associação a tenants
- **Testes**: 128 (specialist) / 327 (projeto)
- **Cobertura**: 87.8%
- **Artefatos**: docs/artefatos/F04-especialistas-crud/
