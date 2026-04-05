# F05 - Treinamento de Especialistas

## Objetivo
Implementar as funcionalidades avançadas de configuração dos especialistas: RAG, MCPs, guardrails, passo a passo e sistema de qualificação por pontuação.

## Pré-requisitos
- F04 (CRUD especialistas)

## Steps

### Step 1: RAG configurável
- [ ] criar entidade Document (id, nome, tipo, conteudo/path, created_at)
- [ ] criar entidade SpecialistDocument (specialist_id, document_id) para N:N
- [ ] migration para tabelas documents e specialist_documents
- [ ] caso de uso: upload de documento
- [ ] caso de uso: listar documentos
- [ ] caso de uso: associar documento a especialista
- [ ] caso de uso: desassociar documento
- [ ] documentos são reutilizáveis entre vários especialistas
- [ ] telas HTMX para gestão de documentos
- [ ] testes

### Step 2: MCPs
- [ ] criar entidade McpServer (id, nome, url, config, headers, created_at)
- [ ] criar entidade SpecialistMcp (specialist_id, mcp_id) para N:N
- [ ] migration
- [ ] caso de uso: CRUD de MCPs
- [ ] caso de uso: associar/desassociar MCP a especialista
- [ ] telas HTMX
- [ ] testes

### Step 3: Guardrails
- [ ] criar entidade Guardrail (id, specialist_id, tipo, regra, mensagem, ativo)
- [ ] migration
- [ ] caso de uso: CRUD de guardrails por especialista
- [ ] tipos de guardrails (ex: tópicos proibidos, limite de escopo, tom de resposta)
- [ ] telas HTMX para configurar guardrails
- [ ] testes

### Step 4: Passo a passo (script de atendimento)
- [ ] criar entidade Step (id, specialist_id, ordem, texto, tipo_dado_esperado, obrigatorio)
- [ ] migration
- [ ] caso de uso: CRUD de steps por especialista
- [ ] caso de uso: reordenar steps
- [ ] telas HTMX com drag-and-drop ou reordenação
- [ ] testes

### Step 5: Sistema de qualificação por pontuação
- [ ] adicionar campo pontuacao a entidade Step
- [ ] criar entidade ScoringConfig (specialist_id, total_pontos, threshold_qualificacao)
- [ ] migration
- [ ] caso de uso: configurar scoring de um especialista
- [ ] lógica: soma dos pontos dos steps atendidos vs threshold
- [ ] acima do threshold → qualificado / abaixo → desqualificado
- [ ] telas HTMX para configurar pontuação e threshold
- [ ] testes

## Critérios de aceite
- documentos RAG são reutilizáveis e associáveis a múltiplos especialistas
- MCPs são associáveis pela interface
- guardrails configuráveis por especialista
- passo a passo é editável e reordenável
- qualificação por pontuação funciona com thresholds configuráveis
- tudo feito 100% pela interface
- cobertura >= 80%
