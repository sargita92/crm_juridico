# F25 — Filtro de Dashboard por Usuário — Design Spec (v1)

**Data**: 2026-05-26
**Status**: Aprovado (brainstorming)
**Feature**: [../../features/F25-dashboard-filtro-por-usuario.md](../../features/F25-dashboard-filtro-por-usuario.md)
**Depende de**: F19 (módulo `internal/dashboard`), F08 (usuários/`user_tenants`)

## Contexto e ponto de partida

O backend do dashboard tenant **já filtra por usuário**. O use case
`GetTenantDashboard.Execute` recebe `TenantInput{TenantID, UserID, IsOwner}` e
calcula um `userFilter *string` que percorre os 5 blocos
(`FunilBlock`, `WhatsAppBlock`, `ResponsaveisBlock`, `TempoFunilBlock`,
`ProdutosBlock`). Hoje:

- `IsOwner == true` → `userFilter = nil` → vê o tenant inteiro (consolidado);
- `IsOwner == false` → `userFilter = &UserID` → vê só os próprios dados.

O handler sempre passa `claims.UserID`, então **não existe** caminho para o owner
ver o dashboard de **outro** usuário. A UI já tem o conceito de "de quem é a
visão": `domain.TenantStats.ScopeIsUser` + `CurrentUserName`, renderizado como um
`scope-badge`.

**F25 = expor um seletor de operador para o owner**, reutilizando a fiação de
filtro que já existe. Sem migration nova.

## Decisão de arquitetura

**Abordagem A — `ViewUserID *string` explícito no `TenantInput`** (escolhida).
Separa "quem pede" (`UserID`, `IsOwner`) de "o que vê" (`ViewUserID`). A política
de domínio fica no use case; o handler valida o pertencimento (OWASP) na borda,
onde o repositório de `user_tenants` já está disponível, e passa o `ViewUserID`
**já validado**.

Descartadas: (B) reaproveitar `UserID`/`IsOwner` no handler — conflagra identidade
com escopo, ruim para log/auditoria; (C) o handler resolver o `*string` de filtro
— vaza a política "não-owner só vê o próprio" para o handler (viola "handlers
finos, domínio isolado").

## Mudanças por camada

### Domínio (`internal/dashboard/domain`)
- Novo tipo `Operator{ ID string; Name string }` (item de lista do seletor).

### Aplicação (`internal/dashboard/application`)
- `TenantInput` ganha `ViewUserID *string`.
- `Execute` passa a decidir o filtro:
  ```go
  var userFilter *string
  switch {
  case !in.IsOwner:           userFilter = &in.UserID    // não-owner: travado em si mesmo
  case in.ViewUserID != nil:  userFilter = in.ViewUserID // owner drillando num operador
  default:                    userFilter = nil           // owner: consolidado
  }
  ```
- `CurrentUserName` passa a resolver o **usuário efetivamente filtrado**
  (`*userFilter`), não mais sempre `in.UserID`. Para não-owner é idêntico; para
  owner em drill-down resolve o nome do operador.
- Novo port `OperatorLister` (interface definida na aplicação):
  ```go
  type OperatorLister interface {
      Operators(ctx context.Context, tenantID string) ([]domain.Operator, error)
  }
  ```
  Lista os operadores do tenant para popular o dropdown.

> Nota de fronteira: a **listagem** dos operadores é dado de apresentação do
> seletor. O port `OperatorLister` é **injetado no `Handler`** (novo parâmetro de
> `NewHandler`) e chamado ao montar a view — **não** entra no `GetTenantDashboard`,
> que continua só calculando stats de um único escopo. A wiring é feita no
> `module.go` do dashboard.

### Infraestrutura (`internal/dashboard/infrastructure`)
- Implementação Gorm de `Operators(ctx, tenantID)`: **um JOIN**
  `user_tenants ⋈ users` com `tenant_id = ? AND is_owner = false`, selecionando
  `users.id, users.name`, ordenado por `users.name`. Sem N+1.
- Validação de pertencimento **reusa** `authdomain.UserTenantRepository.FindByUserAndTenant(userID, tenantID)`:
  válido se encontrou **e** `!IsOwner`. Nenhum método novo de validação.

### Interfaces HTTP (`internal/dashboard/interfaces/http`)
- `renderTenant`:
  1. determina `isOwner` (já existe; admin de plataforma → owner);
  2. se `isOwner`: monta a lista de operadores (`OperatorLister.Operators`);
  3. lê `?user=<id>`. Se `isOwner` **e** o id for operador válido do tenant →
     `ViewUserID = &id` e `SelectedUserID = id`; se inválido → ignora (consolidado)
     e loga `warn`. Se **não** for owner → `?user` é sempre ignorado;
  4. chama `Execute(TenantInput{TenantID, UserID: claims.UserID, IsOwner, ViewUserID})`;
  5. passa ao template: `Stats`, `TenantID`, `Operators`, `SelectedUserID`,
     `CanSelectUser` (= `isOwner`).
- Log `dashboard_rendered` ganha `viewed_user_id` quando há drill-down.

### UI / Templates (`web/templates/tenant/dashboard`)
- **Seletor** no `head-actions` de `page.html`, renderizado só quando
  `CanSelectUser`:
  ```html
  <select name="user" class="dashboard-user-select"
          hx-get="/dashboard/content"
          hx-target="#dashboard-content"
          hx-swap="outerHTML"
          hx-trigger="change">
      <option value="">Consolidado (todos)</option>
      {{range .Operators}}
      <option value="{{.ID}}" {{if eq .ID $.SelectedUserID}}selected{{end}}>{{.Name}}</option>
      {{end}}
  </select>
  ```
  O `<select>` envia `?user=<value>` automaticamente (htmx serializa o `name`).
- **Botão "Atualizar"**: ganha `hx-include="[name='user']"` para preservar a
  seleção no refresh.
- **Sincronia do indicador "vendo: X"**: hoje o `scope-badge` está no `<h1>`
  (page.html), **fora** do `#dashboard-content` trocado — ficaria desatualizado
  num swap parcial. **Correção**: mover o indicador de escopo para **dentro** do
  fragmento `content.html` (topo do `#dashboard-content`), recarregando junto com
  os dados. O `<select>` permanece no header (fora do swap), mantendo o valor
  escolhido entre trocas; no full page load com `?user=<id>` a opção certa vem
  pré-selecionada pelo template.

## Fluxo (sequência)

```
Owner → GET /dashboard[?user=<id>]
  handler: isOwner? → lista operadores; valida ?user; monta ViewUserID
  Execute(TenantInput{..., ViewUserID})
    userFilter = ViewUserID (owner+drill) | nil (owner+consolidado) | &self (não-owner)
    5 blocos filtrados por (tenantID, userFilter)
    CurrentUserName = nome do usuário filtrado (se houver)
  template: header com <select> + fragmento #dashboard-content (blocos + "vendo: X")

Owner troca o <select> → htmx GET /dashboard/content?user=<id>
  → mesmo handler → troca só #dashboard-content
```

## Segurança (OWASP — regra 13)

| Cenário | Comportamento |
|---------|---------------|
| Não autenticado | 401 (middleware `Auth`) |
| Operador comum com `?user=<outro>` | `?user` ignorado → vê só o próprio (escopo travado em `claims.UserID`) |
| Owner com `?user=<id de outro tenant>` | `FindByUserAndTenant` não acha → consolidado; log `warn` |
| Owner com `?user=<não-membro / inexistente>` | idem → consolidado |
| Owner com `?user=<id de um owner>` | `IsOwner==true` na validação → rejeitado → consolidado |
| Isolamento de tenant | todas as queries filtram por `tenantID`; `ViewUserID` só é setado após validar pertencimento |

Defesa em profundidade: mesmo que um id inválido escapasse, as queries dos blocos
filtram por `tenantID AND responsible_user_id = ?`, então um usuário fora do
tenant não retornaria dados.

## Observabilidade (regra 11)

- Log `dashboard_rendered`: campos existentes (`scope`, `tenant_id`, `user_id`,
  `scope_is_user`, `took`) + novo `viewed_user_id` quando há drill-down.
- Métricas existentes (`RenderDuration`, `LoadTotal`) inalteradas.

## Testes (TDD, ≥ 80%)

- **application** (`get_tenant_dashboard_test.go`):
  - owner + `ViewUserID` setado → filtra por aquele usuário;
  - owner + `ViewUserID == nil` → consolidado (filtro nil);
  - não-owner + `ViewUserID` setado → **ignorado**, travado em `UserID`;
  - `CurrentUserName` resolve o usuário filtrado (não o requisitante).
- **infrastructure** (`gorm_*_test.go`): `Operators` retorna só não-owners, com
  nome, ordenado por nome, escopado ao tenant; tenant sem operadores → vazio.
- **interfaces/http**:
  - `tenant_handler_test.go`: monta dropdown só p/ owner; parse e validação de
    `?user`; `SelectedUserID`/`CanSelectUser` no view model;
  - `owasp_test.go`: os cenários da tabela de segurança acima.
- **view** (`view_test.go`): mapeamento de `Operator`/seleção no view model.

## Rotas e contrato

```
GET /dashboard?user=<id>           → página tenant (HTML completo)
GET /dashboard/content?user=<id>   → fragmento #dashboard-content (HTMX)
```
`user` opcional; honrado apenas para owner; valor inválido → consolidado.
`rest/dashboard.http` atualizado com o parâmetro e os casos OWASP.

## Fora de escopo (YAGNI)

Comparação multi-usuário; persistência da seleção; filtros de período; export;
seletor no dashboard admin. **Sem migration nova.**
