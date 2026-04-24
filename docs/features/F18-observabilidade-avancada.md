# F18 - Observabilidade Avançada

**Status:** concluído (2026-04-24) — branch `feature/F18-observabilidade-avancada`, 22 commits de feature.

## Objetivo
Elevar a observabilidade do CRM ao nível production-ready, complementando a cobertura mínima entregue em F08/F09. Adicionar spans em camada de aplicação, histogramas de duração, dashboards Grafana atualizados e regras de alerta.

## Pré-requisitos
- F08 (usuários e permissões) — métricas básicas entregues
- F09 (automações) — métricas básicas entregues

## Motivação
A varredura transversal em F08/F09 entregou o mínimo: OTel span por handler HTTP + uma métrica de negócio chave por módulo (`automation_executions_total`, `permission_changes_total`, `invites_total`). Esta feature fecha o gap para ter visibilidade completa em produção.

## Steps

### Step 1: Spans na camada de aplicação
- [x] span em cada use case dos módulos críticos (automation, permission, auth, notification, funnel, whatsapp)
- [x] propagação de trace_id nos logs Zap
- [x] padrão: `<module>.usecase.<name>` (ex.: `automation.usecase.trigger`)
- [x] testes verificam presença do span no context propagado

### Step 2: Histogramas de duração de negócio
- [x] `automation_execution_duration_seconds{type,outcome}` — duração dos 6 executors
- [x] `permission_check_duration_seconds` — latência do middleware RequirePermission
- [x] `specialist_response_duration_seconds` (se ainda faltante)
- [x] buckets customizados por operação (não usar `DefBuckets` cegamente)

### Step 3: Métricas complementares
- [x] `invites_total{outcome=sent|accepted|expired|revoked}` — `expired` ainda não existe, fechar o ciclo de vida
- [x] adicionar scope `load_balance` a `crm_permission_changes_total` — uso: incrementar em `auth/application/manage_load_balance.go::SetByGroup` (uso cross-module adiado no escopo mínimo)
- [x] `load_balance_fallback_total{reason}` — quando o load balance cai para outro algoritmo
- [x] `notification_read_total{type}` — para medir engajamento
- [x] `automation_rate_limited_total{type}` — quantas automações foram bloqueadas pelo rate limiter

### Step 4: Dashboards Grafana
- [x] dashboard "Overview": requests/s, latência p50/p95/p99, taxa de erro 5xx
- [x] dashboard "WhatsApp": mensagens in/out, sessões ativas, erros de envio
- [x] dashboard "Leads/Kanban": leads criados, movimentações, automações executadas
- [x] dashboard "Especialistas": respostas, latência, taxa de erro
- [x] dashboard "Equipe": convites, permissões alteradas, load balance
- [x] dashboards versionados em `ops/grafana/dashboards/` (JSON)

### Step 5: Regras de alerta (Prometheus Alertmanager)
- [x] alta taxa de erro HTTP: > 5% de 5xx em 5 min
- [x] latência alta: p95 > 2s em 5 min
- [x] WhatsApp desconectado: sessão inativa por > 2 min
- [x] banco lento: p95 query > 500ms
- [x] especialista falhando: taxa de erro > 10%
- [x] automação falhando: taxa de erro > 10% em 5 min
- [x] regras em `ops/prometheus/alerts.yml`

### Step 6: Documentação operacional
- [x] runbook por alerta (o que fazer quando dispara)
- [x] guia de troubleshooting com queries úteis
- [x] atualizar `docs/engenharia/observabilidade.md` com o estado final

## Decisões tomadas
- Exporter de trace: Grafana Tempo via OTLP gRPC (fallback stdout quando `OTEL_EXPORTER_OTLP_ENDPOINT` vazio).
- Retenção: 15d métricas (Prometheus), 7d traces (Tempo).
- Alertmanager dev: receiver `null`; Slack/email via env vars em prod.
- Self-hosted via docker-compose em dev; decisão de hosting em prod permanece no roadmap de infra (fora do escopo de F18).

## Escopo não atendido (gaps conhecidos, follow-up separado)
- `WhatsAppDisconnected` e `SlowDatabase`: alertas descartados na Task 21 porque as métricas-fonte (`crm_whatsapp_sessions_active`, `crm_db_query_duration_seconds`) não existem hoje. Ticket futuro para adicionar gauge de sessão WhatsApp e histograma de latência Gorm.
- `automation_rate_limited_total`: counter registrado mas não instrumentado (não há rate limiter de automação hoje). Pronto para uso quando o rate limiter for introduzido.
- Validação ponta-a-ponta do stack Tempo/Alertmanager no compose é smoke manual; foi feita validação local de `promtool test rules` (passa) e de build/testes unitários (passam).

## Critérios de aceite
- todos os use cases críticos têm span
- dashboards cobrem os 5 perfis de uso principais
- alertas dispararam em staging pelo menos uma vez (validação)
- runbooks escritos para cada alerta
- cobertura >= 80% nos novos testes de instrumentação
