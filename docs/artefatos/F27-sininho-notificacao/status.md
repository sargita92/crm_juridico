# Status F27 — Sininho de notificações (visibilidade + posição)

**Branch**: `feat/F27-sininho-notificacao`
**Status**: design aprovado — aguardando plano de implementação
**Design**: [design-v1.md](design-v1.md)

## Resumo

Melhoria de UI (somente CSS) no sino de notificações:

1. Só exibir o sino quando há notificações **não lidas** (via CSS `:has()` sobre a classe
   `notification-badge-hidden` que o badge já mantém — zero JS/Go novo).
2. Mover o sino do topo-direito (onde cobre os ícones da topbar) para o **inferior-direito**,
   agrupado com os toasts. Nas páginas de chat (WhatsApp/AI Playground) o sino sobe para não
   cobrir o compositor de mensagem.

Único arquivo alterado: `web/static/css/notification.css`. Sem template/JS/Go/endpoint/migration.

## Decisões validadas

- Esconder quando não há não lidas (sem atalho extra na sidebar por ora).
- Floating com partial único (não integrar na topbar).
- Base inferior-direito; exceção escopada para `.wa-main` (chat).
- Efeito "poof" ao marcar todas como lidas com dropdown aberto: aceito; suavização fica como
  melhoria futura opcional.

## Progresso por step

| Step | Descrição | Status | Commit |
|------|-----------|--------|--------|
| — | design-v1 aprovado | concluído | (este) |
| 1 | Visibilidade + reposição base + dropdown para cima + toasts | pendente | — |
| 2 | Exceção escopada `.wa-main` (sino/toasts acima do compositor) | pendente | — |

## Verificação

Manual/visual (sem lógica Go nova) — ver checklist em [design-v1.md](design-v1.md#verificação-manualvisual).
