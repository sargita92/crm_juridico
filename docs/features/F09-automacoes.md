# F09 - Automações

## Objetivo
Implementar o sistema de automações configuráveis por funil e coluna, com interface simples e intuitiva.

## Pré-requisitos
- F07 (funis/kanban)

## Steps

### Step 1: Domínio de automações
- [ ] criar entidade Automation (id, tenant_id, funnel_id, column_id, tipo, config, ativo, created_at)
- [ ] tipos: exclusao_por_tempo, mover_funil, mensagem_automatica, anotacao, trocar_especialista, rate_limit, detectar_produto
- [ ] migration
- [ ] testes unitários

### Step 2: Automação - Exclusão por tempo
- [ ] caso de uso: configurar exclusão de lead após X dias/horas na coluna
- [ ] job/worker que verifica leads expirados periodicamente
- [ ] excluir ou mover para coluna de arquivo
- [ ] testes

### Step 3: Automação - Mover para outro funil
- [ ] caso de uso: configurar que ao cair na coluna X, lead vai para funil Y (coluna inicial)
- [ ] execução automática quando lead entra na coluna configurada
- [ ] testes

### Step 4: Automação - Mensagem automática
- [ ] caso de uso: configurar mensagem a ser enviada quando lead entra numa coluna
- [ ] template de mensagem com variáveis (nome, produto, etc.)
- [ ] envio via integração WhatsApp
- [ ] testes

### Step 5: Automação - Anotações
- [ ] caso de uso: registrar anotação automática ao mudar de coluna
- [ ] criar entidade Note (id, lead_id, conteudo, tipo, created_by, created_at)
- [ ] migration
- [ ] testes

### Step 5.1: Automação - Trocar especialista
- [ ] caso de uso: configurar troca de especialista quando lead entra numa coluna
- [ ] ao trocar: atualiza ConversationState.SpecialistID (F16), IA passa a usar novo especialista
- [ ] cenário: lead chega na "secretária" (especialista genérico), após qualificação muda para especialista do produto
- [ ] testes

### Step 5.2: Automação - Rate limit
- [ ] caso de uso: configurar limite de mensagens IA por período (ex: max 50 msg/dia por tenant)
- [ ] ao atingir limite: IA para de responder, log WARN, notifica responsável (F08)
- [ ] configurável por tenant e/ou por especialista
- [ ] testes

### Step 5.3: Automação - Detectar produto e redirecionar
- [ ] caso de uso: ao receber mensagem, verificar se conteúdo está linkado a algum produto
- [ ] se produto detectado e lead está em funil genérico → mover para funil do produto (via FunnelProductRouter)
- [ ] pode trocar especialista junto (combo com Step 5.1)
- [ ] testes

### Step 6: Motor de automações
- [ ] engine que escuta eventos de movimentação de lead
- [ ] verifica automações configuradas para coluna de destino
- [ ] executa automações na ordem configurada
- [ ] log de execução de automações
- [ ] testes

### Step 7: Telas (HTMX)
- [ ] aba dedicada de automações
- [ ] listagem de automações por funil
- [ ] formulário de criação/edição (simples e intuitivo)
- [ ] seleção de tipo com formulário dinâmico por tipo
- [ ] ativar/desativar automação
- [ ] permissionamento: visível apenas para quem tem permissão
- [ ] configurável se só admin ou tenant pode criar automações

## Critérios de aceite
- todos os 7 tipos de automação funcionam (exclusão, mover funil, mensagem, anotação, trocar especialista, rate limit, detectar produto)
- automações são configuráveis pela interface
- motor executa automações na movimentação de leads
- interface simples e intuitiva
- permissionamento funciona
- cobertura >= 80%
