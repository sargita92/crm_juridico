---
feature: F26 — Gargalo intermitente de banco
agent: Arquiteto (investigação)
version: 1
created_at: 2026-05-26
reason: Registro da Fase 1 (root cause investigation) do debugging sistemático
---

# F26 — Investigação Fase 1 (causa-raiz)

> Método: `superpowers:systematic-debugging`. **Iron Law**: nenhuma correção sem
> investigação de causa-raiz primeiro. Esta é a fase de coleta de evidência.

## Sintoma

Latência intermitente da aplicação chegando a ~19s, atribuída a gargalo de banco.
Sem reprodução confiável. Suspeita inicial: AI Playground (F17).

## Evidência coletada (análise estática)

### Configuração do banco / pool

- Pool: `MaxOpenConns=25`, `MaxIdleConns=10`, **sem** `ConnMaxLifetime`/
  `ConnMaxIdleTime` — `internal/shared/database/database.go:27-28`.
- Logger do Gorm: **`Silent`** — `database.go:16`. **Não há log de slow query.**
- `/metrics` (Prometheus) existe — `cmd/api/main.go:559` — mas **nenhuma métrica
  do pool** (`sql.DBStats`: `InUse`, `WaitCount`, `WaitDuration`).
- **Sem pprof** registrado.
- **Sem tracing por query** (o plugin OTel do Gorm não está instalado).

### Caminho do suspeito (AI Playground / inbox WhatsApp)

- O Playground usa **polling HTMX** e cada ação chama `findContact` →
  `ListByTenant` (até 200 conversas) — `internal/ai/interfaces/http/playground/handler.go`.
- `ListByTenant` delega para `ConversationRepository.FindByTenantID` — a **mesma
  query do inbox real do WhatsApp** (caminho quente) —
  `internal/ai/infrastructure/playground_adapters.go:30`.
- `FindByTenantID` (`internal/whatsapp/infrastructure/gorm_conversation_repository.go:68`)
  faz **3 queries** (count, lista paginada com JOIN em contacts, e "última
  mensagem por conversa" via subquery `MAX(timestamp) GROUP BY conversation_id`).
  **Não é N+1.**
- O `HandleSendAsLead` do Playground é **assíncrono**: enfileira no `Debouncer`,
  que dispara em goroutine própria (`time.AfterFunc`) — não bloqueia o request.

### Caminho do motor de IA

- `ConversationEngine.HandleMessages`
  (`internal/ai/application/conversation_engine.go:133`) **não** envolve a chamada
  ao LLM em uma transação de banco: os repositórios usam `ctx` e a conexão é
  liberada entre chamadas. Logo, **não** há o padrão "transação/conexão presa
  durante a chamada lenta ao LLM" dentro do motor.

### Mudança recente (Phase 1 — "check recent changes")

- O bug foi reportado em 2026-05-25, **mesmo dia** do merge da F23 (cross-sell).
- Migration `000062` trocou `UNIQUE (tenant_id, contact_id)` por índice
  **não-único nas mesmas colunas** em `leads`. → Sem mudança estrutural de plano
  de query; só removeu a unicidade (permite múltiplos leads por contato).

## Hipóteses

| # | Hipótese | Status |
|---|----------|--------|
| H1 | Índice faltando na query de lista de conversas (inbox/playground) | **DESCARTADA** — `idx_messages_timestamp (conversation_id, timestamp)` e `idx_conversations_last_message (tenant_id, last_message_at DESC)` existem (migrations 000016/000017). |
| H2 | Transação/conexão presa durante chamada ao LLM no motor de IA | **DESCARTADA** — motor solta conexão entre repos. |
| H3 | Regressão de plano por causa da migration 000062 | **IMPROVÁVEL** — índice permaneceu nas mesmas colunas. |
| H4 | **Exaustão do pool (25 conns)** sob concorrência (ex.: polling do Playground/inbox + escritas do motor) → requests bloqueiam aguardando conexão | **EM ABERTO** — só confirmável com `WaitCount`/`WaitDuration` do pool. |
| H5 | Query lenta pontual (ainda não identificada: dashboards, agregações, full scan) segurando conexões | **EM ABERTO** — só confirmável com slow-query log + tracing por query. |
| H6 | Gargalo fora do banco (CPU, GC, goroutine leak) percebido como lentidão | **EM ABERTO** — só confirmável com pprof/runtime metrics. |

## Conclusão da Fase 1

A análise estática **descartou o suspeito óbvio** (índice faltando) e revelou que
o sistema **não tem observabilidade de banco** para localizar um stall
intermitente. Confirmar a causa-raiz (H4/H5/H6) exige **evidência de runtime**.

**Próximo passo:** instrumentar (esta entrega) e então observar/reproduzir para
fechar o diagnóstico — ver [`plan-v1.md`](plan-v1.md).
