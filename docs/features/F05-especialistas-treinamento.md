# F05 - Treinamento de Especialistas

## Objetivo
Implementar as funcionalidades avançadas de configuração dos especialistas: RAG, MCPs, guardrails, passo a passo e sistema de qualificação por pontuação.

## Pré-requisitos
- F04 (CRUD especialistas)

## Steps

### Step 1: RAG configurável
- [x] criar entidade Document (id, nome, tipo, conteudo/path, created_at)
- [x] criar entidade SpecialistDocument (specialist_id, document_id) para N:N
- [x] migration para tabelas documents e specialist_documents
- [x] caso de uso: upload de documento
- [x] caso de uso: listar documentos
- [x] caso de uso: associar documento a especialista
- [x] caso de uso: desassociar documento
- [x] documentos são reutilizáveis entre vários especialistas
- [x] telas HTMX para gestão de documentos
- [x] testes

### Step 2: MCPs
- [x] criar entidade McpServer (id, nome, url, config, headers, created_at)
- [x] criar entidade SpecialistMcp (specialist_id, mcp_id) para N:N
- [x] migration
- [x] caso de uso: CRUD de MCPs
- [x] caso de uso: associar/desassociar MCP a especialista
- [x] telas HTMX
- [x] testes

### Step 3: Guardrails
- [x] criar entidade Guardrail (id, specialist_id, tipo, regra, mensagem, ativo)
- [x] migration
- [x] caso de uso: CRUD de guardrails por especialista
- [x] tipos de guardrails (ex: tópicos proibidos, limite de escopo, tom de resposta)
- [x] telas HTMX para configurar guardrails
- [x] testes

### Step 4: Passo a passo (script de atendimento)
- [x] criar entidade Step (id, specialist_id, ordem, texto, tipo_dado_esperado, obrigatorio)
- [x] migration
- [x] caso de uso: CRUD de steps por especialista
- [x] caso de uso: reordenar steps
- [x] telas HTMX com reordenação (mover cima/baixo)
- [x] testes

### Step 5: Sistema de qualificação por pontuação
- [x] adicionar campo pontuacao a entidade Step
- [x] criar entidade ScoringConfig (specialist_id, total_pontos, threshold_qualificacao)
- [x] migration
- [x] caso de uso: configurar scoring de um especialista
- [x] lógica: soma dos pontos dos steps atendidos vs threshold
- [x] acima do threshold → qualificado / abaixo → desqualificado
- [x] telas HTMX para configurar pontuação e threshold
- [x] testes

## Critérios de aceite
- [x] documentos RAG são reutilizáveis e associáveis a múltiplos especialistas
- [x] MCPs são associáveis pela interface
- [x] guardrails configuráveis por especialista
- [x] passo a passo é editável e reordenável
- [x] qualificação por pontuação funciona com thresholds configuráveis
- [x] tudo feito 100% pela interface
- [x] cobertura >= 80%
