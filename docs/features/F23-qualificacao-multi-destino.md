# F23 — Qualificacao Multi-Destino

## Objetivo

Estender o motor de qualificacao de leads para suportar tres destinos de saida (aprovado / humano / reprovado) e adicionar cross-sell por regras explicitas configuradas por especialista.

## Pre-requisitos

- F16 (motor de IA dos especialistas) — pipeline de qualificacao base
- F07 (funis/kanban) — criacao e movimentacao de leads
- F10 (produtos) — resolucao de especialista por produto no cross-sell

## Status: concluido (2026-05-08)

## Design aprovado

Ver [docs/artefatos/F23-qualificacao-multi-destino/design-v1.md](../artefatos/F23-qualificacao-multi-destino/design-v1.md).

## Plano de implementacao

Ver [docs/artefatos/F23-qualificacao-multi-destino/plan-v1.md](../artefatos/F23-qualificacao-multi-destino/plan-v1.md) (11 steps, fases A/B/C).

## Regras de negocio

- **Outcome aprovado**: score >= `threshold_aprovacao` — lead avanca normalmente.
- **Outcome humano (faixa cinzenta)**: `threshold_humano_min` <= score < `threshold_aprovacao` — dispara `HandoffHumanRequested`; lead marcado com `qualification_outcome = human`.
- **Outcome reprovado**: score < `threshold_humano_min` — lead descartado.
- **Cross-sell por keyword**: engine avalia cada mensagem do lead contra regras antes de chamar o LLM; match dispara `CrossSellExecutor`.
- **Cross-sell por step_answer**: match na resposta de um step especifico do formulario de qualificacao.
- **Modos de cross-sell**: `announce` (IA anuncia), `silent` (transparente), `confirm` (aguarda confirmacao do lead).

## Endpoints novos

### JSON API (admin)

| Metodo | Rota | Descricao |
|--------|------|-----------|
| GET | `/admin/specialists/:id/cross-sell-rules` | Lista regras do especialista |
| POST | `/admin/specialists/:id/cross-sell-rules` | Cria regra |
| PUT | `/admin/specialists/:id/cross-sell-rules/:rule_id` | Atualiza regra |
| DELETE | `/admin/specialists/:id/cross-sell-rules/:rule_id` | Remove regra |
| POST | `/admin/specialists/:id/cross-sell-rules/:rule_id/move-up` | Reordena (sobe) |
| POST | `/admin/specialists/:id/cross-sell-rules/:rule_id/move-down` | Reordena (desce) |

### HTMX (fragmentos)

| Metodo | Rota | Descricao |
|--------|------|-----------|
| GET | `/admin/specialists/:id/cross-sell` | Renderiza secao cross-sell |
| POST | `/admin/specialists/:id/cross-sell/config` | Atualiza config (enabled, mode, template) |
| POST | `/admin/specialists/:id/cross-sell-rules/htmx` | Cria regra via form e re-renderiza secao |
| DELETE | `/admin/specialists/:id/cross-sell-rules/htmx/:rule_id` | Remove e re-renderiza |
| POST | `/admin/specialists/:id/cross-sell-rules/htmx/:rule_id/move-up` | Reordena e re-renderiza |
| POST | `/admin/specialists/:id/cross-sell-rules/htmx/:rule_id/move-down` | Reordena e re-renderiza |

## Criterios de aceite

- [x] OutcomeCalculator retorna tres zonas com base nos thresholds configurados
- [x] Engine dispara HandoffHumanRequested ao detectar outcome humano
- [x] Lead persiste `qualification_outcome` e `cross_sell_origin_lead_id`
- [x] CrossSellRuleEvaluator avalia keyword (case-insensitive) e step_answer
- [x] CrossSellExecutor executa nos modos announce/silent/confirm
- [x] Modo confirm persiste `pending_cross_sell_rule_id` na ConversationState
- [x] CRUD de CrossSellRule com reordenacao e isolamento de tenant (OWASP)
- [x] Metricas Prometheus: `qualification_outcome_total`, `cross_sell_triggered_total`
- [x] Tela de scoring atualizada com threshold humano e colunas humano/cross-sell
- [x] Templates HTMX para configuracao de cross-sell no painel do especialista
- [ ] Testcontainers: rodar suite completa sem -short antes do merge (WSL2 limitado durante impl)
