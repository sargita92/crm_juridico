# F18 - Observabilidade Avançada

## Objetivo
Elevar a observabilidade do CRM ao nível production-ready, complementando a cobertura mínima entregue em F08/F09. Adicionar spans em camada de aplicação, histogramas de duração, dashboards Grafana atualizados e regras de alerta.

## Pré-requisitos
- F08 (usuários e permissões) — métricas básicas entregues
- F09 (automações) — métricas básicas entregues

## Motivação
A varredura transversal em F08/F09 entregou o mínimo: OTel span por handler HTTP + uma métrica de negócio chave por módulo (`automation_executions_total`, `permission_changes_total`, `invites_total`). Esta feature fecha o gap para ter visibilidade completa em produção.

## Steps

### Step 1: Spans na camada de aplicação
- [ ] span em cada use case dos módulos críticos (automation, permission, auth, notification, funnel, whatsapp)
- [ ] propagação de trace_id nos logs Zap
- [ ] padrão: `<module>.usecase.<name>` (ex.: `automation.usecase.trigger`)
- [ ] testes verificam presença do span no context propagado

### Step 2: Histogramas de duração de negócio
- [ ] `automation_execution_duration_seconds{type,outcome}` — duração dos 6 executors
- [ ] `permission_check_duration_seconds` — latência do middleware RequirePermission
- [ ] `specialist_response_duration_seconds` (se ainda faltante)
- [ ] buckets customizados por operação (não usar `DefBuckets` cegamente)

### Step 3: Métricas complementares
- [ ] `invites_total{outcome=sent|accepted|expired|revoked}`
- [ ] `load_balance_fallback_total{reason}` — quando o load balance cai para outro algoritmo
- [ ] `notification_read_total{type}` — para medir engajamento
- [ ] `automation_rate_limited_total{type}` — quantas automações foram bloqueadas pelo rate limiter

### Step 4: Dashboards Grafana
- [ ] dashboard "Overview": requests/s, latência p50/p95/p99, taxa de erro 5xx
- [ ] dashboard "WhatsApp": mensagens in/out, sessões ativas, erros de envio
- [ ] dashboard "Leads/Kanban": leads criados, movimentações, automações executadas
- [ ] dashboard "Especialistas": respostas, latência, taxa de erro
- [ ] dashboard "Equipe": convites, permissões alteradas, load balance
- [ ] dashboards versionados em `ops/grafana/dashboards/` (JSON)

### Step 5: Regras de alerta (Prometheus Alertmanager)
- [ ] alta taxa de erro HTTP: > 5% de 5xx em 5 min
- [ ] latência alta: p95 > 2s em 5 min
- [ ] WhatsApp desconectado: sessão inativa por > 2 min
- [ ] banco lento: p95 query > 500ms
- [ ] especialista falhando: taxa de erro > 10%
- [ ] automação falhando: taxa de erro > 10% em 5 min
- [ ] regras em `ops/prometheus/alerts.yml`

### Step 6: Documentação operacional
- [ ] runbook por alerta (o que fazer quando dispara)
- [ ] guia de troubleshooting com queries úteis
- [ ] atualizar `docs/engenharia/observabilidade.md` com o estado final

## Decisões técnicas pendentes
- Onde hospedar Grafana/Prometheus em produção (self-hosted vs Grafana Cloud)
- Exporter de trace em produção (Jaeger, Tempo, Honeycomb)
- Retenção de métricas e traces

## Critérios de aceite
- todos os use cases críticos têm span
- dashboards cobrem os 5 perfis de uso principais
- alertas dispararam em staging pelo menos uma vez (validação)
- runbooks escritos para cada alerta
- cobertura >= 80% nos novos testes de instrumentação
