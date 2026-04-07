# F08 + F09 — Usuários, Permissões e Automações

Spec unificado para as features F08 (Usuários e Permissões) e F09 (Automações), que compartilham infraestrutura de permissões, notificações e eventos.

## Decisões de Design

| Decisão | Escolha |
|---------|---------|
| Modelo de permissões | Híbrido (módulo:ação) com permissão de grupo + individual |
| Resolução de conflitos | Union (mais permissivo) — qualquer fonte que conceda, permite |
| Notificação in-app | SSE + fallback polling (30s) |
| Notificação WhatsApp | Opt-in por usuário |
| Load balance | Apenas na criação do lead |
| Algoritmos load balance | round_robin, least_load, random |
| Motor de automações | Híbrido — sync para leves, async para pesadas |
| Job de expiração | Ticker em goroutine (5min) |
| Convite de usuários | Link com token (sem email service) |
| Organização de módulos | 3 novos (permission, notification, automation) + extensões em auth e funnel |

## Arquitetura

### Novos Módulos

```
internal/
  permission/
    domain/         # PermissionGroup, Permission, UserGroup, ViewProfile, GroupFunnel
    application/    # CRUD grupos, resolver permissões, CRUD perfis
    infrastructure/ # GORM repositories
    interfaces/http/# handlers + middleware de autorização

  notification/
    domain/         # Notification, NotificationPreference
    application/    # Notify, listar, marcar lida, SSE stream, preferências
    infrastructure/ # GORM repositories, SSE handler
    interfaces/http/# handlers

  automation/
    domain/         # Automation, ExecutionLog, RateLimitCounter
    application/    # CRUD automações, AutomationEngine, ExpirationTicker, executors por tipo
    infrastructure/ # GORM repositories
    interfaces/http/# handlers
```

### Módulos Estendidos

- **auth**: InviteToken, LoadBalanceConfig, UserTenant.IsOwner, UserTenant.WhatsAppID
- **funnel**: Lead.ResponsibleUserID, LeadNote.CreatedBy/Type, eventos lead-created/lead-moved
- **shared**: EventBus promovido de whatsapp para shared/events (interface + implementação in-memory)

### Fluxo de Eventos

```
Lead criado     → EventBus → NotificationService (notifica responsável)
                           → AutomationEngine (executa automações da coluna)

Lead movido     → EventBus → NotificationService (notifica responsável)
                           → AutomationEngine (executa automações da coluna destino)

Automação msg   → async    → WhatsApp.SendMessage
Automação leve  → sync     → executa inline (mover, anotar, trocar especialista)

ExpirationTicker→ cada 5min→ verifica leads expirados → executa exclusão/movimentação

Qualquer request→ PermissionMiddleware → resolve permissões (union) → allow/deny
```

---

## Módulo Permission

### Entidades

#### PermissionGroup

| Campo | Tipo | Restrições |
|-------|------|------------|
| ID | string (UUID) | PK |
| TenantID | string (UUID) | FK tenants, NOT NULL |
| Name | string | max 100, NOT NULL |
| Description | string | max 500 |
| CreatedAt | time.Time | |
| UpdatedAt | time.Time | |

#### UserGroup

| Campo | Tipo | Restrições |
|-------|------|------------|
| ID | string (UUID) | PK |
| UserID | string (UUID) | FK users, NOT NULL |
| GroupID | string (UUID) | FK permission_groups, NOT NULL |
| TenantID | string (UUID) | FK tenants, NOT NULL |
| CreatedAt | time.Time | |

UNIQUE: (UserID, GroupID)

#### Permission

| Campo | Tipo | Restrições |
|-------|------|------------|
| ID | string (UUID) | PK |
| TenantID | string (UUID) | FK tenants, NOT NULL |
| GroupID | *string (UUID) | FK permission_groups, NULL = individual |
| UserID | *string (UUID) | FK users, NULL = grupo |
| Resource | string | NOT NULL |
| Action | string | NOT NULL |
| CreatedAt | time.Time | |

Regra: GroupID XOR UserID (nunca ambos preenchidos, nunca ambos nulos).

UNIQUE: (TenantID, GroupID, Resource, Action) — para permissões de grupo
UNIQUE: (TenantID, UserID, Resource, Action) — para permissões individuais

Nota: MySQL permite múltiplos NULLs em UNIQUE constraints, então os constraints acima funcionam corretamente — permissões de grupo (UserID=NULL) não colidem entre si no segundo index, e vice-versa. A regra XOR (GroupID ou UserID, nunca ambos) é enforçada via CHECK constraint na migration.

#### ViewProfile

| Campo | Tipo | Restrições |
|-------|------|------------|
| ID | string (UUID) | PK |
| GroupID | string (UUID) | FK permission_groups, NOT NULL |
| FunnelID | string (UUID) | FK funnels, NOT NULL |
| VisibleColumns | []string | JSON array de column IDs |
| CreatedAt | time.Time | |
| UpdatedAt | time.Time | |

UNIQUE: (GroupID, FunnelID)

#### GroupFunnel

| Campo | Tipo | Restrições |
|-------|------|------------|
| ID | string (UUID) | PK |
| GroupID | string (UUID) | FK permission_groups, NOT NULL |
| FunnelID | string (UUID) | FK funnels, NOT NULL |
| ColumnIDs | []string | JSON array, vazio = todo funil |
| CreatedAt | time.Time | |

UNIQUE: (GroupID, FunnelID)

### Permissões Disponíveis

| Resource | Action | Descrição |
|----------|--------|-----------|
| funnels | manage | Criar/editar/excluir funis e colunas |
| funnels | customize | Personalizar visualização (perfis) |
| leads | view | Ver leads no kanban |
| leads | manage | Criar/mover/editar leads |
| automations | manage | Criar/editar/ativar automações |
| users | manage | Convidar/remover/editar usuários |
| groups | manage | Criar/editar grupos e permissões |
| products | manage | CRUD de produtos |
| specialists | manage | Configurar especialistas e IA |
| whatsapp | view | Ver conversas do WhatsApp |
| whatsapp | send | Enviar mensagens pelo WhatsApp |
| settings | manage | Configurações gerais do tenant |

### Resolução de Permissões (Union)

```
func HasPermission(userID, tenantID, resource, action) bool:
  1. Se user é owner do tenant (UserTenant.IsOwner) → true
  2. Se user é admin global (User.Role == "admin") → true
  3. Buscar permissões individuais do user para (tenantID, resource, action)
  4. Se encontrou → true
  5. Buscar grupos do user no tenant (UserGroup)
  6. Buscar permissões dos grupos para (resource, action)
  7. Se qualquer grupo concede → true
  8. Retorna false
```

### Middleware de Autorização

Novo middleware `RequirePermission(resource, action)` que:
1. Extrai userID e tenantID do contexto (já feito pelo auth middleware)
2. Chama `HasPermission(userID, tenantID, resource, action)`
3. Se false → 403 Forbidden
4. Se true → next handler

Aplicado por rota. Exemplo: `router.POST("/automations", RequirePermission("automations", "manage"), handler.Create)`

### Endpoints

| Método | Rota | Permissão | Descrição |
|--------|------|-----------|-----------|
| GET | /tenant/groups | groups:manage | Listar grupos |
| POST | /tenant/groups | groups:manage | Criar grupo |
| GET | /tenant/groups/:id | groups:manage | Detalhe do grupo |
| PUT | /tenant/groups/:id | groups:manage | Editar grupo |
| DELETE | /tenant/groups/:id | groups:manage | Excluir grupo |
| POST | /tenant/groups/:id/members | groups:manage | Adicionar membro |
| DELETE | /tenant/groups/:id/members/:uid | groups:manage | Remover membro |
| GET | /tenant/groups/:id/permissions | groups:manage | Listar permissões do grupo |
| PUT | /tenant/groups/:id/permissions | groups:manage | Atualizar permissões do grupo |
| GET | /tenant/users/:id/permissions | users:manage | Listar permissões individuais |
| PUT | /tenant/users/:id/permissions | users:manage | Atualizar permissões individuais |
| GET | /tenant/groups/:id/view-profiles | funnels:customize | Listar perfis de visualização |
| PUT | /tenant/groups/:id/view-profiles/:fid | funnels:customize | Configurar perfil |
| GET | /tenant/groups/:id/funnels | groups:manage | Listar associações grupo-funil |
| PUT | /tenant/groups/:id/funnels | groups:manage | Configurar associações |

---

## Módulo Notification

### Entidades

#### Notification

| Campo | Tipo | Restrições |
|-------|------|------------|
| ID | string (UUID) | PK |
| TenantID | string (UUID) | FK tenants, NOT NULL |
| UserID | string (UUID) | FK users, NOT NULL (destinatário) |
| Type | string | enum, NOT NULL |
| Title | string | max 200, NOT NULL |
| Body | string | max 1000 |
| Metadata | map[string]string | JSON |
| Read | bool | default false |
| CreatedAt | time.Time | |

#### NotificationPreference

| Campo | Tipo | Restrições |
|-------|------|------------|
| ID | string (UUID) | PK |
| UserID | string (UUID) | FK users, NOT NULL |
| TenantID | string (UUID) | FK tenants, NOT NULL |
| Channel | string | "in_app" ou "whatsapp" |
| Enabled | bool | |
| CreatedAt | time.Time | |
| UpdatedAt | time.Time | |

UNIQUE: (UserID, TenantID, Channel)

Defaults: in_app = true, whatsapp = false (opt-in).

### Tipos de Notificação

| Type | Origem | Descrição |
|------|--------|-----------|
| lead_assigned | Load balance (F08) | Lead criado e atribuído a você |
| lead_moved | Kanban / Automação | Lead movido de coluna |
| lead_handoff | AI engine (F16) | IA fez handoff para humano |
| lead_qualified | AI scoring (F16) | Lead qualificado/desqualificado pela IA |
| rate_limit_reached | Rate limit (F09) | Limite de mensagens IA atingido |
| automation_error | Automation engine (F09) | Automação falhou na execução |

### Canais de Entrega

**In-App (SSE + Polling)**:
- SSE: `GET /notifications/stream` — conexão persistente, envia evento quando nova notificação. Frontend atualiza badge e mostra toast via HTMX.
- Polling: `GET /notifications?unread=true` — fallback a cada 30s quando SSE desconecta. Frontend reconecta SSE automaticamente.
- Usa o EventBus compartilhado (tipo `notification`) para entregar ao SSE handler.

**WhatsApp (Opcional)**:
- Opt-in nas preferências do usuário.
- Requer `UserTenant.WhatsAppID` preenchido.
- Envia via WhatsApp provider existente (async, goroutine).
- Formato: `"[CRM] {Title}\n{Body}\nVer: {deep-link}"`

### Fluxo de Disparo

```
NotificationService.Notify(userID, tenantID, type, title, body, metadata):
  1. Persistir Notification no banco
  2. Consultar NotificationPreference do user
  3. Se in_app habilitado → publicar evento "notification" no EventBus
     → SSE handler entrega ao frontend
  4. Se whatsapp habilitado e WhatsAppID preenchido → goroutine
     → enviar via WhatsApp provider
```

### Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | /notifications/stream | SSE stream |
| GET | /notifications | Listar notificações (polling + página) |
| GET | /notifications/unread-count | Badge counter |
| POST | /notifications/:id/read | Marcar como lida |
| POST | /notifications/read-all | Marcar todas como lidas |
| GET | /notifications/preferences | Preferências do user |
| PUT | /notifications/preferences | Atualizar preferências |

---

## Módulo Automation

### Entidades

#### Automation

| Campo | Tipo | Restrições |
|-------|------|------------|
| ID | string (UUID) | PK |
| TenantID | string (UUID) | FK tenants, NOT NULL |
| FunnelID | string (UUID) | FK funnels, NOT NULL |
| ColumnID | *string (UUID) | FK funnel_columns, NULL para rate_limit |
| Type | string | enum (7 tipos), NOT NULL |
| Config | map[string]any | JSON, varia por tipo |
| Active | bool | default true |
| Priority | int | ordem de execução na coluna |
| CreatedAt | time.Time | |
| UpdatedAt | time.Time | |

#### ExecutionLog

| Campo | Tipo | Restrições |
|-------|------|------------|
| ID | string (UUID) | PK |
| AutomationID | string (UUID) | FK automations, NOT NULL |
| LeadID | string (UUID) | FK leads, NOT NULL |
| TenantID | string (UUID) | FK tenants, NOT NULL |
| Status | string | "success" ou "error" |
| ErrorMessage | *string | |
| ExecutedAt | time.Time | |

#### RateLimitCounter

| Campo | Tipo | Restrições |
|-------|------|------------|
| ID | string (UUID) | PK |
| TenantID | string (UUID) | FK tenants, NOT NULL |
| SpecialistID | *string (UUID) | FK specialists, NULL = tenant-wide |
| PeriodStart | time.Time | início do período (dia) |
| MessageCount | int | default 0 |
| CreatedAt | time.Time | |

UNIQUE: (TenantID, SpecialistID, PeriodStart)

### 7 Tipos de Automação

#### 1. expiration — Exclusão por tempo

- **Config**: `{ "duration_hours": 72, "action": "archive" | "delete" }`
- **Trigger**: ExpirationTicker (goroutine, a cada 5min)
- **Ação**: verifica `Lead.ColumnEnteredAt + duration_hours < now()`. Se expirado: `archive` move para coluna lost do funil, `delete` faz soft-delete (Lead.Status = "deleted", não aparece no kanban, mantém histórico).
- **Modo**: Ticker

#### 2. move_funnel — Mover para outro funil

- **Config**: `{ "target_funnel_id": "uuid", "target_column_id": "uuid" }`
- **Trigger**: lead entra na coluna configurada
- **Ação**: move lead para funil/coluna destino
- **Modo**: Sync

#### 3. auto_message — Mensagem automática

- **Config**: `{ "template": "Olá {{.ContactName}}, ...", "variables": ["ContactName", "ProductName"] }`
- **Trigger**: lead entra na coluna configurada
- **Ação**: renderiza template com variáveis do lead/contato, envia via WhatsApp provider
- **Modo**: Async (goroutine)

#### 4. auto_note — Anotação automática

- **Config**: `{ "template": "Lead movido para {{.ColumnName}} em {{.Date}}" }`
- **Trigger**: lead entra na coluna configurada
- **Ação**: cria LeadNote com CreatedBy="system", Type="automation"
- **Modo**: Sync

#### 5. switch_specialist — Trocar especialista

- **Config**: `{ "specialist_id": "uuid" }`
- **Trigger**: lead entra na coluna configurada
- **Ação**: atualiza ConversationState.SpecialistID. IA passa a usar novo especialista nas próximas mensagens.
- **Modo**: Sync

#### 6. rate_limit — Limite de mensagens IA

- **Config**: `{ "max_messages": 50, "period": "daily", "specialist_id": "uuid" | null }`
- **Trigger**: interceptor no AI module, antes de cada resposta IA
- **Ação**: incrementa RateLimitCounter. Se `MessageCount >= max_messages`: bloqueia resposta IA, loga WARN, notifica responsável do lead (rate_limit_reached) e owner do tenant.
- **Modo**: Async (notificação)
- **Nota**: esta automação não é vinculada a uma coluna — é configurada a nível de tenant/especialista. ColumnID fica NULL.

#### 7. detect_product — Detectar produto e redirecionar

- **Config**: `{ "switch_specialist": true }`
- **Trigger**: lead entra na coluna configurada e lead tem ProductID detectado
- **Ação**: consulta FunnelProductRouter para encontrar funil do produto. Move lead para funil/coluna do produto. Se `switch_specialist: true`, busca SpecialistProduct e atualiza ConversationState.SpecialistID.
- **Modo**: Sync

### AutomationEngine

```
OnLeadEvent(tenantID, leadID, columnID, eventType):
  1. Buscar automações ativas para (tenantID, columnID) ordenadas por Priority
  2. Para cada automação:
     a. Instanciar executor do tipo
     b. Se SYNC: executar inline
        - Sucesso → logar ExecutionLog(status=success)
        - Erro → logar ExecutionLog(status=error) + notificar (automation_error)
     c. Se ASYNC: disparar goroutine
        - Mesmo fluxo de log ao completar
     d. Erro em uma automação NÃO bloqueia as demais
```

### ExpirationTicker

```
Goroutine iniciada no boot da aplicação:
  ticker := time.NewTicker(5 * time.Minute)
  for range ticker.C:
    1. Buscar todas automações tipo "expiration" ativas
    2. Para cada automação:
       a. Buscar leads na coluna onde ColumnEnteredAt + duration_hours < now()
       b. Para cada lead expirado: executar ação (archive/delete)
       c. Logar ExecutionLog
    3. Contexto com timeout para evitar execução infinita
```

### Endpoints

| Método | Rota | Permissão | Descrição |
|--------|------|-----------|-----------|
| GET | /funnels/:id/automations | automations:manage | Listar por funil |
| POST | /funnels/:id/automations | automations:manage | Criar automação |
| GET | /automations/:id | automations:manage | Detalhe |
| PUT | /automations/:id | automations:manage | Editar |
| DELETE | /automations/:id | automations:manage | Excluir |
| POST | /automations/:id/toggle | automations:manage | Ativar/desativar |
| GET | /automations/:id/logs | automations:manage | Logs de execução |

---

## Extensões — Módulo Auth

### InviteToken

| Campo | Tipo | Restrições |
|-------|------|------------|
| ID | string (UUID) | PK |
| TenantID | string (UUID) | FK tenants, NOT NULL |
| Token | string | crypto/rand 32 bytes hex, UNIQUE |
| CreatedBy | string (UUID) | FK users, NOT NULL |
| GroupIDs | []string | JSON array de group IDs |
| ExpiresAt | time.Time | NOT NULL |
| UsedAt | *time.Time | |
| UsedBy | *string (UUID) | FK users |
| CreatedAt | time.Time | |

### UserTenant (campos novos)

| Campo | Tipo | Restrições |
|-------|------|------------|
| IsOwner | bool | default false |
| WhatsAppID | *string | JID do user para notificações |

O primeiro usuário do tenant é automaticamente owner. Owner pode transferir ownership.

### LoadBalanceConfig

| Campo | Tipo | Restrições |
|-------|------|------------|
| ID | string (UUID) | PK |
| TenantID | string (UUID) | FK tenants, NOT NULL |
| GroupID | string (UUID) | FK permission_groups, NOT NULL |
| Algorithm | string | "round_robin", "least_load", "random" |
| LastIndex | int | default 0 (para round_robin) |
| CreatedAt | time.Time | |
| UpdatedAt | time.Time | |

UNIQUE: (TenantID, GroupID)

### Fluxo de Convite

```
1. Owner/admin gera convite: POST /tenant/invites
   → cria InviteToken com GroupIDs, ExpiresAt (default 7 dias)
   → retorna link: /invite/{token}

2. Owner compartilha link (WhatsApp, email, etc.)

3. Usuário acessa GET /invite/{token}
   → valida token (existe, não expirado, não usado)
   → renderiza página de aceite (criar conta ou login)

4. Usuário aceita POST /invite/{token}/accept
   → se novo: cria User + UserTenant
   → se existente: cria UserTenant
   → atribui aos grupos de GroupIDs
   → marca token como usado (UsedAt, UsedBy)
```

### Fluxo de Load Balance

```
Lead é criado no funil:
  1. Identificar coluna de entrada
  2. Buscar GroupFunnel que cobre essa coluna
  3. Se não há grupo → ResponsibleUserID = NULL, fim
  4. Buscar LoadBalanceConfig do grupo
  5. Se não há config → ResponsibleUserID = NULL, fim
  6. Buscar membros do grupo (UserGroup)
  7. Aplicar algoritmo:
     - round_robin: members[LastIndex % len], incrementa LastIndex
     - least_load: membro com menos leads abertos (status=open)
     - random: rand.Intn(len(members))
  8. Atribuir Lead.ResponsibleUserID = membro selecionado
  9. Notificar responsável (lead_assigned)
```

### Endpoints (Auth)

| Método | Rota | Permissão | Descrição |
|--------|------|-----------|-----------|
| GET | /tenant/users | users:manage | Listar usuários do tenant |
| DELETE | /tenant/users/:id | users:manage | Remover do tenant |
| PUT | /tenant/users/:id/whatsapp | (próprio user ou users:manage) | Configurar WhatsAppID |
| POST | /tenant/invites | users:manage | Gerar convite |
| GET | /tenant/invites | users:manage | Listar convites |
| DELETE | /tenant/invites/:id | users:manage | Revogar convite |
| GET | /invite/:token | (público) | Página de aceite |
| POST | /invite/:token/accept | (público) | Aceitar convite |
| GET | /tenant/load-balance | groups:manage | Listar configs de load balance |
| PUT | /tenant/load-balance/:group_id | groups:manage | Configurar load balance |

---

## Extensões — Módulo Funnel

### Lead (campo novo)

| Campo | Tipo | Restrições |
|-------|------|------------|
| ResponsibleUserID | *string (UUID) | FK users, NULL |

### LeadNote (campos novos)

| Campo | Tipo | Restrições |
|-------|------|------------|
| CreatedBy | string | "system", "ai", ou UUID do usuário |
| Type | string | "manual", "automation", "ai" |

### Novos Eventos no EventBus

**lead-created**:
```json
{
  "type": "lead-created",
  "tenant_id": "uuid",
  "payload": {
    "lead_id": "uuid",
    "funnel_id": "uuid",
    "column_id": "uuid",
    "contact_id": "uuid"
  }
}
```

**lead-moved**:
```json
{
  "type": "lead-moved",
  "tenant_id": "uuid",
  "payload": {
    "lead_id": "uuid",
    "funnel_id": "uuid",
    "from_column_id": "uuid",
    "to_column_id": "uuid",
    "moved_by": "uuid | system | ai"
  }
}
```

### Kanban com ViewProfile

O handler `GetKanban` existente é estendido:
1. Identificar grupos do usuário logado
2. Buscar ViewProfiles de todos os grupos do user para o funil
3. Fazer union das VisibleColumns de todos os perfis (consistente com modelo union de permissões)
4. Filtrar colunas: mostrar apenas as que aparecem no union de VisibleColumns
5. Se usuário não tem ViewProfile (owner/admin ou sem grupo) → mostrar todas as colunas

### Endpoint Novo

| Método | Rota | Permissão | Descrição |
|--------|------|-----------|-----------|
| PUT | /leads/:id/assign | leads:manage | Atribuir responsável manualmente |

---

## Promoção do EventBus para Shared

O EventBus atualmente vive em `internal/whatsapp/domain/events.go`. Será promovido para `internal/shared/events/`:

```
internal/shared/events/
  event.go          # interface EventBus, tipos Event, EventType
  memory_bus.go     # implementação in-memory (movida do whatsapp)
```

Tipos de evento expandidos:
- `new-message` (whatsapp — existente)
- `conversation-update` (whatsapp — existente)
- `lead-created` (funnel — novo)
- `lead-moved` (funnel — novo)
- `notification` (notification — novo)

O módulo whatsapp passa a importar de shared/events em vez de usar sua própria implementação. Todos os novos módulos usam a mesma interface.

---

## Migrations Necessárias

1. `000035_create_permission_groups.up.sql` — tabela permission_groups
2. `000036_create_user_groups.up.sql` — tabela user_groups
3. `000037_create_permissions.up.sql` — tabela permissions (com check constraint GroupID XOR UserID)
4. `000038_create_view_profiles.up.sql` — tabela view_profiles
5. `000039_create_group_funnels.up.sql` — tabela group_funnels
6. `000040_create_notifications.up.sql` — tabela notifications
7. `000041_create_notification_preferences.up.sql` — tabela notification_preferences
8. `000042_create_automations.up.sql` — tabela automations
9. `000043_create_execution_logs.up.sql` — tabela execution_logs
10. `000044_create_rate_limit_counters.up.sql` — tabela rate_limit_counters
11. `000045_create_invite_tokens.up.sql` — tabela invite_tokens
12. `000046_add_is_owner_to_user_tenants.up.sql` — coluna is_owner no user_tenants
13. `000047_add_whatsapp_id_to_user_tenants.up.sql` — coluna whatsapp_id no user_tenants
14. `000048_create_load_balance_configs.up.sql` — tabela load_balance_configs
15. `000049_add_responsible_user_id_to_leads.up.sql` — coluna responsible_user_id no leads
16. `000050_add_created_by_type_to_lead_notes.up.sql` — colunas created_by e type no lead_notes

---

## Testes

### Testes OWASP (por módulo)

Cada endpoint deve ter testes de:
- **401**: request sem token
- **403**: request com token mas sem permissão (usando RequirePermission middleware)
- **Isolamento de tenant**: user do tenant A não acessa dados do tenant B
- **Anti-injection**: SQL injection nos campos de texto (name, description, template)

### Cobertura

Meta: >= 80% em cada módulo (permission, notification, automation).

### Testes Unitários

- Resolução de permissões (union de grupo + individual)
- Cada tipo de automação (7 executors)
- Load balance (3 algoritmos)
- Template rendering (variáveis em auto_message e auto_note)
- ExpirationTicker (leads expirados)
- Rate limit counter (incremento, reset de período)

### Testes de Integração

- Fluxo completo: lead criado → load balance atribui → notificação enviada
- Fluxo completo: lead movido → automação executa → log gravado
- Convite: gerar link → aceitar → user associado ao tenant com grupos
- SSE: conectar stream → criar notificação → evento recebido
- ViewProfile: configurar perfil → kanban filtra colunas
