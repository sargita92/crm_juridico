# HighHTTPLatency

**Severidade**: warning
**Fonte**: [`infra/prometheus/alerts.yml`](../../../infra/prometheus/alerts.yml) — grupo `crm_http`

## Sintoma

p95 global de `http_request_duration_seconds` maior que 2 segundos, sustentado por 5 minutos.

```promql
histogram_quantile(0.95,
  sum by (le) (rate(http_request_duration_seconds_bucket[5m]))
) > 2
```

## Impacto

- Usuários percebem a UI "travando"; HTMX swaps ficam lentos.
- Timeouts em clientes (front, webhook, automações encadeadas) amplificam o problema.
- Leads podem ser distribuídos com atraso, quebrando SLA interno de resposta.

## Diagnóstico

1. **Dashboard**: **CRM Jurídico — Overview** (uid `crm-overview`).
   - Painel *Latency p95 / p99 (global)* confirma.
   - Painel *Top 5 endpoints by latency p95* identifica gargalos.
   - Cruzar com **CRM Jurídico — WhatsApp** (`crm-whatsapp`) e **Leads / Kanban** (`crm-leads-kanban`) se o culpado for rota dessas features.

2. **Queries úteis**:

   ```promql
   # p95 por endpoint (descobrir os mais lentos)
   topk(10, histogram_quantile(0.95,
     sum by (le, method, path) (rate(http_request_duration_seconds_bucket[5m]))))

   # Distribuição de buckets de um endpoint específico
   sum by (le) (rate(http_request_duration_seconds_bucket{path="/tenants/.../leads"}[5m]))
   ```

3. **Logs**: filtrar os `request_id` das rotas lentas e inspecionar duração por camada (handler, use case, infra). Logger central é [`internal/shared/observability/logging.go`](../../../internal/shared/observability/logging.go).

4. **Traces** (Tempo): qualquer request lento traz `trace_id` no log. Abrir o trace e identificar o span mais demorado (geralmente GORM, chamada whatsmeow, ou motor de IA em `internal/ai/`).

5. **Recursos**: conferir CPU/mem do container `app` e do `mysql`; cache Redis se houver; conexões abertas do pool GORM.

## Mitigação

- Reduzir volume de automações pesadas em `internal/automation/infrastructure/executor*` temporariamente (ver dashboard *Leads / Kanban*).
- Se a rota lenta for consulta específica ao banco, aplicar `LIMIT`, índice ou cache. Conferir queries N+1 em repositórios GORM.
- Escalar réplicas do app ou aumentar pool MySQL como paliativo até o fix.

## Causa raiz (pós-incidente)

- Adicionar benchmark/test de latência no handler afetado.
- Considerar criar histograma específico de negócio (ex: `crm_db_query_duration_seconds`) se o ponto lento for repetido.
- Ajustar os buckets de `http_request_duration_seconds` se os buckets atuais esconderem a cauda.

## Escalation

1. Primeiros 30 min — on-call do backend.
2. Se após 30 min ainda > 2s, escalar para DBA / líder de Backend.
3. Se começar a virar timeout (5xx), trata como `HighHTTPErrorRate` — escalar como critical.
