# F16 — Motor de IA dos Especialistas

## Objetivo

Implementar a integração com provider de LLM para que os especialistas (agentes de IA) respondam automaticamente às conversas do WhatsApp, usando todo o treinamento configurado em F05 (prompt, RAG, guardrails, steps, scoring).

## Pré-requisitos

- F05 (treinamento de especialistas — concluído)
- F06 Steps 1-4 (WhatsApp recebendo/enviando mensagens)
- F07 (funis/kanban — IA interage com o kanban: move leads, atualiza score)
- F10 (produtos — IA associa lead ao produto correto pela conversa)

## Visão geral do fluxo

```text
Mensagem chega no WhatsApp
  → Identifica tenant e especialista associado
  → Identifica produto pela mensagem (palavras-chave ou especialista vinculado)
  → Cria lead no kanban (coluna inicial) se primeira mensagem
  → Monta contexto:
      • system prompt do especialista
      • documentos RAG relevantes
      • guardrails como instruções
      • step atual do script de atendimento
      • produto associado (contexto específico)
      • histórico da conversa
  → Envia para provider de IA (LLM)
  → Recebe resposta
  → Aplica guardrails de saída
  → Avalia scoring (pontuação do step)
  → Atualiza kanban: move lead entre colunas conforme progresso
  → Envia resposta via WhatsApp
  → Persiste mensagem e atualiza estado da conversa
```

## Steps

### Step 1: Interface de provider de IA (abstração)

- [ ] criar interface `AIProvider` em `internal/ai/domain/`
  - `GenerateResponse(ctx, request AIRequest) (AIResponse, error)`
  - `GenerateStreamResponse(ctx, request AIRequest) (<-chan AIChunk, error)`
- [ ] definir structs: `AIRequest` (messages, system prompt, temperature, max_tokens), `AIResponse` (content, usage, finish_reason), `AIChunk`
- [ ] definir `AIMessage` (role: system/user/assistant, content)
- [ ] testes unitários das structs e validações

### Step 2: Implementação do provider (Anthropic/Claude)

- [ ] implementar `ClaudeProvider` que satisfaz `AIProvider`
- [ ] usar Anthropic API (Messages API)
- [ ] configuração via env vars: `AI_API_KEY`, `AI_MODEL`, `AI_MAX_TOKENS`
- [ ] suporte a system prompt separado (como Anthropic API espera)
- [ ] retry com backoff exponencial para erros transientes (429, 500, 503)
- [ ] timeout configurável por request
- [ ] métricas: latência, tokens consumidos, erros por tipo
- [ ] testes com mock do HTTP client

> **Nota**: A interface `AIProvider` permite trocar para OpenAI, Groq, Ollama, etc. no futuro sem impactar o domínio. Anthropic/Claude é a implementação inicial.

### Step 3: Associação especialista ↔ produto

- [ ] criar entidade SpecialistProduct (specialist_id, product_id) para N:N
- [ ] migration
- [ ] caso de uso: associar/desassociar produto a especialista
- [ ] tela admin HTMX: vincular produtos ao especialista (similar a docs/MCPs)
- [ ] ao receber mensagem, IA identifica produto:
  - 1º: especialista vinculado a produto específico → associa direto
  - 2º: análise da mensagem pelo LLM contra produtos cadastrados no tenant
  - 3º: palavras-chave do produto (fallback sem IA)
- [ ] associar produto ao lead automaticamente
- [ ] se não identificar → lead fica sem produto (associação manual depois)
- [ ] testes

### Step 4: Montagem de contexto (Context Builder)

- [ ] criar `ContextBuilder` em `internal/ai/application/`
- [ ] buscar prompt do especialista
- [ ] buscar documentos RAG associados e montar contexto relevante
- [ ] buscar guardrails ativos e converter em instruções do system prompt
- [ ] identificar step atual do script de atendimento para a conversa
- [ ] buscar histórico de mensagens da conversa (últimas N mensagens)
- [ ] montar `AIRequest` completo com todos os componentes
- [ ] limite de tokens: truncar histórico se exceder janela de contexto
- [ ] testes unitários com cenários variados (com/sem RAG, com/sem steps, etc.)

### Step 5: Engine de conversa

- [ ] criar `ConversationEngine` em `internal/ai/application/`
- [ ] orquestrar: receber mensagem → ContextBuilder → AIProvider → resposta
- [ ] gerenciar estado da conversa:
  - step atual no script de atendimento
  - dados coletados em cada step
  - pontuação acumulada
  - produto identificado
- [ ] avançar step automaticamente quando resposta do lead satisfaz o step atual
- [ ] calcular scoring e determinar qualificação quando todos os steps forem concluídos
- [ ] registrar resposta do especialista como mensagem na conversa
- [ ] trigger de envio via WhatsApp provider
- [ ] testes de integração do fluxo completo

### Step 6: Interação com kanban

- [ ] ao receber primeira mensagem: criar lead na coluna inicial do funil
- [ ] associar produto ao lead (identificado no Step 3)
- [ ] conforme lead avança nos steps do especialista → mover entre colunas
- [ ] ao completar steps: mover para coluna "qualificado" ou "desqualificado" baseado no scoring
- [ ] step não atendido → mover para coluna configurável
- [ ] atualizar score do lead no kanban em tempo real
- [ ] registrar histórico de movimentações automáticas (origem: IA)
- [ ] testes

### Step 7: Guardrails em runtime

- [ ] validar resposta do LLM contra guardrails antes de enviar
- [ ] guardrail de tópicos proibidos: verificar se resposta contém tópicos bloqueados
- [ ] guardrail de escopo: rejeitar e regenerar se fugiu do escopo
- [ ] guardrail de tom: instruções no system prompt (enforcement via prompt engineering)
- [ ] fallback: se resposta violar guardrail, enviar mensagem padrão configurável
- [ ] log de violações de guardrails para auditoria
- [ ] testes

### Step 8: Handoff humano (takeover)

- [ ] possibilidade de humano assumir conversa manualmente
- [ ] ao assumir, especialista para de responder automaticamente
- [ ] ao devolver para IA, especialista retoma do ponto onde parou
- [ ] indicador visual na interface: "atendimento por IA" vs "atendimento manual"
- [ ] caso de uso: assumir conversa
- [ ] caso de uso: devolver conversa para IA
- [ ] testes

### Step 9: Configuração de provider por tenant (interface admin)

- [ ] tela admin para configurar provider de IA padrão
- [ ] possibilidade de tenant usar provider/modelo diferente (futuro: multi-provider)
- [ ] configuração de limites: max tokens por resposta, max mensagens por dia
- [ ] dashboard básico de uso: tokens consumidos, mensagens respondidas, qualificações
- [ ] testes

## Critérios de aceite

- [ ] especialista responde automaticamente via WhatsApp usando LLM
- [ ] contexto montado corretamente: prompt + RAG + guardrails + step atual + produto + histórico
- [ ] especialista pode ser vinculado a produtos pela interface admin
- [ ] IA identifica produto automaticamente pela conversa (ex: "revisar aposentadoria" → produto "Revisão de Aposentadoria")
- [ ] lead é criado no kanban e associado ao produto identificado
- [ ] lead se move automaticamente no kanban conforme avança nos steps
- [ ] script de steps é seguido sequencialmente com avanço automático
- [ ] scoring funciona: lead é qualificado/desqualificado ao final do script
- [ ] guardrails são aplicados em runtime (entrada e saída)
- [ ] humano pode assumir e devolver conversa
- [ ] provider abstrato via interface (trocar provider sem impactar domínio)
- [ ] métricas de IA: latência, tokens, erros
- [ ] isolamento de tenant em todas as operações
- [ ] testes OWASP: acesso não autorizado, isolamento de tenant
- [ ] cobertura >= 80%

## Decisões técnicas

| Decisão | Escolha | Justificativa |
|---------|---------|---------------|
| Provider inicial | Anthropic (Claude) | Melhor system prompt, tool use nativo, boa relação custo/qualidade |
| Protocolo | HTTP REST (Messages API) | Simples, bem documentado, SDK disponível em Go |
| RAG | Contexto no prompt (v1) | Sem vector DB na v1; documentos pequenos concatenados no system prompt |
| Streaming | Opcional (v1 sem) | WhatsApp não se beneficia de streaming; resposta completa é suficiente |
| Histórico | Últimas 20 mensagens | Limite para controlar tokens; configurável por especialista |
