# AutomationFailing

**Severidade**: warning
**Fonte**: [`infra/prometheus/alerts.yml`](../../../infra/prometheus/alerts.yml) — grupo `crm_automation`

## Sintoma

Proporção de execuções de automação (`crm_automation_execution_duration_seconds_count`) com `outcome="error"` maior que 10% nos últimos 5 minutos.

```promql
sum(rate(crm_automation_execution_duration_seconds_count{outcome="error"}[5m]))
  /
sum(rate(crm_automation_execution_duration_seconds_count[5m])) > 0.10
```

Métrica instrumentada em [`internal/automation/infrastructure/metrics.go`](../../../internal/automation/infrastructure/metrics.go); os 6 executores vivem em `internal/automation/infrastructure/executor_*.go`.

## Impacto

- Automações do kanban (mover lead, criar nota, trocar especialista, detectar produto, enviar mensagem, expirar) param de executar.
- Funil do lead congela; SLA interno de resposta pode não ser cumprido.
- Fluxos dependentes (envio de WhatsApp via `auto_message`) podem empilhar mensagens pendentes ou duplicar quando o retry for acionado.

## Diagnóstico

1. **Dashboard**: **CRM Jurídico — Leads / Kanban** (uid `crm-leads-kanban`).
   - Painel *Automações executadas/s (por tipo)* — identifica qual executor sofre.
   - Painel *Taxa de erro de automações (%)* — confirma.
   - Painel *Automações rate-limited/s* — descartar falso positivo quando o problema é na verdade rate-limit.
   - Painel *Latência p95 de automações* — cauda longa por executor.

2. **Queries úteis**:

   ```promql
   # Erro por tipo (identifica o executor problemático)
   sum by (type) (rate(crm_automation_execution_duration_seconds_count{outcome="error"}[5m]))

   # Taxa de erro por tipo
   sum by (type) (rate(crm_automation_execution_duration_seconds_count{outcome="error"}[5m]))
     /
   sum by (type) (rate(crm_automation_execution_duration_seconds_count[5m]))

   # Rate-limiting ativo
   sum by (type) (rate(crm_automation_rate_limited_total[5m]))
   ```

3. **Logs** ([`internal/automation/`](../../../internal/automation/)):
   - Filtrar por `automation_type` ou `executor` e abrir o trace via `trace_id` para ver onde o executor falha.
   - Executor `auto_message` depende de whatsmeow — se for ele, cruzar com dashboard *WhatsApp*.
   - Executor `switch_specialist` depende de IA — cruzar com dashboard *Especialistas* e o alerta `SpecialistFailing`.
   - Executor `detect_product` pode falhar por regex/config do tenant.

4. **Domínio**: consultar `automation_executions` no banco (MySQL) agrupando por `status=error` e `type` nas últimas horas para ver padrões.

## Mitigação

- Se um único `type` concentra o erro: desabilitar a regra do tenant correspondente via admin até o fix.
- Se executor `auto_message`: verificar se o tenant está com WhatsApp conectado; revisar logs de whatsmeow.
- Se executor `switch_specialist` coincide com `SpecialistFailing`: priorizar aquele runbook primeiro.
- Se o problema é sobrecarga (muitas execuções): considerar reduzir polling do orquestrador ou aumentar cooldown.

## Causa raiz (pós-incidente)

- Registrar `type`, tenants envolvidos e último deploy que tocou `internal/automation/`.
- Adicionar teste de integração cobrindo o caso (mock de whatsmeow / IA).
- Avaliar se o alerta de 10% é adequado: para tráfego baixo pode gerar ruído; pode ser movido para ser condicional a um volume mínimo.

## Escalation

1. Primeiros 30 min — on-call do backend.
2. Se envolve `auto_message` e WhatsApp, escalar para o responsável pela integração whatsmeow.
3. Se for regressão de domínio, abrir hotfix seguindo DoD normal (branch + PR).
