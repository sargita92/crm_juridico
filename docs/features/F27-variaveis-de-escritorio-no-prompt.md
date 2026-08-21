# F27 — Variáveis de escritório no prompt do especialista

- **Épico**: 2 — Admin — Tenants e Especialistas
- **Prioridade**: alta
- **Dependência**: F04, F05
- **Status**: backlog
- **Reportado**: 2026-08-21

## Relato

> "conseguir ter tipo uma parte que eu consigo complementar o prompt do
> especialista com algo para o escritório... exemplo: `Bom dia! Bem vindo ao
> escritorio {escritorio}`, aí pro escritório eu completo o prompt com `nome do
> escritório é Advs Unidoss`, aí ele vai mandar para o usuário `Bom dia! Bem
> vindo ao escritorio Advs Unidoss`. Uma forma de personalizar alguns detalhes
> para aquele escritório sem precisar ficar criando 500 especialistas com o
> prompt parecido e mudando só detalhes."

## Problema

Um especialista é compartilhado entre vários escritórios (associação N:N,
[`specialist_tenants`](../../migrations)). O prompt, porém, é **único e global**:
qualquer detalhe específico de um escritório — nome, endereço, horário de
atendimento, telefone, forma de tratamento — obriga a **clonar o especialista**
inteiro só para trocar uma frase.

O resultado é proliferação de especialistas quase idênticos: cada correção de
prompt precisa ser replicada N vezes, e as cópias divergem com o tempo. É o
mesmo problema que os guardrails compartilhados resolveram (`e907dac`), agora do
lado do prompt.

## Objetivo de negócio

Um especialista, N escritórios, cada um com sua personalização — sem cópias.

## Direção provável (a validar com Arquiteto)

Duas peças que se combinam:

1. **Placeholders no prompt do especialista** — `{escritorio}`, `{telefone}`,
   `{horario}` — interpolados na montagem do prompt.
2. **Valores por escritório** — preenchidos na associação especialista↔tenant ou
   num cadastro de variáveis do tenant.

Ponto de integração natural: [`ContextBuilder.Build`](../../internal/ai/application/context_builder.go),
que já monta persona + produto + documentos + guardrails + dados coletados. A
interpolação entra na etapa 1, sobre `specialist.Prompt`.

## Pontos a decidir

- Placeholder ausente ou não preenchido: erro, string vazia, ou valor default?
- Variáveis livres (o admin cria as chaves que quiser) ou catálogo fixo?
- Escopo: só o prompt, ou também steps e mensagens de guardrail?
- Pré-visualização no playground com os valores de um escritório escolhido.

## Não confundir com

Documentos de referência por especialista, que já existem e são globais. Aqui o
dado varia **por escritório**, não por especialista.
