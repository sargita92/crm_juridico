# F08 Step 6 — Telas de Equipe (HTMX)

## Contexto

O backend do F08 está quase completo: grupos de permissão, perfis de visualização, associação grupo↔funil, convites, gestão de usuários e notificações já estão implementados e com cobertura >= 80%. Falta (a) completar o backend de **load balance** (há domínio e repositório, faltam use case e endpoints HTTP) e (b) construir as telas HTMX para que o usuário gerencie tudo pela interface.

Este design cobre as 4 telas pendentes do F08 Step 6:

1. Gestão de usuários do tenant
2. Gestão de grupos de permissão
3. Perfis de visualização
4. Load balance (inclui conclusão do backend)

## Decisões de design

| Decisão | Escolha |
|---------|---------|
| Navegação | Item único **"Equipe"** no sidebar do tenant |
| Estrutura | 2 abas: **Usuários** / **Grupos** |
| Detail do grupo | Página própria com seções verticais (membros, permissões, funis, perfis de visualização, load balance) |
| Permissões individuais | Modal acionado na linha do usuário |
| Load balance | Seção dentro do detail do grupo |
| Perfis de visualização | Seção dentro do detail do grupo (por funil atribuído) |
| Notificações | Fora de escopo deste pacote |

## Navegação

Novo item **"Equipe"** no sidebar (`partials/tenant_sidebar.html`), posicionado entre "Leads" e "Produtos". Visível quando o usuário tem permissão `users:read` OU `groups:manage`. Ícone: grupo de pessoas.

Rota principal: `GET /tenant/team` → redirect para `/tenant/team/users`.

A aba ativa é derivada da URL (`/tenant/team/users` ou `/tenant/team/groups`). Cada aba só renderiza se o usuário tem a permissão correspondente; se só tem uma, a outra fica oculta.

## Página da aba "Usuários"

### Cabeçalho

- Título "Usuários do Tenant"
- Contador ("N usuários ativos")
- Botão primário **"+ Convidar Usuário"**

### Tabela principal (`#users-table`)

| Coluna | Conteúdo |
|--------|----------|
| Nome | Nome + email abaixo (cinza) |
| Papel | Badge "Owner" (dourado) ou "Membro" |
| Grupos | Badges dos grupos (até 3 visíveis + "+N" se mais) |
| WhatsApp ID | Número formatado ou "—" + botão ✏️ inline |
| Ações | 🔑 Permissões · 🗑️ Remover (oculto para owner) |

### Interações HTMX

- **🔑 Permissões** — `hx-get="/tenant/team/users/:id/permissions-modal"` → modal com:
  - Permissões herdadas dos grupos (badges cinzas, read-only)
  - Checklist de permissões individuais extras (editável)
  - Submit via `hx-put="/tenant/users/:id/permissions"` → fecha modal, dispara `refreshUsers`
- **🗑️ Remover** — `hx-delete="/tenant/users/:id"` com `hx-confirm="Remover este usuário do tenant?"` → 403 retornado para owner
- **✏️ WhatsApp** — `hx-get="/tenant/team/users/:id/whatsapp-modal"` → modal com input → `hx-put="/tenant/users/:id/whatsapp"`

### Seção "Convites Pendentes" (abaixo da tabela, colapsável)

Tabela simples com: link do convite (copiável) · grupos atribuídos · expira em · ações (🔗 copiar · 🗑️ revogar).

### Modal "Convidar Usuário"

Campos:
- Select multi (checkbox-list) de grupos de permissão
- Input de dias até expirar (default: 7)

Submit: `hx-post="/tenant/invites"` → retorna o link gerado em destaque com botão "Copiar". Após copiar, botão "Novo convite" ou "Fechar".

### Empty states

- Sem usuários: "Convide seu primeiro membro" + botão
- Sem convites pendentes: "Nenhum convite pendente"

## Página da aba "Grupos"

### Cabeçalho

- Título "Grupos de Permissão"
- Botão primário **"+ Novo Grupo"**

### Tabela (`#groups-table`)

| Coluna | Conteúdo |
|--------|----------|
| Nome | Nome + descrição (cinza) |
| Membros | Contador + avatar stack (até 3) |
| Funis | Badges dos funis atribuídos (ou "Todos") |
| Balanceamento | Ícone do algoritmo (🔁 round-robin / 📉 least-load / 🎲 random) + nome |
| Ações | 👁️ Abrir · 🗑️ Excluir |

Clique no nome ou em 👁️ → navega para `/tenant/team/groups/:id`.

### Modal "Novo Grupo"

Campos: nome + descrição. Após criar, redireciona para o detail recém-criado.

## Detail do grupo (`/tenant/team/groups/:id`)

Página própria mantendo o sidebar "Equipe" ativo. Layout similar ao detail do especialista.

### Header

- Breadcrumb: Equipe › Grupos › *Nome do grupo*
- Nome editável inline
- Descrição editável inline
- Botão "🗑️ Excluir grupo" no canto superior direito

### Seções verticais (cards colapsáveis)

Cada seção é um fragment carregado independentemente. Save de cada seção é atômico — não há botão "Salvar tudo" no topo. Feedback visual: inline "✓ salvo" após resposta 2xx.

#### 👥 Membros

Lista de usuários do grupo · botão "+ Adicionar membro" (modal com search) · remover inline.

Endpoints: `GET/POST/DELETE /tenant/groups/:id/members`.

#### 🔐 Permissões

Matriz de checkboxes (resource × action) agrupada por categoria.

- **Categorias (linhas):** Leads, Usuários, Grupos, Funis, Automações, Produtos, Especialistas, Convites, Configurações
- **Actions (colunas):** read, create, update, delete, manage
- Células com `—` quando action não faz sentido para o resource
- Submit via `hx-put="/tenant/groups/:id/permissions"` retorna o fragment atualizado

#### 🎯 Funis atribuídos

Lista de funis disponíveis com checkbox · opção "Todos os funis" como toggle.

Endpoint: `GET/PUT /tenant/groups/:id/funnels`.

#### 👁️ Perfis de visualização

Para cada funil atribuído, lista as colunas do funil com checkbox "coluna visível". "Todas as colunas" é o default.

Endpoint: `GET /tenant/groups/:id/view-profiles` · `PUT /tenant/groups/:id/view-profiles/:fid`.

#### ⚖️ Load Balance

- Radio group com os 3 algoritmos:
  - `round_robin` — "Distribui em ordem circular"
  - `least_load` — "Atribui para quem tem menos leads"
  - `random` — "Sorteia aleatoriamente"
- Toggle "Balanceamento ativo"
- Texto informativo: "Quando ativo, novos leads serão atribuídos automaticamente aos membros deste grupo."
- Save via `hx-put="/tenant/groups/:id/load-balance"`

Endpoint novo (ver seção "Backend de Load Balance").

## Backend de Load Balance

O domínio (`internal/auth/domain/load_balance.go`) e o repositório (`internal/auth/infrastructure/gorm_load_balance_repo.go`) já existem. Faltam:

### Ajuste no domínio

Adicionar campo `Active bool` em `LoadBalanceConfig` para permitir desligar o balanceamento sem excluir a config. Default `true` em configs existentes.

Migration nova: `000051_add_active_to_load_balance_configs.{up,down}.sql` — adiciona coluna `active BOOLEAN NOT NULL DEFAULT TRUE`. (050 é a última migration existente.)

### Use case `ManageLoadBalanceUseCase`

Arquivo: `internal/auth/application/manage_load_balance.go`.

```go
type ManageLoadBalanceUseCase struct {
    repo      domain.LoadBalanceConfigRepository
    groupRepo permissiondomain.GroupRepository
}

func (uc *ManageLoadBalanceUseCase) GetByGroup(ctx context.Context, tenantID, groupID string) (*domain.LoadBalanceConfig, error)

type SetLoadBalanceInput struct {
    TenantID  string
    GroupID   string
    Algorithm domain.LoadBalanceAlgorithm
    Active    bool
}

func (uc *ManageLoadBalanceUseCase) SetByGroup(ctx context.Context, input SetLoadBalanceInput) (*domain.LoadBalanceConfig, error)
```

Validações:
- Grupo deve pertencer ao tenant (via `groupRepo`)
- Algoritmo deve ser um dos 3 valores permitidos
- Tenant/group IDs não podem estar vazios

### Endpoints HTTP

Ficam no módulo `permission` para manter coesão das rotas `/tenant/groups/:id/*`. O módulo `permission` importa o `ManageLoadBalanceUseCase` via interface no construtor.

| Método | Rota | Auth | Descrição |
|--------|------|------|-----------|
| GET | `/tenant/groups/:id/load-balance` | `requirePerm("groups", "manage")` | Retorna config atual ou 404 |
| PUT | `/tenant/groups/:id/load-balance` | `requirePerm("groups", "manage")` | Cria ou atualiza config |

### Testes

- Use case: todos os 3 algoritmos válidos, algoritmo inválido, grupo inexistente, tenant mismatch
- Handler: 200 (get/put), 404 (grupo não existe), 403 (sem permissão), 400 (algoritmo inválido)
- OWASP: 401 sem token, 403 tenant diferente
- Cobertura >= 80%

## Estrutura de arquivos

```
Modify: web/templates/partials/tenant_sidebar.html         — adiciona item "Equipe"
Create: web/templates/team/                                — pasta nova
Create: web/templates/team/shell.html                      — layout compartilhado (header + tabs)
Create: web/templates/team/users_page.html                 — página aba usuários
Create: web/templates/team/users_table.html                — fragment tabela usuários
Create: web/templates/team/invites_table.html              — fragment tabela convites
Create: web/templates/team/user_permissions_modal.html     — modal permissões individuais
Create: web/templates/team/user_whatsapp_modal.html        — modal WhatsApp
Create: web/templates/team/invite_new_modal.html           — modal convidar
Create: web/templates/team/groups_page.html                — página aba grupos
Create: web/templates/team/groups_table.html               — fragment tabela grupos
Create: web/templates/team/group_new_modal.html            — modal novo grupo
Create: web/templates/team/group_detail.html               — página detail do grupo
Create: web/templates/team/group_section_members.html      — seção membros
Create: web/templates/team/group_section_permissions.html  — seção permissões
Create: web/templates/team/group_section_funnels.html      — seção funis
Create: web/templates/team/group_section_view_profiles.html — seção perfis de visualização
Create: web/templates/team/group_section_load_balance.html — seção load balance

Create: internal/auth/interfaces/http/page_handler.go       — handler HTML aba usuários
Create: internal/auth/interfaces/http/page_handler_test.go
Modify: internal/auth/module.go                             — wire PageHandler + registra rotas HTML
Create: internal/permission/interfaces/http/page_handler.go — handler HTML aba grupos + detail
Create: internal/permission/interfaces/http/page_handler_test.go
Modify: internal/permission/interfaces/http/routes.go       — registra rotas HTML
Modify: internal/permission/module.go                       — wire cross-deps

Create: internal/auth/application/manage_load_balance.go
Create: internal/auth/application/manage_load_balance_test.go
Modify: internal/auth/module.go                             — wire ManageLoadBalanceUseCase + expor
Modify: internal/permission/interfaces/http/handler.go      — endpoints load-balance (usa UC do auth)
Modify: internal/permission/interfaces/http/routes.go       — /tenant/groups/:id/load-balance
Modify: internal/auth/domain/load_balance.go                — adiciona Active bool
Create: migrations/000051_add_active_to_load_balance_configs.up.sql
Create: migrations/000051_add_active_to_load_balance_configs.down.sql

Modify: cmd/api/main.go                                     — passa novas deps cross-module
Create: rest/team.http                                      — requests HTML + load-balance JSON
```

## Rotas HTML (novas)

| Método | Rota | Módulo | Retorna |
|--------|------|--------|---------|
| GET | `/tenant/team` | `auth` | Redirect → `/tenant/team/users` |
| GET | `/tenant/team/users` | `auth` | Página completa (shell + aba usuários) |
| GET | `/tenant/team/users/table` | `auth` | Fragment — tabela de usuários |
| GET | `/tenant/team/users/:id/permissions-modal` | `auth` | Fragment — modal de permissões individuais |
| GET | `/tenant/team/users/:id/whatsapp-modal` | `auth` | Fragment — modal WhatsApp |
| GET | `/tenant/team/invites/table` | `auth` | Fragment — tabela de convites pendentes |
| GET | `/tenant/team/invites/new-modal` | `auth` | Fragment — modal "Convidar" |
| GET | `/tenant/team/groups` | `permission` | Página completa (shell + aba grupos) |
| GET | `/tenant/team/groups/table` | `permission` | Fragment — tabela de grupos |
| GET | `/tenant/team/groups/new-modal` | `permission` | Fragment — modal "Novo Grupo" |
| GET | `/tenant/team/groups/:id` | `permission` | Página completa — detail do grupo |
| GET | `/tenant/team/groups/:id/section/:name` | `permission` | Fragment — seção (members/permissions/funnels/view-profiles/load-balance) |

Endpoints de API existentes (JSON) permanecem. Forms HTML submetem via HTMX para esses novos endpoints HTML que retornam fragments atualizados.

## Cross-module dependencies

- `permission.PageHandler` depende de:
  - Funnel use cases (listar funis do tenant e suas colunas)
  - User repo (listar usuários do tenant para adicionar em grupos)
  - `ManageLoadBalanceUseCase` (do módulo `auth`)
- `auth.PageHandler` depende de:
  - Group repo (listar grupos para o select do modal de convite)
  - `ResolvePermissionUseCase` (para mostrar permissões herdadas no modal de override)

Todas as dependências injetadas como **interfaces** no construtor. Sem acoplar domínio de um módulo ao de outro — apenas pequenos contratos necessários ao rendering.

## Permissionamento

| Recurso | Permissão |
|---------|-----------|
| Item "Equipe" no sidebar | `users:read` OU `groups:manage` |
| Aba Usuários (rotas `/tenant/team/users*`) | `users:read` (modais de edição exigem `users:update`/`users:delete`) |
| Aba Grupos (rotas `/tenant/team/groups*`) | `groups:manage` |
| Convites (rotas `/tenant/team/invites*`) | `invites:create` / `invites:read` |
| Load balance (`/tenant/groups/:id/load-balance`) | `groups:manage` |
| Perfis de visualização (`/tenant/groups/:id/view-profiles/*`) | `funnels:customize` |

## Observabilidade

Consistente com padrão do projeto:
- Logs estruturados com `request_id`, `tenant_id`, `user_id` em cada handler
- Métricas Prometheus herdadas dos middlewares HTTP existentes
- Erros de use case logados com contexto antes de retornar

## Padrões reaproveitados

- `modal-overlay` + `openModal`/`closeModal` de `admin.js`
- Evento `refreshTable` (mesmo padrão da tela de automações F09)
- CSS existente de `main.css` — **sem novo CSS**
- Templates fragmentados: página completa (`*_page.html`) + fragments recarregáveis (`*_table.html`, `*_modal.html`, `section_*.html`)

## Critérios de aceite

- Item "Equipe" aparece no sidebar do tenant para quem tem permissão
- Aba Usuários lista usuários, permite convidar, editar permissões individuais, configurar WhatsApp, remover
- Aba Grupos lista grupos, permite criar, navegar para detail
- Detail do grupo permite editar membros, permissões (matriz), funis, perfis de visualização, load balance
- Load balance configurável por grupo (round-robin / least-load / random + toggle ativo)
- Owner não pode ser removido (403 explícito)
- Cobertura >= 80% em todos os handlers e use cases
- Arquivo `rest/team.http` atualizado
- Testes OWASP (401/403) nos novos endpoints

## Fora de escopo

- Notificações (badge/toast/painel) — pacote separado
- Atribuição automática via load balance na criação de leads — integração separada (item pendente em status.md)
- Reset de senha, 2FA, perfil do usuário
- Novo CSS (usar classes existentes de `main.css`)
- JavaScript customizado além do mínimo para modais (reutilizar `admin.js`)
- Reordenação de prioridade por drag-and-drop
