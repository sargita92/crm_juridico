# F29 — Tools do Especialista: usabilidade e flexibilidade

- **Épico**: 7 — IA e MCP
- **Prioridade**: média
- **Dependência**: F15
- **Status**: backlog
- **Reportado**: 2026-08-21

## Relato

> "melhorar o 'Tools do Especialista' que hoje é difícil de entender e não parece
> ser flexível."

## Estado atual

[`tools_section.html`](../../web/templates/specialist/tools_section.html) lista as
tools agrupadas em três categorias com rótulo em português (Consulta de Dados,
Ações no CRM, Automações), e cada linha tem Adicionar / Associada+remover.

Onde trava:

1. **Nome cru de programador.** A linha mostra `{{.Name}}` direto do registry:
   `create_note`, `move_lead`, `update_score`, `switch_specialist`. Só a
   categoria foi traduzida — a tool em si não.
2. **Descrição escrita para o LLM, não para o operador.** O texto vem do
   `ToolDefinition`, que existe para o modelo decidir quando chamar a função.
   Quem configura o especialista lê uma instrução dirigida a outro leitor.
3. **Não diz quando dispara.** A tela informa que a tool está associada, nunca em
   que situação o especialista vai usá-la, nem o que acontece com o lead depois.
4. **Restrição por etapa fica em outra tela.** `ForcedTools` e `RestrictedTools`
   são configurados no editor de steps
   ([`tool_resolver.go`](../../internal/ai/application/tool_resolver.go)), mas é
   ali que se decide o que a tool realmente faz em cada momento. Quem está na
   tela de Tools não vê essa metade.
5. **Catálogo fechado — a "falta de flexibilidade".** As dez tools são
   registradas em código
   ([`internal/ai/infrastructure/tools/`](../../internal/ai/infrastructure/tools/)).
   O admin escolhe entre as existentes e nada mais: não dá para criar uma tool,
   parametrizar uma existente, nem apontar para um serviço próprio. Toda
   necessidade nova vira deploy.

## Objetivo de negócio

Quem monta um especialista entende, sem ajuda do dev, o que cada tool faz e
quando ela age — e consegue cobrir caso novo sem esperar release.

## Direção provável (a validar com PO e Arquiteto)

Duas frentes, separáveis em entregas:

- **Usabilidade (curto prazo)**: rótulo humano e descrição voltada ao operador
  por tool, uma linha de "dispara quando", e visibilidade das restrições de etapa
  aqui mesmo. Não mexe em domínio — é catálogo de apresentação, como
  `guardrailTypes` e `stepDataTypes` já fazem em seus handlers.
- **Flexibilidade (maior)**: tool configurável pelo admin. O caminho natural é o
  MCP interno já previsto em [F15](F15-mcp-interno-especialistas.md) — em vez de multiplicar
  tools nativas, permitir apontar o especialista para um endpoint MCP do próprio
  escritório.

## Pontos a decidir

- Rótulo e texto ficam em código (catálogo estático) ou viram dado editável?
- A tela mostra as restrições de etapa só para leitura, ou permite editar dali?
- Flexibilidade via MCP (F15) ou via tool genérica de webhook parametrizável?
- Precisa de simulação — "testar esta tool neste especialista" — antes de salvar?
