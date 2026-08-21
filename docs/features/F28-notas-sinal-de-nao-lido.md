# F28 — Sinal de não-lido nas notas da conversa

- **Épico**: 3 — WhatsApp e Funil de Vendas
- **Prioridade**: média
- **Dependência**: F06, F07
- **Status**: backlog
- **Reportado**: 2026-08-21

## Relato

> "nas notas das conversas ter alguma forma de notificar que tem mensagem não
> lida."

## Problema

As notas da conversa ([`notes_panel.html`](../../web/templates/whatsapp/notes_panel.html))
vivem num drawer fechado por padrão. Nada na tela indica que há conteúdo novo lá
dentro: para saber se um colega anotou alguma coisa, é preciso abrir conversa por
conversa.

Mensagens do WhatsApp já têm contador de não-lidas
([`unread_badge.html`](../../web/templates/whatsapp/unread_badge.html),
[`get_unread_total.go`](../../internal/whatsapp/application/get_unread_total.go));
as notas não têm equivalente.

## Ambiguidade a resolver antes de especificar

O relato admite duas leituras, com implementações bem diferentes:

- **(A) Nota não lida** — sinalizar que existe nota que *este usuário* ainda não
  viu. Exige rastrear leitura **por usuário × nota** (tabela nova), já que a nota
  é escrita por um membro da equipe para os outros.
- **(B) Mensagem não lida, exibida no drawer** — reaproveitar o contador de
  mensagens que já existe, apenas trazendo o sinal para dentro do painel de
  notas. Bem mais barato, mas provavelmente redundante com o badge atual.

**Confirmar com o reporter antes de escrever a spec.** A leitura (A) é a que
casa com "notas" no relato e é a que não tem solução hoje.

## Objetivo de negócio

Quem abre uma conversa enxerga, sem procurar, que há contexto novo deixado pela
equipe — o valor da nota só existe se ela for lida.

## Pontos a decidir (assumindo A)

- Marcar como lida ao abrir o drawer, ou exigir ação explícita?
- Badge por conversa na lista, ou só dentro do chat aberto?
- Notas criadas pela IA ([`create_note`](../../internal/ai/infrastructure/tools/create_note.go))
  e por automação ([`auto_note`](../../internal/automation/application/executors/auto_note.go))
  contam como não lidas?
- Propagação em tempo real via SSE, como os demais eventos da tela.

## Relacionado

Depende de a conversa ter lead, que é pré-requisito das notas hoje — ver a nota
sobre conversas sem lead em [F07](F07-funis-kanban.md) e o débito de UX do funil.
