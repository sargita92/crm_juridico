# Design — F08 Step 4/4.1: Load Balance integrado ao fluxo de criação de lead

**Data:** 2026-04-18
**Feature:** F08 (Usuários e Permissões)
**Escopo:** Step 4 (load balance) + Step 4.1 (responsável do lead e notificação inicial)
**Status:** design aprovado, aguardando plano de implementação

## 1. Contexto

O backend de load balance está completo (ver [status.md](./status.md)):

- Entidade `LoadBalanceConfig` (`internal/auth/domain/load_balance.go`) com 3 algoritmos (`round_robin`, `least_load`, `random`), campo `active`
- `ManageLoadBalanceUseCase` com CRUD
- Flag `active` em migration 000051
- `GroupFunnel` (`internal/permission`) associa grupo ↔ funil/colunas
- `Lead.ResponsibleUserID` existe em `internal/funnel/domain/lead.go`
- `AssignLeadUseCase` faz reatribuição manual

O que falta é o **elo de ligação**: no momento em que um lead é criado (manualmente, via WhatsApp ou via IA), o sistema deve escolher automaticamente um responsável usando a configuração de load balance do grupo responsável pelo funil/coluna inicial. Hoje todo lead nasce sem responsável.

## 2. Objetivo

Ao criar um lead em qualquer dos três fluxos:

1. Atribuir `ResponsibleUserID` automaticamente antes do `INSERT`
2. Garantir que todo lead nasça com um responsável (sem estado órfão)
3. Emitir evento para que o responsável seja notificado (F08 Step 4.1)
4. Instrumentar o caminho com métricas, logs e traces

## 3. Decisões de design

### 3.1 Cascata de fallback

Todo lead sai do `CreateLeadUseCase` com responsável atribuído. A cascata do picker é:

1. **Caminho feliz:** grupo responsável pelo funil/coluna → `LoadBalanceConfig` ativo → aplica algoritmo sobre membros ativos do grupo
2. **Fallback:** em qualquer falha do caminho feliz (sem grupo, sem config, config inativa, grupo sem membros ativos, erro inesperado de infra) → escolhe o **owner do tenant** (`UserTenant.IsOwner = true`)
3. **Erro duro:** se nem o owner existir (improvável na prática), a criação do lead aborta com erro

### 3.2 Unicidade: 1 grupo com load balance ativo por funil/coluna

Para evitar ambiguidade quando múltiplos grupos estão associados ao mesmo funil/coluna, aplicamos uma **restrição no modelo**: só pode existir **1 grupo com `LoadBalanceConfig` ativo** cobrindo a mesma coluna.

- A validação vive no `ManageLoadBalanceUseCase.SetByGroup` e no endpoint de ativação
- Outros grupos podem estar associados ao funil para **visualização/permissão**, mas não para distribuição
- Tentativa de ativar um segundo grupo que cubra a mesma coluna → 409 Conflict com mensagem explicativa

### 3.3 Definição de "menor carga" (`least_load`)

Carga de um usuário = quantidade de **leads ativos** (`archived = false`) atribuídos a ele **no tenant todo**.

Query conceitual:

```sql
SELECT responsible_user_id, COUNT(*) AS load
FROM leads
WHERE tenant_id = ? AND archived = false AND responsible_user_id IN (...)
GROUP BY responsible_user_id
```

Desempate (usuário sem nenhum lead ativo não aparece na query → load = 0): ordenar pelo `user.created_at` ASC (mais antigo primeiro).

### 3.4 Reatribuição em movimentação de lead

Explicitamente **fora de escopo**: quando um lead muda de coluna/funil (manual, IA ou automação `move_funnel` da F09), o responsável **não** é recalculado. O requisito F08 Step 4.1 fala em **notificar** em movimentações, não em reatribuir. `AssignLeadUseCase` continua sendo o único caminho de reatribuição, e é sempre manual/explícito.

## 4. Arquitetura

### 4.1 Port `ResponsiblePicker` (novo)

Para manter `funnel` desacoplado de `auth`/`permission`, criamos uma porta no domínio do funnel:

```go
// internal/funnel/domain/responsible_picker.go
package domain

import "context"

type PickOutcome string

const (
    PickOutcomePicked         PickOutcome = "picked"
    PickOutcomeFallbackOwner  PickOutcome = "fallback_owner"
)

type PickResult struct {
    UserID    string
    Algorithm string
    GroupID   string  // vazio se outcome == fallback_owner
    Outcome   PickOutcome
}

type ResponsiblePicker interface {
    PickForFunnel(ctx context.Context, tenantID, funnelID, columnID string) (PickResult, error)
}
```

Erros retornados pela porta são erros **duros** (ex: owner inexistente). Casos de fallback não são erros — são `Outcome = fallback_owner`.

### 4.2 Implementação `LoadBalancePicker` (auth infra)

Local: `internal/auth/infrastructure/load_balance_picker.go`.

Dependências injetadas:

- `permission.GroupFunnelRepository` — achar grupo responsável pela coluna
- `permission.GroupMemberRepository` (ou `UserGroupRepository` existente) — membros ativos do grupo
- `auth.LoadBalanceConfigRepository` — config do grupo
- `auth.UserTenantRepository` — owner do tenant (fallback) + `is_active` dos membros
- `funnel.LeadRepository` — contagem para `least_load` (interface pequena e read-only; OK o auth depender de uma porta exposta por funnel — ver 4.3)

Lógica:

```
groups := groupFunnelRepo.FindCoveringColumn(tenantID, funnelID, columnID)
activeGroups := filtrar os que têm LoadBalanceConfig ativo
switch len(activeGroups):
  case 0: → fallback owner
  case 1: → aplicar algoritmo
  case >1: → erro de configuração (não deveria ocorrer pela regra 3.2);
           log ERROR + fallback owner (não quebra produção)

members := groupMemberRepo.FindActiveMembers(groupID)
if len(members) == 0: → fallback owner

switch algorithm:
  round_robin: pick members[(cfg.LastIndex+1) % len(members)]; cfg.LastIndex++; repo.Update(cfg)
  least_load:  counts := leadRepo.CountActiveByUsers(tenantID, memberIDs); pick min (desempate por user.created_at)
  random:      crypto/rand index
```

Atualização de `LastIndex` no round-robin acontece **na mesma transação** do `CreateLead`? Ver 4.4.

### 4.3 Porta read-only em `funnel` para contagem

Para evitar que o `auth` dependa diretamente de um repositório do `funnel`, expomos uma porta pequena em `funnel/domain`:

```go
type LeadLoadCounter interface {
    CountActiveByUsers(ctx context.Context, tenantID string, userIDs []string) (map[string]int, error)
}
```

Implementada pelo repositório GORM do funnel. Wiring injeta essa porta no `LoadBalancePicker`.

### 4.4 Transação e atualização de `LastIndex`

- `CreateLeadUseCase` abre transação (já abre hoje para `INSERT` + evento? verificar no plan; se não, abrir)
- `picker.PickForFunnel` roda **dentro** da transação; a escrita de `LastIndex` (round-robin) participa da mesma transação
- Se `CreateLead` falhar depois do pick, `LastIndex` também faz rollback → consistência preservada
- Alternativa descartada: `LastIndex` em transação separada → causa "pulos" no round-robin quando create falha

## 5. Integração no `CreateLeadUseCase`

Três entry points já existem e compartilham a mesma entidade/repo: `Execute` (HTTP manual), `CreateFromConversation` (WhatsApp), e o fluxo da IA (que chama `CreateFromConversation` ou variante). Todos passam a ter o mesmo bloco.

Fluxo após a mudança:

```
1. Valida input (como hoje)
2. Monta Lead{} sem responsável
3. pick := picker.PickForFunnel(ctx, tenantID, funnelID, columnID)
4. lead.AssignResponsible(pick.UserID)
5. leadRepo.Create(lead)  [mesma tx]
6. eventBus.Publish(EventLeadCreated{..., ResponsibleUserID: pick.UserID})
7. eventBus.Publish(EventLeadResponsibleAssigned{
       LeadID, ResponsibleUserID,
       Reason: "created",
       Outcome: pick.Outcome,       // "picked" ou "fallback_owner"
       Algorithm: pick.Algorithm,
   })  // TenantID vai no envelope Event.TenantID, não no payload
```

### 5.1 Mudança no payload de `EventLeadCreated`

O evento já existe e é publicado em `internal/funnel/application/create_lead.go`. Adicionamos `responsible_user_id` ao payload. Consumidores atuais (ninguém depende ainda, conforme exploração) não quebram.

### 5.2 Novo evento `EventLeadResponsibleAssigned`

Adicionado em `internal/shared/events/event.go` como novo tipo. O `tenant_id` viaja no envelope `events.Event.TenantID` (comum a todos os eventos do sistema), não duplicado no payload. Payload tipado `ResponsibleAssignedPayload`:

```
{
  lead_id, responsible_user_id,
  reason: "created",          // futuro: "reassigned", "auto_move" etc.
  outcome: "picked" | "fallback_owner",
  algorithm: "round_robin" | "least_load" | "random" | ""  // vazio em fallback
}
```

## 6. Notification wiring

- `notification.Module` assina `EventLeadResponsibleAssigned` no bootstrap
- Handler converte o evento em `NotifyService.Notify(userID, NotificationTypeLeadAssigned, payload)`
- Canal in-app (SSE + polling) já existe e funciona
- Canal WhatsApp **opcional** (por `NotificationPreference` do usuário) — lógica de preferência já existe; basta consultar antes de despachar pelo canal extra
- Sem telas novas de notificação nesta entrega (virão na próxima — foco 2 do plano)

## 7. Observabilidade

Métricas Prometheus (namespace `crm_lead_responsible_picker`):

- `_total{algorithm, outcome}` — counter
  - `algorithm` ∈ `round_robin`, `least_load`, `random`, `none` (none = fallback direto)
  - `outcome` ∈ `picked`, `fallback_owner`, `error`
- `_duration_seconds{algorithm}` — histogram (buckets padrão do projeto)

Log estruturado (nível INFO):

- Campos: `tenant_id`, `funnel_id`, `column_id`, `group_id`, `algorithm`, `picked_user_id`, `outcome`, `duration_ms`
- Em `outcome=fallback_owner`: inclui `fallback_reason` (`no_group`, `no_active_config`, `no_active_members`, `multiple_active_groups`, `picker_error`)

Tracing OpenTelemetry:

- Span `load_balance.pick` dentro do span de `CreateLead`
- Atributos: os mesmos campos do log

Dashboards: não criamos dashboard novo nesta entrega — as métricas ficam disponíveis para o dashboard consolidado do funil.

## 8. Segurança / OWASP

- `PickForFunnel` só aceita IDs já validados pelo caller; não executa SQL com input cru
- `LeadLoadCounter.CountActiveByUsers` é query parametrizada (GORM/placeholders)
- Isolamento de tenant em **todas** as queries do picker (`WHERE tenant_id = ?`)
- Evento `EventLeadResponsibleAssigned` carrega `tenant_id` no payload para o handler do notification garantir o isolamento

## 9. Testes

Unitários (TDD):

- `LoadBalancePicker`: um caso por algoritmo + um por cada caminho de fallback (5 fallbacks × 3 algoritmos = consolidar em tabela), incluindo erro de config com múltiplos grupos ativos
- `LeadLoadCounter`: teste de integração com testcontainers (MySQL real)
- `CreateLeadUseCase`: já tem testes; adicionar casos cobrindo picker + fallback + evento publicado corretamente
- `NotificationModule`: teste cobrindo recebimento de `EventLeadResponsibleAssigned` e chamada correta ao `NotifyService`

OWASP:

- `AssignLeadUseCase` já tem testes 401/403/tenant isolation; confirmar que o fluxo automático não quebra esses testes existentes
- Confirmar que um usuário de outro tenant nunca é elegível como responsável (test case dedicado)

Cobertura-alvo: ≥ 80% no `LoadBalancePicker` e manter ≥ 80% no `CreateLeadUseCase`.

## 10. Arquivos .http

Atualizar `rest/team.http` e/ou `rest/funnel.http`:

- Nota destacando que a criação de lead agora popula `responsible_user_id` automaticamente
- Exemplo de `GET /funnel/leads/{id}` mostrando o campo preenchido
- Cenário manual: criar funil sem LB ativo → criar lead → confirmar responsável = owner

## 11. Out of scope (ficam para próximas entregas)

- Telas HTMX de notificação (badge, toast, painel) — próxima iteração dentro da mesma feature
- Configurável admin-vs-tenant para criar automações (F09 Step 7 item aberto)
- Especialista vinculado a produto (F10 Step 3 item aberto)
- Observabilidade dos demais endpoints novos do F08/F09 (fica como task separada na status.md)

## 12. Critérios de aceite desta entrega

- [ ] Todo lead criado (HTTP manual, WhatsApp, IA) nasce com `ResponsibleUserID` preenchido
- [ ] Cascata de fallback funciona: sem grupo/config/membros → owner do tenant
- [ ] Regra de unicidade (3.2) validada no `SetByGroup` com teste 409
- [ ] `least_load` usa contagem de leads ativos no tenant
- [ ] Evento `EventLeadResponsibleAssigned` publicado em toda criação
- [ ] `notification.Module` consome o evento e chama `NotifyService`
- [ ] Métricas, logs e traces emitidos conforme seção 7
- [ ] Cobertura ≥ 80% em `LoadBalancePicker` e `CreateLeadUseCase`
- [ ] Testes OWASP existentes continuam passando
- [ ] `rest/*.http` atualizados
- [ ] `docs/artefatos/F08-F09-.../status.md` marcado com Step 4 e 4.1 concluídos
