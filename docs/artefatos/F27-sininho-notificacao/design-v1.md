# Design F27 — Sininho de notificações (visibilidade + posição)

**Data**: 2026-05-26
**Branch**: `feat/F27-sininho-notificacao`
**Tipo**: melhoria de UI (somente CSS)

## Problema

O sino de notificações tem dois incômodos:

1. **Sempre visível.** O botão do sino aparece em toda página tenant mesmo sem
   nenhuma notificação; só o badge de contagem some quando zerado
   (`partials/notification_badge.html` alterna a classe `notification-badge-hidden`).
2. **Atrapalha outros ícones.** Ele é `position: fixed; top: 16px; right: 16px`
   (`web/static/css/notification.css`), flutuando **por cima** dos ícones da topbar
   (busca, Conversas, Leads — `partials/tenant_topbar.html`) nas 12 páginas que usam
   topbar.

## Objetivo

- O sino só aparece quando há notificações **não lidas**.
- Reposicioná-lo para um canto que não cubra outros ícones.

## Decisões (validadas com o usuário)

- **Esconder quando não há não lidas** (badge zerado); reaparece ao chegar algo novo.
  Aceito o efeito colateral: enquanto está tudo lido, o sino some e, com ele, o acesso
  rápido à página `/tenant/notifications` (que não está na sidebar). Sem atalho extra
  na sidebar por ora.
- **Floating, partial único** para todas as páginas (sem integrar na topbar).
- **Canto inferior-direito** como base, agrupado com os toasts (que já vivem ali).
- **Exceção das páginas de chat** (WhatsApp e AI Playground): o compositor de mensagem
  ocupa o inferior-direito, então o sino sobe nessas páginas para não cobrir o botão de
  enviar.

## Solução

### 1. Visibilidade — só CSS, zero JS

O estado "tem não lidas?" já é mantido em tempo real pela classe `notification-badge-hidden`
do badge, por três caminhos que **já existem**:

- OOB do SSE (`toast_render.go` emite toast + `notification_badge_oob.html`; aplicado por
  `sse-bridge.js`);
- poll `every 30s` no wrapper do badge;
- evento `refreshBadge from:body` (ex.: após "marcar todas como lidas").

Logo, dá para derivar a visibilidade do sino só com CSS, usando `:has()`:

```css
.notification-bell-container { display: none; }                 /* escondido por padrão */
.notification-bell-container:has(.notification-badge:not(.notification-badge-hidden)) {
    display: block;
}
```

Consequências:

- 0 não lidas → sino some; chega notificação (OOB) → reaparece na hora; "marcar todas como
  lidas" → some.
- **Sem flash no load**: começa escondido; só aparece se o badge vier > 0.
- O `<span>` do badge segue no DOM mesmo com o container escondido (`display:none` no pai
  não remove o filho), então poll e SSE continuam funcionando e reativam o sino.

**Comportamento conhecido (aceito):** se o dropdown estiver aberto e o usuário marcar todas
como lidas, o sino — e o dropdown junto — somem na hora. Coerente com "esconder quando não
há não lidas". Suavizar (manter visível enquanto o dropdown está aberto) fica como melhoria
opcional futura, não incluída nesta entrega.

### 2. Posição — base inferior-direito

Em `web/static/css/notification.css`:

| Seletor | Antes | Depois |
|---|---|---|
| `.notification-bell-container` | `top: 16px` | `bottom: 16px` (mantém `right: 16px`, `z-index: 1000`) |
| `.notification-dropdown` | `top: 52px` | `bottom: 56px; top: auto` (abre para cima) |
| `.toast-container` | `bottom: 16px` | `bottom: 72px` (empilha acima do sino) |

### 3. Exceção das páginas de chat (`.wa-main`)

WhatsApp e AI Playground usam `<main class="admin-content wa-main">` e têm o compositor
(`.wa-input-area`, ~60px) no inferior-direito. Sobe o sino e os toasts só nessas páginas,
via CSS escopado com `:has()` (sem JS, sem alterar template):

```css
.admin-layout:has(.wa-main) .notification-bell-container { bottom: 84px; }
.admin-layout:has(.wa-main) .toast-container { bottom: 140px; }
```

(Valores de px aproximados; ajuste fino na implementação para folga sobre o compositor.)

## Escopo e impacto

- **Arquivo alterado: somente `web/static/css/notification.css`.** Sem mudança em template,
  JS, Go, endpoints ou migrations.
- Sem nova lógica Go → sem novos testes unitários; os testes de notificação existentes
  permanecem intocados e devem seguir verdes.
- Regras 13 (OWASP por endpoint) e 14 (`rest/.http`) não se aplicam — não há endpoint novo.
- Cobertura inalterada (sem código Go novo).
- Dependência: seletor CSS `:has()` (suportado em navegadores modernos — adequado para um
  CRM interno).

## Verificação (manual/visual)

1. 0 não lidas → sino **não** aparece.
2. Chega notificação não lida (SSE) → sino aparece na hora, badge com a contagem.
3. Abrir o dropdown → abre **para cima**, alinhado ao canto inferior-direito.
4. Toasts → empilham **acima** do sino, sem sobrepô-lo.
5. Páginas de chat (WhatsApp/AI Playground) → sino fica **acima** do compositor; botão de
   enviar livre.
6. Páginas com topbar → topo-direito (busca/Conversas/Leads) **sem** o sino por cima.
7. "Marcar todas como lidas" → sino some.

## Alternativas consideradas (descartadas)

- **Sino dentro da topbar** (ícone alinhado): único jeito 100% sem sobreposição, mas exige
  tratamento separado nas 3 páginas sem topbar e mais trabalho; usuário preferiu floating.
- **Inferior-esquerdo**: conflita com o rodapé da sidebar (Trocar de escritório / Sair).
- **Topo-direito deslocado**: frágil, volta a sobrepor conforme a topbar mude.
