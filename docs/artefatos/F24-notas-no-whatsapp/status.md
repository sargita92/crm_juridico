# Status F24 — Notas do cliente na tela de WhatsApp

**Branch**: `feature/F24-notas-no-whatsapp`
**Status**: em andamento (design aprovado)
**Design**: [design-v1.md](design-v1.md)
**Plano**: plan-v1.md (a gerar)

## Resumo da feature

Dá acesso às notas do cliente (`LeadNote`, hoje só no drawer do Kanban) direto na tela
de WhatsApp, via um drawer deslizante acionado por um botão "📝 Notas" no cabeçalho do
chat. Permite ver e adicionar notas. Em conversas com vários leads (cross-sell), opera
sobre o lead atual da conversa (o mais recente por `created_at`).

Arquitetura: porta `LeadNotesService` no `whatsapp/domain` implementada por um adapter
no `funnel`, espelhando o padrão do `LeadCreator`. Reusa entidade/tabela/use case de
notas; sem migration nova.

## Fluxo de agentes

- PO / UI/UX / Arquiteto: design consolidado em design-v1.md (brainstorming).
- Dev Backend: pendente
- Dev Front-end: pendente
- QA: pendente (OWASP + cobertura ≥80%)
- Segurança: pendente

## Commits da feature

| Commit | Descrição |
|--------|-----------|
| (a preencher) | |
