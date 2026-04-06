# F06 - Integração com WhatsApp

## Objetivo
Conectar o sistema ao WhatsApp para receber e enviar mensagens, criando a base para o fluxo de leads.

## Pré-requisitos
- F02 (autenticação e multitenancy)

## Steps

### Step 1: Domínio de conversas
- [ ] criar entidade Contact (id, tenant_id, nome, telefone, whatsapp_id, created_at)
- [ ] criar entidade Conversation (id, tenant_id, contact_id, status, last_message_at, created_at)
- [ ] criar entidade Message (id, conversation_id, direcao, conteudo, tipo, timestamp)
- [ ] migrations
- [ ] testes unitários do domínio

### Step 2: Integração com WhatsApp via whatsmeow
- [ ] definir interface de provider WhatsApp (abstração para trocar implementação no futuro)
- [ ] implementar provider com `whatsmeow` (desenvolvimento/testes)
- [ ] gerenciar sessão do WhatsApp (login via QR code, persistência de sessão)
- [ ] caso de uso: receber mensagem (event handler do whatsmeow)
- [ ] caso de uso: enviar mensagem
- [ ] caso de uso: criar/atualizar contato ao receber mensagem
- [ ] testes com mocks do provider

> **Nota**: whatsmeow é a implementação inicial para reduzir custos de desenvolvimento.
> A interface de provider permite trocar para WhatsApp Business API (Meta) no futuro sem impactar o domínio.

### Step 3: Processamento de mensagens
- [ ] handler interno que processa eventos do whatsmeow
- [ ] processamento ass��ncrono de mensagens recebidas (fila interna ou goroutines)
- [ ] criação automática de contato e conversa ao receber primeira mensagem
- [ ] instrumentação: métricas de mensagens recebidas/enviadas, latência de processamento
- [ ] testes de integração

### Step 4: Interface de conversas (HTMX)
- [ ] template de lista de conversas (sidebar estilo WhatsApp Web)
- [ ] template de detalhe da conversa (mensagens)
- [ ] campo de envio de mensagem
- [ ] atualização em tempo real via SSE ou polling com HTMX
- [ ] busca de conversas
- [ ] indicador de novas mensagens

### Step 5: Conexão com especialista

> **Movido para F16 (Motor de IA dos Especialistas)**. Este step foi extraído para uma feature dedicada que cobre: integração com provider de IA, montagem de contexto (prompt + RAG + guardrails + steps + histórico), engine de conversa, scoring, guardrails em runtime e handoff humano. Ver [F16](F16-motor-ia-especialistas.md).

## Critérios de aceite
- mensagens do WhatsApp chegam e são persistidas
- mensagens podem ser enviadas pela interface
- conversas são isoladas por tenant
- interface similar ao WhatsApp Web
- especialista atende automaticamente
- possível assumir conversa manualmente
- cobertura >= 80%
