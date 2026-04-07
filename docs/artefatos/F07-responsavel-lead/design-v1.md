# Feature: Responsavel no Lead — Design Spec

**Data**: 2026-04-07
**Status**: Aprovado
**Prioridade**: 1 (primeira a implementar)

## Contexto

O modelo `Lead` ja possui `ResponsibleUserID` (migration 000043) e o `AssignLeadUseCase` esta implementado. Falta a parte visual: exibir o responsavel no card do kanban, permitir atribuicao no drawer de detalhes, e filtrar o board por responsavel.

## Escopo

### Backend (ajustes)

1. **GetKanbanUseCase**: incluir nome do responsavel no output `KanbanLead` (novo campo `ResponsibleUserName string`)
2. **FindByFunnelID**: adicionar filtro opcional `ResponsibleUserID` no `LeadFilter`
3. **Handler do kanban**: aceitar query param `responsible` e repassar ao use case

### UI — Card do Kanban

- Nome do responsavel exibido no card (ex: "Joao Silva")
- Icone de pessoa pequeno antes do nome
- Nomes longos truncados com `text-overflow: ellipsis` (max-width adequado ao card)
- Se nao atribuido: texto "Sem responsavel" em cinza claro

### UI — Drawer de Detalhes

- Secao "Responsavel" (placeholder existente) vira dropdown funcional
- Lista usuarios do tenant (via adapter `UserNameProvider` ou similar)
- Botao "Atribuir" / "Trocar" faz PUT `/tenant/leads/:id/assign` via HTMX
- Se nao atribuido: "Sem responsavel — Atribuir"
- Ao atribuir, card do kanban atualiza via HTMX swap

### UI — Filtro no Kanban

- Novo dropdown no topo do kanban ao lado dos filtros existentes (produto, busca)
- Opcoes: "Todos" (default) | "Meus leads" (atalho para usuario logado) | Lista de usuarios do tenant
- `hx-get="/tenant/leads/kanban?responsible=USER_ID"` recarrega o board filtrado
- Combinavel com os filtros existentes (produto, busca)

## Modelo de Dados

Nenhuma alteracao — `Lead.ResponsibleUserID` e `AssignLeadUseCase` ja existem.

## Dependencias

- `UserNameProvider` (adapter existente) para resolver nomes
- Lista de usuarios do tenant (repositorio existente em auth)

## Testes

- Unitarios: filtro por responsavel no repositorio, nome no output do kanban
- Integracao: handler com query param responsible
- OWASP: isolamento de tenant no filtro, usuario so ve leads do proprio tenant
