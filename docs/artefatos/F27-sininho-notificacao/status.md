# Status F27 — Sininho de notificações (visibilidade + posição)

**Branch**: `feat/F27-sininho-notificacao`
**Status**: implementado (CSS) — build + testes verdes; smoke visual no app pendente
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
| — | design-v1 aprovado | concluído | — |
| 1 | Visibilidade via `:has` + reposição inferior-direito + dropdown p/ cima + toasts acima | concluído | (impl) |
| 2 | Exceção escopada `.wa-main` (sino/toasts acima do compositor) | concluído | (impl) |

## Verificação

- `go build ./cmd/... ./internal/... ./pkg/...` → sucesso.
- `go test ./internal/notification/...` → 84 testes verdes (nenhuma lógica Go alterada).
- CSS relido e conferido; static servido do disco (`router.Static("/static","web/static")`),
  então é aplicado em runtime sem rebuild.
- **Pendente (manual/visual no app logado):** checklist em
  [design-v1.md](design-v1.md#verificação-manualvisual) — sino some sem não lidas; aparece ao
  chegar não lida; dropdown abre para cima; toasts acima do sino; no chat o sino fica acima do
  compositor; topbar sem o sino por cima.
