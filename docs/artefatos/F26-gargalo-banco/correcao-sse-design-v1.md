---
feature: F26 — Correção (gargalo intermitente de banco → era SSE/HTTP1.1)
agent: Arquiteto / Brainstorming
version: 1
created_at: 2026-05-26
reason: Design da correção definitiva após a instrumentação (PR #15) localizar a causa-raiz
---

# F26 — Correção: uma única conexão SSE por página

## Causa-raiz confirmada (não era o banco)

A instrumentação da F26 (PR #15) descartou o banco: **durante a lentidão**, o app
ficava a ~0% CPU, o pool com `go_sql_wait_count=0` e o MySQL com ~4 conexões e
nenhuma query ativa/lenta. As medições mostraram **6 conexões TCP** abertas na
porta do app — o teto de **~6 conexões por host do HTTP/1.1**.

Cada página tenant abre **até 2 conexões SSE persistentes**:

- sino de notificações → `sse-connect="/notifications/stream"` (em ~toda página);
- aba WhatsApp/Playground → `sse-connect="/tenant/whatsapp/events"`.

A navegação da sidebar é **page load completo** (links `<a href>`, sem `hx-boost`).
Ao trocar de aba rápido, as conexões SSE da página antiga ainda não fecharam
quando a nova já abre as suas — o total ultrapassa 6 e **todo request novo
(o próprio SSE, o poll de 3s, assets) fica enfileirado no browser** → freeze de
~19s com o servidor ocioso.

## Objetivo

Garantir **uma única conexão SSE por página carregada**, eliminando a conexão
dupla (WhatsApp/Playground) e reduzindo a sobreposição durante a troca de abas.

## Design

### Backend — endpoint unificado `GET /tenant/stream` (auth + tenant)

Reaproveita o handler de stream de notificações (já tem `eventBus`, `renderer`,
`claims`, `listUC`). Única mudança de lógica: em vez de **pular** eventos que não
são `notification`, ele os **encaminha** como `event: <tipo>`:

| Evento no bus | Saída SSE |
|---------------|-----------|
| `notification` (do usuário autenticado) | `event: notification` + fragmento HTML (toast + OOB badge) — igual hoje |
| `new-message`, `conversation-update`, `lead-*` | `event: <tipo>`\n`data: {}` — igual ao handler atual do WhatsApp |

- Mantém keepalive periódico e a métrica `SSEActiveStreams`.
- Isolamento por tenant preservado (`Subscribe(tenantID)`); notification filtra por `UserID`.

Rotas **removidas**: `/notifications/stream` e `/tenant/whatsapp/events`
(o `HandleSSE` do WhatsApp vira código morto e sai).

### Frontend — um único `sse-connect` por página

- `partials/notification_bell.html`: **deixa de** ter `hx-ext="sse"`/`sse-connect`;
  vira **consumidor puro** (`sse-swap="notification"`).
- Em **todas** as páginas tenant, o wrapper `<div class="admin-layout">` recebe
  `hx-ext="sse" sse-connect="/tenant/stream"` — ancestral comum do sino e do conteúdo
  (placement **uniforme**).
- `whatsapp/page.html` e `ai/playground.html`: removem o `sse-connect` do
  `.wa-container`. Os consumidores internos (`hx-trigger="sse:new-message"`,
  `hx-trigger="sse:conversation-update"`) passam a ler do stream único (já são
  descendentes do `.admin-layout`).

Páginas tenant que incluem o sino (todas recebem o `sse-connect` no `.admin-layout`):
team (shell, group_detail), pagamentos (portal_unavailable, tenant_list),
notification/list, whatsapp/page, tenant/dashboard/page, funnel (kanban, list,
detail, form), files/list, ai/playground, product/product_list, automation/list.

### Degradação graciosa (inalterada)

Sino mantém fallback `every 30s`; chat mantém `every 5s`. Se o SSE cair, o
real-time degrada para polling — sem quebra funcional.

## Testes

- **Backend (integração):** `/tenant/stream` emite `notification` (com fragmento)
  para o usuário dono; **encaminha** `new-message`/`conversation-update`; exige
  auth (401 sem claims); isolamento por tenant. Reaproveita os testes SSE existentes.
- **Regressão:** rotas antigas removidas não quebram build/refs (`rest/` atualizado).
- **Validação manual (repro):** trocar de aba rápido com DevTools → confirmar
  **1 conexão `eventsource` por página** e ausência do freeze; `X-Response-Time`
  e métricas de pool estáveis (instrumentação da PR #15).

## Fora de escopo

- HTTP/2 (h2c) e SharedWorker (1 conexão por browser): seriam reforços extras,
  desnecessários dado o repro (abas da sidebar numa mesma aba do browser).
- Migrar a navegação para `hx-boost`/shell persistente (refactor maior).
- Tuning de pool / índices (banco descartado como causa).

## Rastreabilidade

- Causa-raiz e medições: [investigacao-v1.md](investigacao-v1.md) (atualizar com o achado SSE) e a sessão de diagnóstico ao vivo.
- Instrumentação que localizou o problema: PR #15 (`feature/F26-gargalo-banco`).
