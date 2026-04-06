# F15 — MCP Interno para Especialistas

## Resumo

Implementar um servidor MCP (Model Context Protocol) interno que expõe ferramentas do CRM para os especialistas (agentes IA). Isso permite que os especialistas executem ações no sistema de forma estruturada — consultar leads, atualizar status no funil, buscar dados de clientes, agendar follow-ups, etc.

## Motivação

Atualmente os especialistas são treinados com conhecimento estático. Com MCP, eles ganham a capacidade de **agir** no sistema: consultar dados em tempo real, executar operações e retornar resultados ao usuário via WhatsApp de forma autônoma e controlada.

## Escopo

### Ferramentas (tools) a expor via MCP

- **Leads**: buscar, listar, atualizar status, mover no funil
- **Contatos**: buscar dados do contato, histórico de interações
- **Funil**: listar etapas, mover lead entre etapas
- **Agenda**: criar/consultar compromissos e follow-ups
- **Produtos**: listar produtos disponíveis, consultar detalhes
- **Documentos**: listar documentos do lead (quando F14 estiver pronto)

### Requisitos

1. Servidor MCP interno (JSON-RPC sobre stdio ou HTTP/SSE)
2. Autenticação por especialista — cada especialista só acessa dados do seu tenant
3. Rate limiting por especialista
4. Logging de todas as chamadas de ferramentas (auditoria)
5. Permissões granulares — admin configura quais ferramentas cada especialista pode usar
6. Isolamento de tenant obrigatório em todas as ferramentas

### Fora de escopo (v1)

- MCP externo (expor ferramentas para clientes ou sistemas terceiros)
- Resources e Prompts do MCP (apenas Tools na v1)

## Dependências

- F05 (Treinamento de Especialistas) — especialistas precisam existir
- F07 (Funis/Kanban) — para ferramentas de funil (pode ser incremental)

## Critérios de aceite

- [ ] Servidor MCP funcional com pelo menos 3 ferramentas implementadas
- [ ] Especialista consegue usar ferramentas durante conversa no WhatsApp
- [ ] Isolamento de tenant validado com testes OWASP
- [ ] Logs de auditoria de todas as chamadas
- [ ] Admin consegue habilitar/desabilitar ferramentas por especialista
- [ ] Testes de integração cobrindo cenários de sucesso e erro
- [ ] Cobertura >= 80%
