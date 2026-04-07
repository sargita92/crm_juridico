# F09 - Automações

## Objetivo
Implementar o sistema de automações configuráveis por funil e coluna, com interface simples e intuitiva.

## Pré-requisitos
- F07 (funis/kanban)

## Steps

### Step 1: Domínio de automações
- [x] criar entidade Automation (id, tenant_id, funnel_id, column_id, tipo, config, ativo, created_at)
- [x] tipos: exclusao_por_tempo, mover_funil, mensagem_automatica, anotacao, trocar_especialista, rate_limit, detectar_produto
- [x] migration
- [x] testes unitários

### Step 2: Automação - Exclusão por tempo
- [x] caso de uso: configurar exclusão de lead após X dias/horas na coluna
- [x] job/worker que verifica leads expirados periodicamente
- [x] excluir ou mover para coluna de arquivo
- [x] testes

### Step 3: Automação - Mover para outro funil
- [x] caso de uso: configurar que ao cair na coluna X, lead vai para funil Y (coluna inicial)
- [x] execução automática quando lead entra na coluna configurada
- [x] testes

### Step 4: Automação - Mensagem automática
- [x] caso de uso: configurar mensagem a ser enviada quando lead entra numa coluna
- [x] template de mensagem com variáveis (nome, produto, etc.)
- [x] envio via integração WhatsApp
- [x] testes

### Step 5: Automação - Anotações
- [x] caso de uso: registrar anotação automática ao mudar de coluna
- [x] criar entidade Note (id, lead_id, conteudo, tipo, created_by, created_at)
- [x] migration
- [x] testes

### Step 5.1: Automação - Trocar especialista
- [x] caso de uso: configurar troca de especialista quando lead entra numa coluna
- [x] ao trocar: atualiza ConversationState.SpecialistID (F16), IA passa a usar novo especialista
- [x] cenário: lead chega na "secretária" (especialista genérico), após qualificação muda para especialista do produto
- [x] testes

### Step 5.2: Automação - Rate limit
- [x] caso de uso: configurar limite de mensagens IA por período (ex: max 50 msg/dia por tenant)
- [x] ao atingir limite: IA para de responder, log WARN, notifica responsável (F08)
- [x] configurável por tenant e/ou por especialista
- [x] testes

### Step 5.3: Automação - Detectar produto e redirecionar
- [x] caso de uso: ao receber mensagem, verificar se conteúdo está linkado a algum produto
- [x] se produto detectado e lead está em funil genérico → mover para funil do produto (via FunnelProductRouter)
- [x] pode trocar especialista junto (combo com Step 5.1)
- [x] testes

### Step 6: Motor de automações
- [x] engine que escuta eventos de movimentação de lead
- [x] verifica automações configuradas para coluna de destino
- [x] executa automações na ordem configurada
- [x] log de execução de automações
- [x] testes

### Step 7: Telas (HTMX)
- [x] aba dedicada de automações
- [x] listagem de automações por funil
- [x] formulário de criação/edição (simples e intuitivo)
- [x] seleção de tipo com formulário dinâmico por tipo
- [x] ativar/desativar automação
- [x] permissionamento: visível apenas para quem tem permissão
- [ ] configurável se só admin ou tenant pode criar automações

## Critérios de aceite
- todos os 7 tipos de automação funcionam (exclusão, mover funil, mensagem, anotação, trocar especialista, rate limit, detectar produto)
- automações são configuráveis pela interface
- motor executa automações na movimentação de leads
- interface simples e intuitiva
- permissionamento funciona
- cobertura >= 80%
