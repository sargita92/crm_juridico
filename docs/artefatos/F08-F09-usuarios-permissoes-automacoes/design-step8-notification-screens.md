# F09 Step 8 — Telas de Notificações (HTMX)

## Contexto

O backend de notificação está completo: domínio (6 tipos), persistência, SSE stream por usuário, contador de não lidas, mark-read/all, preferências (`in_app` e `whatsapp`). Dois tipos são efetivamente emitidos hoje: `lead_assigned` (via load balance do Step 4.1) e `automation_error` (via `NotifierAdapter` do módulo automation). Os outros quatro tipos (`lead_moved`, `lead_handoff`, `lead_qualified`, `rate_limit_reached`) estão definidos no domínio mas ainda não têm emissores.

Este step entrega a camada de visualização: sino + dropdown + toast + página dedicada, fechando o loop de UX do Step 4.1. Sem essas telas, as notificações persistidas no banco não são vistas pelo usuário.

## Decisões de design

| Decisão | Escolha | Alternativas consideradas |
|---------|---------|---------------------------|
| Placement do sino | Sino flutuante `position: fixed` canto superior direito | Top bar novo; sino no sidebar |
| Interação ao clicar | Dropdown compacto (~400×500px) + página dedicada acessível via "Ver todas" | Drawer lateral; só página dedicada |
| Push em tempo real | Toast auto-dismiss 5s + badge + SSE | Só badge; só som |
| Click-through | Deep-link apenas para `lead_assigned` → `/tenant/leads?open=<lead_id>` (abre drawer existente); outros tipos só marcam lida | Deep-link inteligente por tipo; sem deep-link |
| Preferências | **Fora de escopo** | Tela/modal; toggle global |
| Área | Só `/tenant/*` (sem `/admin/*`) | Ambas áreas |
| Mark all | Botão "Marcar todas" no rodapé do dropdown | Sem botão |
| Integração de layout | Partials compartilhados (`tenant_head.html` + `notification_bell.html`) incluídos manualmente nas páginas do tenant | Migrar todas as páginas para `layouts/tenant.html` |
| Formato SSE | SSE emite HTML fragment (toast + badge) — HTMX-first | Manter JSON e parsear no cliente |

## Navegação

- **Sem** item próprio no sidebar. Acesso a `/tenant/notifications` se dá pelo link "Ver todas" no rodapé do dropdown.
- Sino flutuante aparece em todas as páginas do tenant (usuário autenticado com `tenantMw` aplicado).
- Ícone: sininho SVG (stroke 2). Badge numérico (contagem de não lidas) sobreposto ao canto superior direito do sino; oculto se contagem é zero.

## Componentes

### 1. Sino + badge (`partials/notification_bell.html`)

```html
<div class="notification-bell-container" id="notification-bell">
    <button class="notification-bell"
            hx-get="/tenant/notifications/dropdown"
            hx-target="#notification-dropdown"
            hx-swap="innerHTML"
            onclick="toggleNotificationDropdown()">
        <svg>...</svg>
        <span id="notification-badge"
              hx-get="/tenant/notifications/badge"
              hx-trigger="load, every 30s, refreshBadge from:body"
              hx-swap="outerHTML"></span>
    </button>
    <div id="notification-dropdown" class="notification-dropdown" style="display:none"></div>
</div>

<div id="toast-container" class="toast-container"
     hx-ext="sse"
     sse-connect="/notifications/stream"
     sse-swap="notification"
     hx-swap="beforeend"></div>
```

- Badge auto-atualiza via polling leve (30s) como fallback ao SSE.
- Evento custom `refreshBadge` disparado após `mark-read` / `mark-all-read` força recarga do badge.
- SSE conecta uma única vez (uma conexão por aba). Fecha automaticamente ao descarregar a página (contract do HTMX SSE ext).

### 2. Dropdown (`partials/notification_dropdown.html`)

Conteúdo carregado sob demanda quando o sino é clicado:

- Cabeçalho: título "Notificações" + contagem de não lidas.
- Lista das 10 últimas (ordenadas desc por `created_at`), cada linha via `partial notification/item.html`.
- Não lidas destacadas: fundo sutilmente azulado + bolinha `●` à esquerda.
- Rodapé:
  - Botão "Marcar todas como lidas" — `hx-post="/notifications/read-all"` + `hx-swap="none"` + `hx-on::after-request="htmx.trigger('body', 'refreshBadge'); htmx.trigger('#notification-dropdown', 'reload')"`.
  - Link "Ver todas" — navega para `/tenant/notifications`.
- Estado vazio: ícone + "Nenhuma notificação".

Abrir/fechar controlado por `toggleNotificationDropdown()` inline (reusa padrão de `openModal`/`closeModal` de `admin.js`). Clique fora fecha.

### 3. Toast (`partials/notification_toast.html`)

Template renderizado pelo handler SSE quando um evento chega. Auto-dismiss com vanilla JS (sem nova dependência — hyperscript não é usado no projeto):

```html
<div id="toast-{{.ID}}" class="toast-notif toast-notif-{{.Type}}"
     hx-trigger="click"
     hx-post="/notifications/{{.ID}}/read"
     hx-swap="none"
     {{if eq .Type "lead_assigned"}}
     hx-on::after-request="window.location.href='/tenant/leads?open={{.Metadata.lead_id}}'"
     {{end}}>
    <div class="toast-icon">{{typeIcon .Type}}</div>
    <div class="toast-body">
        <div class="toast-title">{{.Title}}</div>
        <div class="toast-text">{{.Body}}</div>
    </div>
    <button class="toast-close" onclick="this.parentElement.remove()">×</button>
    <script>setTimeout(function(){var el=document.getElementById('toast-{{.ID}}');if(el)el.remove();},5000);</script>
</div>
```

- Empilhamento natural por `hx-swap="beforeend"` no container.
- Toasts antigos saem por auto-dismiss (5s); novos empurrados pra baixo.
- Clique no toast sempre marca como lida; no caso de `lead_assigned`, após o `POST /read` navega pro kanban com `?open=<lead_id>`.

### 4. Página dedicada (`notification/list.html`)

Layout tenant padrão (sidebar + conteúdo), header "Notificações". Passa `ActiveNav=""` ao `tenant_sidebar.html` — nenhum item fica destacado (comportamento já suportado pela lógica `{{if .ActiveNav}}{{if eq .ActiveNav "X"}}active{{end}}{{end}}` do sidebar). Sem novo item no sidebar.

Tabs em `nav.tabs`:
- "Não lidas" (default) — `hx-get="/tenant/notifications/list?filter=unread"`.
- "Todas" — `hx-get="/tenant/notifications/list?filter=all"`.

Ambas carregam em `#notifications-list`. Cada item: ícone do tipo + título + corpo + timestamp ("há 2 min" via helper). Itens não lidas exibem botão "Marcar como lida" (hx-post individual). Itens `lead_assigned` mostram botão "Abrir lead" → deep-link.

Paginação: botões "Anterior" / "Próximo" com `offset`/`limit=20` via HTMX (backend já suporta).

Estado vazio: ícone + mensagem ("Nenhuma notificação ainda." para tab "Todas"; "Tudo em dia!" para "Não lidas").

### 5. Deep-link no kanban

**Alteração em `internal/funnel/interfaces/http/handler.go`** (handler `Kanban`): lê query `?open=<lead_id>`. Se presente:
1. Valida via `GetLeadDetailUseCase` que o lead existe e pertence ao tenant.
2. Se válido, passa `OpenLeadID` no template data.
3. Se inválido/cross-tenant: ignora silenciosamente (não retorna erro — log `warn`).

**Alteração em `web/templates/funnel/kanban.html`**:

```html
{{if .OpenLeadID}}
<div hx-get="/tenant/leads/{{.OpenLeadID}}"
     hx-trigger="load once"
     hx-target="#lead-modal"
     hx-swap="innerHTML"></div>
{{end}}
```

Reaproveita o drawer `#lead-modal` já existente (carregado pelo mesmo endpoint `GET /tenant/leads/:id` usado pelo clique no card).

## Mapeamento tipo → ícone/cor

Implementado via template helper `typeIcon` e classes CSS:

| Tipo (backend) | Ícone | Cor | Classe CSS |
|----------------|-------|-----|------------|
| `lead_assigned` | 👤 | azul | `notif-lead-assigned` |
| `lead_moved` | 🔀 | cinza | `notif-lead-moved` |
| `lead_handoff` | 🤝 | laranja | `notif-lead-handoff` |
| `lead_qualified` | ⭐ | verde | `notif-lead-qualified` |
| `rate_limit_reached` | 🚫 | vermelho | `notif-rate-limit` |
| `automation_error` | ⚠️ | vermelho | `notif-automation-error` |

Cores aplicadas na borda esquerda do item (2px) e no background do ícone — consistente com o padrão usado em alertas.

## Rotas HTML (novas)

Todas sob `/tenant/notifications`, protegidas por `authMw` + `tenantMw`. Sem permissão adicional (todo usuário autenticado vê as próprias notificações).

| Método | Rota | Retorna | Descrição |
|--------|------|---------|-----------|
| GET | `/tenant/notifications` | Página completa | Página dedicada com tabs |
| GET | `/tenant/notifications/list` | Fragmento | Lista + paginação (query: `filter`, `limit`, `offset`) |
| GET | `/tenant/notifications/dropdown` | Fragmento | 10 últimas do usuário autenticado |
| GET | `/tenant/notifications/badge` | Fragmento | `<span class="badge">N</span>` ou vazio |

Rotas JSON existentes permanecem sob `/notifications/*` (API) — são consumidas pelo frontend via HTMX com `hx-post`:

| Existente | Uso na UI |
|-----------|-----------|
| `POST /notifications/:id/read` | Clique individual marca lida |
| `POST /notifications/read-all` | Botão "Marcar todas" |

### Mudança no SSE handler (`GET /notifications/stream`)

Hoje emite JSON. Passa a emitir HTML fragment renderizado via `partials/notification_toast.html` + badge OOB swap:

```go
html := renderToast(notif) + renderBadgeOOB(unreadCount)
c.SSEvent("notification", html)
```

O `renderBadgeOOB` emite `<span id="notification-badge" hx-swap-oob="true">N</span>`, o que atualiza o contador do sino ao vivo sem polling. O toast vai pro `#toast-container` via `sse-swap="notification"` + `hx-swap="beforeend"`.

## Estrutura de arquivos

### Novos
```
internal/notification/interfaces/http/
  page_handler.go               # 4 handlers HTML novos
  page_handler_test.go          # testes unit (80%+)
  toast_render.go               # helper de renderização do toast+badge OOB
web/templates/notification/
  list.html                     # página completa
  list_items.html               # fragmento lista + paginação
  item.html                     # item reusado no dropdown e na lista (toast tem template próprio)
web/templates/partials/
  tenant_head.html              # <head> compartilhado (htmx + sse + css)
  notification_bell.html        # sino + badge + SSE container + dropdown placeholder
  notification_dropdown.html    # conteúdo do dropdown (10 últimas)
  notification_toast.html       # markup do toast emitido via SSE
web/static/css/notification.css # classes específicas (importado por main.css via @import)
```

### Alterações
```
internal/notification/interfaces/http/handler.go    # SSE passa a emitir HTML
internal/notification/interfaces/http/routes.go     # registra novo PageHandler
internal/notification/module.go                     # wire PageHandler + HTML templates
internal/funnel/interfaces/http/handler.go          # handler Kanban lê ?open=<lead_id>
web/templates/funnel/kanban.html                    # loader condicional do drawer
web/templates/{whatsapp,funnel,team,product,automation,ai}/*.html
                                                    # substituem <head> inline por tenant_head.html
                                                    # incluem notification_bell.html no body
web/static/css/main.css                             # @import notification.css (ou inline)
web/static/js/admin.js                              # +toggleNotificationDropdown() + click-outside handler
cmd/api/main.go                                     # (se necessário) registrar template helper typeIcon
```

## Permissionamento

- Sino visível para todo usuário autenticado no tenant (sem permissão específica).
- `tenantMw` garante isolamento: usuário de um tenant nunca vê notificações de outro.
- Filtro por `user_id` nos use cases já existentes (backend) — cada usuário vê apenas as próprias notificações, mesmo dentro do próprio tenant.
- Deep-link valida ownership do lead antes de abrir drawer (handler do kanban).

## Testes

### Unit (`page_handler_test.go`) — cobertura ≥ 80%
- `GET /tenant/notifications` renderiza página com tab "Não lidas" default.
- `GET /tenant/notifications/list?filter=unread` retorna só não lidas.
- `GET /tenant/notifications/list?filter=all` retorna todas.
- Paginação respeita `limit` e `offset`.
- `GET /tenant/notifications/dropdown` retorna exatamente 10 itens ordenados desc.
- `GET /tenant/notifications/badge` retorna número correto; vazio se zero.
- Estado vazio renderiza mensagem apropriada por tab.
- Item `lead_assigned` com `metadata.lead_id` renderiza botão "Abrir lead"; outros tipos não.

### OWASP (integração por rota)
- 401 sem token JWT em todas as 4 rotas HTML + SSE stream.
- 403 sem tenant context.
- Isolamento: usuário A **não** recebe notificações de usuário B no mesmo tenant (filtragem por `user_id`).
- Isolamento de tenant: notificações de tenant T2 invisíveis ao usuário do tenant T1.
- SSE filtra por `claims.UserID` antes de emitir (teste já coberto pelo handler atual — expandir cobertura).

### Deep-link no kanban
- `GET /tenant/leads?open=<lead_id>` com lead válido do tenant → `OpenLeadID` passado ao template; drawer carrega via HTMX.
- `?open=<lead_id>` com lead **inexistente** → ignorado silenciosamente (página renderiza normalmente, log warn).
- `?open=<lead_id>` com lead de **outro tenant** → ignorado silenciosamente (log warn; não 404 para evitar lado-canal de inferência de IDs).

### SSE (unit + integração)
- Conexão recebe apenas eventos do próprio `user_id`.
- HTML fragment renderizado é válido (testcontainers opcional; cobertura primária via unit do `renderToast`).
- Badge OOB swap incluído no payload quando há evento.

## Observabilidade

Métricas Prometheus (coerentes com padrão dos outros módulos):
- `crm_notifications_delivered_total{type}` — counter incrementado quando `NotifyService.Notify` persiste com sucesso.
- `crm_notifications_sse_active_streams` — gauge de streams abertos.
- `crm_notifications_http_requests_total{route,status}` — counter HTTP padrão para as 4 novas rotas.

Spans OTel nas 4 novas rotas HTML (`notification.page.list`, `notification.dropdown`, `notification.badge`, `notification.stream.emit`).

Logs estruturados (já há `zap.Logger` no handler): `notification_delivered`, `notification_marked_read`, `notification_mark_all_read`, incluindo `tenant_id` e `user_id`.

## Fora de escopo

Ficam para follow-ups dedicados:

- **Preferências de notificação** — backend pronto, mas canal WhatsApp ainda não tem emissor de saída. Oferecer preferências com único canal utilizável confunde o usuário.
- **Emissores dos tipos ainda não ativos** — `lead_moved`, `lead_handoff`, `lead_qualified`, `rate_limit_reached`. Cada um amarra a contexto específico (kanban move, IA handoff, qualification engine, WhatsApp gateway).
- **Som / desktop notifications** (Notification API do browser).
- **Admin area** — sem emissores direcionados a usuários admin hoje.
- **Reconexão SSE avançada** — `htmx-ext-sse@2.2.2` já faz backoff nativo; monitorar em produção antes de customizar.
- **Notificações em lote (digest)** — e-mail diário com resumo, ou agrupamento no próprio sino ("3 novos leads").
- **Filtros por tipo na página dedicada** — hoje só filtro "não lidas / todas".

## Dependências externas

- `htmx-ext-sse@2.2.2` — já carregado em `layouts/tenant.html`; será propagado via `partials/tenant_head.html`.
- Sem hyperscript. Toast usa `<script>` inline com `setTimeout` (2 linhas).
- Sem bibliotecas novas.

## Critérios de aceite (DoD)

- [ ] Sino aparece em todas as páginas do tenant com badge correto.
- [ ] Nova notificação aparece em toast + badge atualiza em até 1s (SSE).
- [ ] Dropdown abre com 10 últimas, botão "marcar todas" funciona, link "Ver todas" leva à página dedicada.
- [ ] Página dedicada com tabs "Não lidas" / "Todas" + paginação funcional.
- [ ] Clique em notif `lead_assigned` abre o lead no kanban (drawer automático via `?open=`).
- [ ] Clique em outros tipos marca como lida e fecha dropdown.
- [ ] Cobertura ≥ 80% em `internal/notification/interfaces/http`.
- [ ] Testes OWASP (401/403/isolamento) passando.
- [ ] Build + containers ok.
- [ ] `rest/notifications.http` criado com exemplos dos 4 novos endpoints.
