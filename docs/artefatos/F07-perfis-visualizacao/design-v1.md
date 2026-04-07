# Feature: Perfis de Visualizacao no Kanban — Design Spec

**Data**: 2026-04-07
**Status**: Aprovado
**Prioridade**: 2 (segunda a implementar)

## Contexto

O quadro de leads (kanban) exibe todas as colunas do funil. Para equipes grandes, isso pode ser visualmente poluido. A feature permite criar "perfis de visao" — presets nomeados que filtram quais colunas sao exibidas.

## Regras de Negocio

- **Grupo**: por padrao ve todas as colunas (sem restricao). Owner/admin pode personalizar.
- **Perfis do grupo**: ate 10 por grupo por funil. Criados pelo owner/admin do tenant.
- **Perfis individuais**: ate 2 por usuario por funil. Criados pelo proprio usuario.
- Sem perfil ativo = todas as colunas visiveis (default).
- Perfil filtra apenas a visualizacao — nao afeta permissoes de mover leads.

## Modelo de Dados

Nova entidade `KanbanViewProfile`:

```
kanban_view_profiles
├── id           UUID PK
├── tenant_id    UUID FK NOT NULL
├── funnel_id    UUID FK NOT NULL
├── group_id     UUID FK NULLABLE (perfil do grupo)
├── user_id      UUID FK NULLABLE (perfil individual)
├── name         VARCHAR(100) NOT NULL
├── column_ids   JSON NOT NULL (array de UUIDs)
├── created_at   TIMESTAMP
├── updated_at   TIMESTAMP
```

**Constraints**:
- CHECK: `group_id IS NOT NULL OR user_id IS NOT NULL` (ao menos um preenchido)
- Unique: `(tenant_id, funnel_id, group_id, name)` para perfis de grupo
- Unique: `(tenant_id, funnel_id, user_id, name)` para perfis individuais

**Validacoes na aplicacao**:
- Max 10 perfis por grupo por funil
- Max 2 perfis por usuario por funil
- `column_ids` deve conter apenas IDs de colunas existentes no funil

## Modulo

Estende `internal/funnel/`:

- `domain/kanban_view_profile.go` — entidade + validacoes de limite
- `application/manage_view_profiles.go` — Create, Update, Delete, List (por grupo, por usuario)
- `application/get_kanban.go` — recebe `profile_id` opcional, filtra colunas do output
- `infrastructure/gorm_view_profile_repository.go` — CRUD + contagem para validar limites

## UI — Seletor no Kanban

- Dropdown "Visao" posicionado ao lado do seletor de funil
- Opcoes agrupadas:
  - "Todas as colunas" (default)
  - --- Perfis do grupo ---
  - [lista de perfis do grupo do usuario]
  - --- Meus perfis ---
  - [lista de perfis individuais]
- `hx-get="/tenant/leads/kanban?profile_id=XXX"` recarrega o board
- Combinavel com outros filtros (produto, responsavel, busca)

## UI — Gestao de Perfis do Grupo

- Acessivel pelo owner/admin na tela de gestao do grupo (F08 screens)
- Formulario: nome + checkboxes das colunas do funil selecionado
- Lista de perfis existentes com editar/excluir
- Indicador de limite (ex: "3 de 10 perfis usados")

## UI — Perfis Individuais

- Botao "Salvar visao atual" no kanban (ao lado do dropdown de visao)
- Modal: nomear o perfil, confirmar colunas visiveis
- Se ja tem 2 perfis: mensagem "Limite atingido — exclua um perfil para criar outro"
- Gestao dos perfis individuais no proprio dropdown (icone de editar/excluir ao lado)

## Dependencias

- F08 screens (gestao de grupos) para a UI de perfis do grupo
- GetKanbanUseCase existente para integracao do filtro

## Testes

- Unitarios: validacao de limites (10 grupo, 2 individual), filtragem de colunas
- Integracao: CRUD de perfis, kanban com profile_id
- OWASP: usuario nao pode acessar perfis de outro tenant, nao pode criar perfis para grupos que nao pertence
