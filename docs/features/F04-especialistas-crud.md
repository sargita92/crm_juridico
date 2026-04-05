# F04 - CRUD de Especialistas (Admin)

## Objetivo
Permitir que o admin crie, edite, liste e exclua especialistas (agentes de IA) e os associe a tenants.

## Pré-requisitos
- F03 (CRUD tenants)

## Steps

### Step 1: Domínio de especialista
- [ ] criar entidade Specialist (id, nome, descricao, prompt, status, created_at, updated_at)
- [ ] criar entidade SpecialistTenant (specialist_id, tenant_id) para associação N:N
- [ ] criar repositório de Specialist
- [ ] migration para tabelas specialists e specialist_tenants
- [ ] testes unitários do domínio
- [ ] testes de integração do repositório

### Step 2: Casos de uso básicos
- [ ] criar especialista
- [ ] listar especialistas (com filtros)
- [ ] buscar especialista por ID
- [ ] editar especialista
- [ ] excluir especialista
- [ ] testes unitários

### Step 3: Associação com tenants
- [ ] caso de uso: associar especialista a tenant(s)
- [ ] caso de uso: desassociar especialista de tenant
- [ ] caso de uso: listar tenants de um especialista
- [ ] caso de uso: listar especialistas de um tenant
- [ ] testes

### Step 4: Edição de prompt
- [ ] campo de texto rico para editar prompt do especialista
- [ ] validação básica (não vazio, tamanho máximo)
- [ ] histórico de alterações do prompt (opcional, recomendado)
- [ ] testes

### Step 5: Telas admin (HTMX)
- [ ] template de listagem de especialistas
- [ ] template de formulário de criação/edição
- [ ] template de detalhe do especialista
- [ ] seção de associação com tenants (multi-select)
- [ ] editor de prompt
- [ ] interações via HTMX

## Critérios de aceite
- admin consegue criar, listar, editar e excluir especialistas
- associação com tenants funciona (N:N)
- prompt é editável pela interface
- cobertura >= 80%
