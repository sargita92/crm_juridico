# F17 — AI Playground (Desenvolvimento)

## O que é?

O AI Playground é um ambiente **isolado de desenvolvimento** para testar o pipeline de processamento de mensagens sem usar WhatsApp real. Permite simular mensagens de leads e ver como o AI responde.

## Pré-requisitos

- ✅ `AI_PLAYGROUND_ENABLED=true` no `.env`
- ✅ Usuário logado em um tenant
- ✅ Pelo menos um contato com conversa aberta

## Como usar

### 1. Acessar o Playground

```
http://localhost:8080/tenant/ai/playground
```

Você verá:
- **Sidebar esquerda**: Lista de contatos que já têm conversas
- **Chat area (direita)**: Histórico de mensagens do contato selecionado

### 2. Selecionar um contato

Clique em qualquer contato na lista. O histórico de mensagens carregará.

> **Nota**: Só contatos com conversas abertas aparecem na lista. Veja [Criar lead fake](#criar-lead-fake) se quiser adicionar novos contatos.

### 3. Enviar uma mensagem

1. Digite a mensagem no input inferior
2. Clique no botão **Enviar** (ícone de seta)
3. Aguarde **2-3 segundos** — a mensagem será processada e a resposta aparecerá

#### ⚠️ Por que não vejo a mensagem imediatamente?

O playground usa **polling a cada 2 segundos** para atualizar o histórico. Fluxo:
1. Você envia: `POST /conversation/{id}/send` → retorna `204 No Content`
2. Backend injeta a mensagem no pipeline real
3. AI processa e responde
4. A cada 2s: `GET /conversation/{id}` atualiza a tela

**Resultado**: Você vê a mensagem aparecer em ~2 segundos.

### 4. Resetar a conversa

Clique no botão **Reset** no canto superior direito. Isso:
- Zera todo o state de conversação
- Remove o lead do funnel (volta à coluna inicial)
- Limpa as mensagens

## Criar lead fake

Para adicionar um novo contato ao playground:

```bash
./scripts/create-playground-lead.sh "João Silva" "11987654321"
```

Isso cria:
- ✅ Novo contato
- ✅ Conversa aberta
- ✅ Pronto para usar no playground

**Exemplo com múltiplos leads:**

```bash
./scripts/create-playground-lead.sh "Maria Santos" "11912345678"
./scripts/create-playground-lead.sh "Pedro Costa" "11998765432"
./scripts/create-playground-lead.sh "Ana Oliveira" "11914141414"
```

## Contato de teste padrão

O fixture já cria um contato de teste:

- **Nome**: Teste Playground
- **Telefone**: 5511900000000
- **ID**: `550e8400-e29b-41d4-a716-4466554400fe`

## Fluxo de teste recomendado

1. **Criar um lead novo**:
   ```bash
   ./scripts/create-playground-lead.sh "Teste AI" "11999888777"
   ```

2. **Ir ao playground**:
   - http://localhost:8080/tenant/ai/playground

3. **Enviar uma pergunta**:
   - _"Preciso de informações sobre aposentadoria"_

4. **Observar a resposta**:
   - O AI responde em poucos segundos
   - Veja logs: `docker logs crm-api`

5. **Testar fluxos avançados**:
   - Envie `/reset` como mensagem para resetar via comando
   - Mude de especialista no admin
   - Mude de funnel/coluna e veja a automação executar

## Troubleshooting

### "Nenhum contato disponível"

Você não tem contatos com conversas. Solução:
```bash
./scripts/create-playground-lead.sh "Seu Nome" "11999999999"
```

### Mensagem não aparece após 5 segundos

1. Verifique os logs:
   ```bash
   docker logs crm-api | grep playground
   ```

2. Verifique se o tenant está correto (da URL)

3. Confirme que o contato tem uma conversa associada

### Form de envio não responde

1. Abra o console do navegador (`F12 → Console`)
2. Procure por erros do HTMX
3. Verifique se o `/send` endpoint retorna 204

## Detalhe técnico: Por que 204?

O endpoint de `/send` retorna `204 No Content` porque:
- A mensagem é injetada no pipeline real
- O resultado (resposta do AI) será apanhado pelo próximo polling
- Não precisa de resposta imediata

Isso mantém o playground isolado do WhatsApp real e permite testar automações e scripts sem complexidade extra.

## Próximos passos

- [Sandbox de especialistas](../features/F16-specialist-sandbox.md) — para testar MCP
- [Observabilidade](../engenharia/observabilidade.md) — para monitorar o AI Playground
