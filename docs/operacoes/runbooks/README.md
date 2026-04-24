# Runbooks operacionais

Guias para responder aos alertas definidos em [`infra/prometheus/alerts.yml`](../../../infra/prometheus/alerts.yml).

Cada runbook segue o template:

- **Sintoma** — o que o alerta observa
- **Impacto** — efeito para usuários e tenants
- **Diagnóstico** — dashboard, queries Prometheus e logs a consultar
- **Mitigação** — ações rápidas para estancar o problema
- **Causa raiz** — análise pós-incidente
- **Escalation** — quando e para quem escalar

## Índice

| Alerta | Severidade | Runbook |
|--------|------------|---------|
| `HighHTTPErrorRate` | critical | [`http-5xx-alto.md`](http-5xx-alto.md) |
| `HighHTTPLatency` | warning | [`http-latencia-alta.md`](http-latencia-alta.md) |
| `SpecialistFailing` | critical | [`especialista-falhando.md`](especialista-falhando.md) |
| `AutomationFailing` | warning | [`automacao-falhando.md`](automacao-falhando.md) |

## Alertas planejados ainda não ativos

Dois alertas descritos no plano original da F18 (`WhatsAppDisconnected` e
`SlowDatabase`) dependem de métricas que ainda não existem no código
(`crm_whatsapp_sessions_active` como gauge, `crm_db_query_duration_seconds`
como histograma). Quando essas métricas forem instrumentadas, reabrir a
questão e adicionar as regras correspondentes em `alerts.yml` junto dos
runbooks `whatsapp-desconectado.md` e `banco-lento.md`.
