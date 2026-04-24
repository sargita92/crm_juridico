# SpecialistFailing

**Severidade**: critical
**Fonte**: [`infra/prometheus/alerts.yml`](../../../infra/prometheus/alerts.yml) — grupo `crm_specialist`

## Sintoma

Proporção de respostas do motor de especialistas com `outcome="error"` maior que 10% nos últimos 5 minutos.

```promql
sum(rate(crm_specialist_response_duration_seconds_count{outcome="error"}[5m]))
  /
sum(rate(crm_specialist_response_duration_seconds_count[5m])) > 0.10
```

(A métrica é instrumentada em [`internal/shared/observability/metrics.go`](../../../internal/shared/observability/metrics.go) e observada em `internal/ai/application/`.)

## Impacto

- Conversas no WhatsApp param de receber respostas automáticas do especialista.
- Leads ficam "aguardando IA" no kanban até fallback humano.
- Clientes percebem o bot como "quebrado" — afeta a qualificação do lead.

## Diagnóstico

1. **Dashboard**: **CRM Jurídico — Especialistas** (uid `crm-especialistas`).
   - Painel *Respostas/min (por outcome)* mostra volume de erro vs. sucesso.
   - Painel *Latência p95 / p99* indica se é falha rápida ou timeout longo.

2. **Queries úteis**:

   ```promql
   # Volume absoluto de erro
   sum(rate(crm_specialist_response_duration_seconds_count{outcome="error"}[5m]))

   # Latência de erro vs. sucesso
   histogram_quantile(0.95,
     sum by (le, outcome) (rate(crm_specialist_response_duration_seconds_bucket[5m])))
   ```

3. **Logs** (diretório [`internal/ai/application/`](../../../internal/ai/application/)):
   - Filtrar por `specialist_id` e `tenant_id` — cruzar com o span `ai.usecase.respond` no Tempo.
   - Verificar mensagens de erro do provider LLM (chave expirada, rate limit, modelo removido).

4. **Provider externo**: se o motor usa API de terceiros (OpenAI, Anthropic, Bedrock), consultar status page do provider e ver se bate com o pico.

## Mitigação

- Se o erro é provider-side (chave / rate limit): trocar a chave ou aplicar backoff; se possível, alternar para provider secundário.
- Se o erro é de prompt/template: reverter último deploy do especialista em `internal/ai/` e abrir hotfix.
- Se for um único `specialist_id` problemático, desativá-lo temporariamente via admin.
- Se for rate limit de automações enviando mensagens à IA, consultar também `crm_automation_rate_limited_total` (dashboard *Leads / Kanban*).

## Causa raiz (pós-incidente)

- Registrar `specialist_id`, `tenant_id` e provider envolvidos.
- Adicionar teste de integração (mock do provider) se o bug foi regressão de código.
- Considerar instrumentar um counter `crm_specialist_response_total{reason="timeout|5xx|..."}` se o motivo do erro for recorrente (fora do escopo da F18).

## Escalation

1. Primeiros 15 min — on-call do backend / time de IA.
2. Persistindo, acionar responsável pelo pipeline de especialistas.
3. Se afeta múltiplos tenants com produto em contrato, escalar para Produto e Comercial.
