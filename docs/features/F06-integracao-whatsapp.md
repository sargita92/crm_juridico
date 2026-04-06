# F06 - Integração com WhatsApp

## Objetivo
Conectar o sistema ao WhatsApp para receber e enviar mensagens, criando a base para o fluxo de leads.

## Pré-requisitos
- F02 (autenticação e multitenancy)

## Status: concluído

## Steps

### Step 1: Domínio de conversas
- [x] criar entidade Contact (id, tenant_id, nome, telefone, whatsapp_id, created_at)
- [x] criar entidade Conversation (id, tenant_id, contact_id, status, last_message_at, unread_count, created_at)
- [x] criar entidade Message (id, conversation_id, direcao, conteudo, tipo, status, whatsapp_msg_id, timestamp)
- [x] migrations (000015-000018: contacts, conversations, messages, whatsapp_sessions)
- [x] testes unitários do domínio (28 testes, 100% cobertura)

### Step 2: Integração com WhatsApp via whatsmeow
- [x] definir interface de provider WhatsApp (abstração para trocar implementação no futuro)
- [x] implementar provider com `whatsmeow` (desenvolvimento/testes)
- [x] gerenciar sessão do WhatsApp (login via QR code, persistência de sessão em SQLite)
- [x] caso de uso: receber mensagem (event handler do whatsmeow)
- [x] caso de uso: enviar mensagem
- [x] caso de uso: criar/atualizar contato ao receber mensagem
- [x] testes com mocks do provider

> **Nota**: whatsmeow é a implementação inicial para reduzir custos de desenvolvimento.
> A interface de provider permite trocar para WhatsApp Business API (Meta) no futuro sem impactar o domínio.
> Sessões são persistidas em SQLite por tenant (volume Docker `whatsmeow-data`).

### Step 3: Processamento de mensagens
- [x] handler interno que processa eventos do whatsmeow
- [x] processamento de mensagens recebidas via goroutine do provider
- [x] criação automática de contato e conversa ao receber primeira mensagem
- [x] deduplicação de mensagens via UNIQUE constraint em whatsapp_msg_id
- [x] instrumentação: métricas de mensagens recebidas/enviadas, latência de processamento
- [x] testes de integração (27 testes com testcontainers)

### Step 4: Interface de conversas (HTMX)
- [x] layout do tenant com sidebar própria (WhatsApp, Leads)
- [x] template de lista de conversas (sidebar estilo WhatsApp Web)
- [x] template de detalhe da conversa (mensagens em balões)
- [x] campo de envio de mensagem
- [x] atualização via SSE + polling fallback (5s)
- [x] busca de conversas por nome do contato
- [x] indicador de novas mensagens (badge unread_count)
- [x] tela de conexão QR code (gerado server-side via go-qrcode)
- [x] redirect automático após pareamento

### Step 5: Conexão com especialista

> **Movido para F16 (Motor de IA dos Especialistas)**. Este step foi extraído para uma feature dedicada que cobre: integração com provider de IA, montagem de contexto (prompt + RAG + guardrails + steps + histórico), engine de conversa, scoring, guardrails em runtime e handoff humano. Ver [F16](F16-motor-ia-especialistas.md).

## Decisões técnicas

### Arquitetura
- Módulo `internal/whatsapp/` com DDD + Clean Architecture (domain → infrastructure → application → interfaces/http)
- Interface `WhatsAppProvider` abstrai whatsmeow (substituível por Meta Business API no futuro)
- `EventBus` in-memory para SSE (pub/sub por tenant)
- Module interface refatorada para `Middlewares{Auth, Tenant, Admin}` (impacto em todos os módulos)

### Persistência de sessão
- whatsmeow usa SQLite (único dialeto suportado além de Postgres)
- Um arquivo `.db` por tenant em `storage/whatsmeow/`
- Volume Docker `whatsmeow-data` persiste entre restarts
- Reconexão automática quando sessão existe no SQLite

### Tempo real
- SSE (`/tenant/whatsapp/events`) envia event names (`new-message`, `conversation-update`)
- HTMX usa event names como `hx-trigger` para buscar dados atualizados
- Polling fallback a cada 5s para resiliência

### QR Code
- Gerado server-side via `go-qrcode` (PNG)
- Conexão em goroutine background (nunca bloqueia HTTP handler)
- `connectState` per-tenant rastreia QR codes e status
- `GetQRChannel` do whatsmeow (padrão recomendado) antes de `Connect`

### Segurança
- `IsConnected` verifica `client.IsConnected() && client.Store.ID != nil` (pareado, não apenas conectado ao servidor)
- Tenant ID extraído do JWT (não do request)
- 23 testes OWASP (401, 403, tenant isolation, SQL injection, XSS)

## Métricas Prometheus
- `whatsapp_messages_received_total{tenant_id}`
- `whatsapp_messages_sent_total{tenant_id, status}`
- `whatsapp_message_processing_duration_seconds{direction}`

## Artefatos
- `docs/artefatos/F06-integracao-whatsapp/po-stories/v1.md` — 8 stories
- `docs/artefatos/F06-integracao-whatsapp/uiux-wireframes/v1.md` — wireframes WhatsApp Web
- `docs/artefatos/F06-integracao-whatsapp/arquiteto-design/v1.md` — design técnico + 10 steps
- `docs/artefatos/F06-integracao-whatsapp/qa-cenarios/v1.md` — 83 cenários de teste
- `docs/artefatos/F06-integracao-whatsapp/qa-validacao/v1.md` — validação pós-implementação
- `docs/artefatos/F06-integracao-whatsapp/seguranca-review/v1.md` — review de segurança

## Critérios de aceite
- [x] mensagens do WhatsApp chegam e são persistidas
- [x] mensagens podem ser enviadas pela interface
- [x] conversas são isoladas por tenant
- [x] interface similar ao WhatsApp Web
- [ ] especialista atende automaticamente (movido para F16)
- [ ] possível assumir conversa manualmente (movido para F16)
- [x] cobertura >= 80% (domain 100%, application 86%)
