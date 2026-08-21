# F28 — Sinal de não-lido nas notas da conversa

- **Épico**: 3 — WhatsApp e Funil de Vendas
- **Prioridade**: média
- **Dependência**: F24, F27
- **Status**: backlog
- **Reportado**: 2026-08-21

## Relato

> "nas notas das conversas ter alguma forma de notificar que tem mensagem não
> lida."

## Problema

As notas da conversa (F24) vivem num drawer fechado por padrão
([`notes_panel.html`](../../web/templates/whatsapp/notes_panel.html)). Nada na
tela indica que há conteúdo novo lá dentro: para saber se um colega anotou
alguma coisa, é preciso abrir conversa por conversa.

Mensagens do WhatsApp já têm contador de não-lidas
([`unread_badge.html`](../../web/templates/whatsapp/unread_badge.html)); as notas
não têm equivalente.

## O que já existe (não construir de novo)

A plataforma tem um módulo de notificações completo
([`internal/notification/`](../../internal/notification/)) com exatamente as
peças que este pedido precisa:

- `Notification` é **por usuário** (`UserID`) e já carrega `Read bool`
  ([`domain/notification.go`](../../internal/notification/domain/notification.go)) —
  não é preciso tabela nova de leitura por usuário.
- Marcação de lido, listagem e preferências por usuário já implementadas
  (`mark_read.go`, `list_notifications.go`, `manage_preferences.go`).
- O sininho na topbar já existe e, desde a F27, **só aparece quando há não
  lidas** — que é literalmente o comportamento pedido aqui.
- Produtores já integrados servem de molde: o módulo de automação dispara via
  `NotifierAdapter` ([`adapters.go`](../../internal/automation/infrastructure/adapters.go)).

O trabalho provável, então, é **um tipo novo** (`TypeNoteCreated`, ao lado de
`TypeLeadAssigned`/`TypeLeadMoved`/…) mais o disparo no ponto de criação da nota,
e não uma feature de notificação do zero.

## Ambiguidade a resolver antes de especificar

O relato admite duas leituras:

- **(A) Nota não lida** — sinalizar que existe nota que *este usuário* ainda não
  viu. É a leitura que casa com "notas" no relato e a que não tem solução hoje.
- **(B) Mensagem não lida, exibida no drawer** — trazer para dentro do painel de
  notas o contador de mensagens que já existe. Barato, mas redundante com o badge
  atual.

**Confirmar com o reporter.** Assumindo (A), o custo caiu bastante depois de
mapear o módulo de notificações — vale reavaliar a prioridade.

## Objetivo de negócio

Quem abre uma conversa enxerga, sem procurar, que há contexto novo deixado pela
equipe — o valor da nota só existe se ela for lida.

## Pontos a decidir (assumindo A)

- Quem recebe: responsável pelo lead, todos do grupo, ou só quem já participou da
  conversa?
- Notificar também no drawer/lista de conversas, ou só pelo sininho da F27?
- Notas criadas pela IA ([`create_note`](../../internal/ai/infrastructure/tools/create_note.go))
  e por automação ([`auto_note`](../../internal/automation/application/executors/auto_note.go))
  notificam, ou só as escritas por humano? (Notificar as da IA pode virar ruído.)
- O autor da nota não deve receber notificação da própria nota.
- Preferência por usuário para desligar este tipo, como os demais já permitem.
