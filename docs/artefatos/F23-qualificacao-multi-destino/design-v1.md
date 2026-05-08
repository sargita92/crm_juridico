# Qualificação multi-destino — Design

**Data**: 2026-05-07
**Origem**: Evolução do sistema de qualificação por pontuação (F05 Step 5 / F16 Step 5–6).
**Branch alvo**: a definir (sugestão: `feature/Fxx-qualificacao-multi-destino`).

## 1. Contexto

Hoje o motor de qualificação tem dois desfechos no fim do script de atendimento:

- `score >= threshold_qualificacao` → **aprovado** (move pra coluna "qualificado").
- `score < threshold_qualificacao` → **reprovado** (move pra coluna "desqualificado").

A operação real precisa de dois caminhos intermediários:

1. **Cross-sell**: o lead não é fit pro produto atual mas é fit pra outro produto do tenant — deve ser transferido para o especialista responsável por esse outro produto.
2. **Atendimento humano**: leads em faixa cinzenta de score precisam ser validados por um humano antes de virar aprovado/reprovado, em vez de cair no balde "desqualificado".

Este design adiciona esses dois desfechos com mudança mínima no modelo atual e mantém retrocompatibilidade.

## 2. Objetivos e não-objetivos

**Objetivos**

- Suportar 4 destinos finais: `aprovado`, `humano`, `cross_sell`, `reprovado`.
- Permitir cross-sell por **regras explícitas** (palavra-chave, resposta de step, intent) e por **sugestão da IA** (tool call).
- Permitir atendimento humano por **faixa cinzenta de score**.
- Permitir que cada funil mapeie destino → coluna do kanban.
- Configurar transição de cross-sell (silenciosa, anunciada ou com confirmação).
- Preservar comportamento atual quando o tenant não configura faixas/regras novas.

**Não-objetivos (ficam pra evolução futura)**

- Engine de regras genérico (Approach B do brainstorm).
- FSM declarativa pro ciclo de vida do lead (Approach C).
- Triggers extras pra humano (sentimento, pedido explícito, guardrail repetitivo) — só faixa de score por agora.
- Load balance ou notificação push de leads em fila humana (reutiliza o que F07/F08/F09 já oferecem).

## 3. Modelo de dados

### 3.1 Mudanças em entidades existentes

**`scoring_configs`**

- Renomear `threshold_qualificacao` → `threshold_aprovado` (`int`, score mínimo para aprovado).
- Adicionar `threshold_humano_min` (`int`, default `0`). Score nessa faixa cai em `humano`. Quando `0`, comportamento binário atual é preservado.
- Invariante: `0 ≤ threshold_humano_min ≤ threshold_aprovado ≤ scoring_configs.total_pontos`.

**`specialists`**

- `cross_sell_enabled` (`bool`, default `false`).
- `cross_sell_mode` (`enum: announce | silent | confirm`, default `announce`).
- `cross_sell_announcement_template` (`text`, nullable). Suporta placeholder `{{produto}}`.
- `allow_ai_cross_sell_suggestion` (`bool`, default `false`). Habilita tool call da IA além das regras explícitas.

**`leads`**

- `qualification_outcome` (`enum: em_andamento | aprovado | humano | cross_sell | reprovado`, default `em_andamento`).
- `cross_sell_origin_lead_id` (FK self, nullable). Aponta pro lead anterior quando o lead foi criado por transferência de cross-sell.

### 3.2 Entidades novas

**`cross_sell_rules`** — regra explícita por especialista.

| Campo | Tipo | Notas |
|---|---|---|
| `id` | PK | |
| `specialist_id` | FK | NOT NULL |
| `ordem` | int | Avaliação por ordem crescente |
| `trigger_type` | enum | `keyword` \| `step_answer` \| `intent` |
| `trigger_config` | json | Schema depende do tipo (ver 3.3) |
| `target_product_id` | FK products | NOT NULL |
| `ativo` | bool | default `true` |
| `created_at`, `updated_at` | timestamps | |

Constraints:

- `target_product_id` precisa ter pelo menos um specialist vinculado (`specialist_products`) que **não** seja o `specialist_id` da regra.
- Validação de tenant: `specialist.tenant_id == product.tenant_id`.

**`funnel_outcome_mappings`** — mapeia destino final em coluna por funil.

| Campo | Tipo | Notas |
|---|---|---|
| `funnel_id` | FK | |
| `outcome` | enum | `aprovado` \| `humano` \| `cross_sell` \| `reprovado` |
| `column_id` | FK | NOT NULL |
| PK | (`funnel_id`, `outcome`) | |

Constraint: `column.funnel_id == funnel_id`.

### 3.3 Schema de `trigger_config`

- `keyword`: `{ "termos": ["trabalhista", "rescisão"] }` — match case-insensitive em qualquer mensagem do cliente.
- `step_answer`: `{ "step_id": 7, "regex": "(?i)^sim$" }` — match na resposta capturada do step.
- `intent`: `{ "intent_name": "duvida_trabalhista" }` — depende da lista de intents cadastrados no tenant (extensão de F16).

### 3.4 Migration

- Renomear coluna em `scoring_configs` (rename, não drop+add) para preservar dados.
- Adicionar coluna `threshold_humano_min` com default `0`.
- Backfill de `funnel_outcome_mappings`: para cada funil existente, mapear `aprovado` → coluna "qualificado", `reprovado` → "desqualificado". `humano` e `cross_sell` apontam pra "desqualificado" por default até o tenant configurar.
- Adicionar colunas em `specialists` e `leads` com defaults seguros.

## 4. Lógica do engine

### 4.1 Avaliação de regras de cross-sell

A cada mensagem do cliente, **antes** de invocar o LLM:

1. Carrega `cross_sell_rules` ativas do specialist atual ordenadas por `ordem ASC`.
2. Para cada regra, avalia trigger:
   - `keyword`: normaliza mensagem (lower, sem acentos), procura qualquer termo da lista.
   - `step_answer`: olha resposta capturada do `step_id`; aplica regex.
   - `intent`: depende do classificador de intent (mensagem atual ou contexto curto).
3. **Primeiro match vence**. Interrompe avaliação e dispara cross-sell (4.3).
4. Se nenhuma regra dispara, segue fluxo normal (LLM).

### 4.2 Tool call da IA

Quando `specialist.allow_ai_cross_sell_suggestion = true` e nenhuma regra disparou, o LLM recebe a tool:

```
suggest_cross_sell(target_product_id: int, reason: string)
```

Engine valida ao receber a chamada:

- `target_product_id` existe no tenant.
- Tem especialista vinculado distinto do atual.
- Tenant não bloqueou cross-sell pra esse par origem→destino (futuro; por agora não há blocklist).

Se válido, dispara cross-sell (4.3). Se inválido, ignora a tool call e registra warning.

### 4.3 Execução do cross-sell

1. Resolver novo specialist a partir do `target_product_id` (primeiro vinculado, ordem estável).
2. Aplicar `cross_sell_mode` do specialist atual:
   - `announce`: envia template renderizado (`{{produto}}` → nome do produto alvo).
   - `silent`: nenhuma mensagem.
   - `confirm`: envia pergunta e aguarda resposta afirmativa (regex configurável; default `(?i)^(sim|s|ok|claro)\b`). Em caso de resposta negativa, mantém specialist atual e marca regra como "consultada e recusada" (não tenta novamente nessa conversa).
3. Marca lead atual: `qualification_outcome = cross_sell`, move pra coluna mapeada `cross_sell` no funil atual.
4. Cria novo Lead:
   - Mesmo `wa_contact_id`, mesmo tenant.
   - `funnel_id` = funil padrão do novo specialist; `column_id` = coluna inicial.
   - `cross_sell_origin_lead_id` = ID do lead atual.
   - Reseta scoring (novo specialist tem seus próprios steps/pontos).
5. Conversa migra de specialist atualizando o campo correspondente na entidade Conversation (campo exato a confirmar no plano — F06/F16 usa `specialist_id`). Histórico fica acessível, mas o ContextBuilder do novo specialist trunca/resume conforme orçamento de tokens.
6. Auditoria: registra evento `cross_sell_executed` com `from_lead_id`, `to_lead_id`, `trigger` (`rule_id` ou `ai_tool_call`), `mode`, `reason`.

### 4.4 Cálculo do outcome final

Quando todos os steps obrigatórios são concluídos:

```
if score >= threshold_aprovado:
    outcome = aprovado
elif score >= threshold_humano_min and threshold_humano_min > 0:
    outcome = humano
else:
    outcome = reprovado
```

Quando `threshold_humano_min = 0`, a faixa `humano` desaparece e cai no comportamento binário atual.

### 4.5 Comportamento em `humano`

- `lead.qualification_outcome = humano`, move pra coluna mapeada.
- Conversation: `ai_paused = true` (mesmo mecanismo do F16 Step 8).
- Dispara notificação interna pros usuários do tenant com permissão no funil (reutiliza pipeline F08/F09).
- Usuário pode:
  - Assumir conversa, atender, e depois marcar manualmente `aprovado` ou `reprovado` (transição final).
  - Devolver pra IA (`ai_paused = false`), caso queira que ela retome.

### 4.6 Comportamento em `aprovado` e `reprovado`

Sem mudança em relação ao fluxo atual: move pra coluna mapeada e dispara automações (F09).

## 5. Interfaces / Telas

### 5.1 Treinamento do especialista (estende F05 Step 5)

- Bloco "Pontuação":
  - Inputs `threshold_aprovado` e `threshold_humano_min` com validação inline.
  - Texto de ajuda dinâmico: "Score ≥ X aprovado · Y–Z humano · < Y reprovado".
- Bloco "Cross-sell":
  - Toggle `cross_sell_enabled`.
  - Radio `cross_sell_mode` (announce/silent/confirm).
  - Textarea `cross_sell_announcement_template` (com preview do placeholder).
  - Checkbox `allow_ai_cross_sell_suggestion`.
- Bloco "Regras de cross-sell":
  - Listagem ordenada das regras com reordenar via mover cima/baixo (mesmo padrão de F05 Step 4).
  - Adicionar/editar regra: tipo de trigger, config conforme tipo (3.3), produto alvo (select com produtos do tenant que tenham specialist vinculado), ativo.

### 5.2 Configuração de funil (estende F07)

- Bloco "Destinos":
  - 4 selects: aprovado, humano, cross_sell, reprovado → colunas do funil.
  - Validação: cada destino tem coluna distinta? (warning, não bloqueio — pode coexistir).

## 6. Compatibilidade e rollout

- `threshold_humano_min = 0` por default ⇒ tenants atuais não percebem mudança.
- `cross_sell_enabled = false` por default ⇒ regras e tool call não são avaliados.
- Backfill cobre `funnel_outcome_mappings` pra todos os funis existentes.
- Renomear `threshold_qualificacao` → `threshold_aprovado` é breaking em qualquer SQL/relatório que use o nome antigo. Auditar `internal/` e `web/templates/` antes de migrar e atualizar referências.

## 7. Observabilidade

Métricas (Prometheus):

- `qualification_outcome_total{tenant,specialist,outcome}` — contador por desfecho.
- `cross_sell_triggered_total{tenant,specialist,trigger_type}` — `rule_keyword`, `rule_step_answer`, `rule_intent`, `ai_tool_call`.
- `cross_sell_confirmation_declined_total{tenant,specialist}`.
- `human_handoff_queue_size{tenant,funnel}` — gauge.
- `human_handoff_resolution_latency_seconds{tenant}` — histograma do tempo entre virar `humano` e receber decisão final.

Logs (slog) com `tenant_id`, `lead_id`, `specialist_id`, `outcome`, `trigger_type`, `target_product_id`.

Traces: spans em `engine.evaluate_rules`, `engine.execute_cross_sell`, `engine.compute_outcome`.

## 8. Testes

Cobertura ≥ 80% por convenção do projeto.

**Domínio**

- Cálculo de outcome com 3 zonas e com `threshold_humano_min = 0` (binário).
- Validação de invariante de thresholds.
- Validação de `trigger_config` por tipo.

**CrossSellRule**

- Matcher `keyword` com normalização de acentos/case.
- Matcher `step_answer` com regex válido/inválido.
- Matcher `intent` com nome desconhecido (não match).
- Ordenação: regra de menor `ordem` ganha em caso de múltiplos matches.
- Regra inativa não dispara.

**Engine**

- Regra dispara cross-sell antes de chamar LLM.
- Tool call da IA respeitada quando flag ligada e nenhuma regra match.
- Tool call ignorada quando flag desligada.
- Validação de `target_product_id` (não existe, sem specialist, mesmo specialist).

**Transição cross-sell**

- `announce` envia template renderizado.
- `silent` não envia mensagem.
- `confirm` aguarda resposta; afirmação dispara, negação preserva specialist atual.
- Novo lead criado com `cross_sell_origin_lead_id` correto.
- Conversation migra `current_specialist_id`.

**Outcome final**

- Pausa IA em `humano`.
- Move pra coluna mapeada.
- Notificação disparada pros usuários do funil.
- Usuário marca aprovado/reprovado manualmente após atendimento humano.

**Mapeamento de funil**

- `FunnelOutcomeMapping` ausente pra um destino: fallback documentado (mantém na coluna atual + log warning).
- Coluna referenciada não pertence ao funil: rejeita no save.

**OWASP**

- Tenant A não consegue criar regra apontando pra produto do tenant B.
- Tenant A não consegue mapear coluna do funil do tenant B.
- Cross-sell não atravessa fronteira de tenant.
- Endpoints exigem 401/403 conforme F02.

## 9. Riscos e mitigações

| Risco | Mitigação |
|---|---|
| Renomear coluna `threshold_qualificacao` quebra código antigo | Audit + replace em uma única PR; testes de regressão; deploy com migration reversível |
| Cross-sell em loop (A→B→A) | Bloquear cross-sell se `cross_sell_origin_lead_id` ancestral já passou pelo specialist alvo |
| Regras conflitantes/ambíguas | Ordenação explícita + UI que destaca primeira que dispararia em mensagem de teste |
| Threshold inconsistente (humano_min > aprovado) | Validação no domain + no form |
| Pool humano sem ninguém pra atender | Métrica `human_handoff_queue_size` + alerta a partir de threshold (config futura) |

## 10. Backlog de evolução (fora do escopo)

- Triggers adicionais pra humano: sentimento negativo, pedido explícito, guardrail violado N vezes.
- Load balance/round-robin no humano usando lógica de responsável de F07.
- Engine de regras genérico (Approach B) caso o número de regras de cross-sell + humano cresça.
- FSM declarativa por specialist (Approach C) caso o ciclo de vida do lead fique mais complexo.
- Blocklist de pares origem→destino de cross-sell.
