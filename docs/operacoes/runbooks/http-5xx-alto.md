# HighHTTPErrorRate

**Severidade**: critical
**Fonte**: [`infra/prometheus/alerts.yml`](../../../infra/prometheus/alerts.yml) — grupo `crm_http`

## Sintoma

Proporção de requests HTTP com `status=5xx` maior que 5% da taxa total, sustentada por 5 minutos.

```promql
sum(rate(http_requests_total{status=~"5.."}[5m]))
  /
sum(rate(http_requests_total[5m])) > 0.05
```

## Impacto

- Usuários recebem erro 500/502/503/504 em fluxos de CRM (lead, kanban, automações, especialistas).
- Webhooks da Meta / whatsmeow podem reprocessar mensagens quando o handler devolve 5xx.
- Dashboards e contadores de tenant ficam inconsistentes enquanto o erro persiste.

## Diagnóstico

1. **Dashboard**: abrir **CRM Jurídico — Overview** (uid `crm-overview`).
   - Painel *Error Rate 5xx (%)* confirma a magnitude.
   - Painel *Requests per Second by endpoint* revela qual rota concentra o erro.
   - Painel *Top 5 endpoints by latency p95* ajuda a cruzar com latência.

2. **Queries úteis** (Prometheus):

   ```promql
   # Top 10 endpoints mais afetados
   topk(10, sum by (method, path) (rate(http_requests_total{status=~"5.."}[5m])))

   # Distribuição por código exato
   sum by (status) (rate(http_requests_total{status=~"5.."}[5m]))
   ```

3. **Logs** (Zap estruturado): filtrar `status=5` no log do Gin:

   ```bash
   # dentro do container app
   grep '"status":5' /var/log/app.log | tail -100
   ```

   Em produção, usar o backend de logs (Loki / ELK) com o filtro `status=~"5.."` e agrupar por `request_id` para rastrear.

4. **Traces**: selecionar um `trace_id` a partir do log (campo injetado pelo middleware `observability.LoggerFromContext`) e abrir no Grafana Tempo para ver onde a chamada falhou (banco, whatsmeow, IA).

## Mitigação

- Se um único endpoint concentra o erro: considerar deploy rollback ou toggle de feature flag.
- Se o culpado é dependência externa (WhatsApp, Meta, MySQL): verificar conectividade e health de cada dependência; se confirmado, circuit-break / desabilitar a integração afetada.
- Se o erro é 5xx global: checar uso de CPU/memória do app e do MySQL; escalar réplicas ou aumentar limites temporariamente.

## Causa raiz (pós-incidente)

- Registrar o PR/commit culpado no incidente.
- Avaliar se faltou cobertura de teste (unitário/integração) ou alerta mais sensível.
- Se envolveu dependência externa, abrir ação para adicionar timeout / retry / bulkhead em `internal/<feature>/infrastructure/`.

## Escalation

1. Primeiros 15 min — on-call do backend.
2. Se persistir após mitigação inicial, acionar líder de Backend.
3. Impacto multi-tenant ou perda de dados: acionar Segurança e Produto.
