# F07 - Funis de Vendas (Kanban)

## Objetivo
Implementar o sistema de funis de vendas em formato kanban, conectado às conversas do WhatsApp.

## Pré-requisitos
- F06 (integração WhatsApp)

## Steps

### Step 1: Domínio de funis e colunas
- [ ] criar entidade Funnel (id, tenant_id, nome, descricao, ativo, created_at)
- [ ] criar entidade Column (id, funnel_id, nome, ordem, tipo, cor, created_at)
- [ ] tipos de coluna: inicial, intermediária, qualificado, desqualificado, manual
- [ ] migrations
- [ ] testes unitários

### Step 2: Domínio de leads
- [ ] criar entidade Lead (id, tenant_id, funnel_id, column_id, contact_id, conversation_id, product_id, assigned_to, score, status, created_at, updated_at)
- [ ] migration
- [ ] testes unitários

### Step 3: Casos de uso de funis
- [ ] criar funil
- [ ] listar funis do tenant
- [ ] editar funil (nome, colunas)
- [ ] adicionar/remover/reordenar colunas
- [ ] permissionamento: configurável se só admin ou tenant pode personalizar
- [ ] testes

### Step 4: Fluxo automático de leads
- [ ] ao receber primeira mensagem no WhatsApp → criar lead na coluna inicial
- [ ] conforme lead atende steps do especialista → mover entre colunas
- [ ] se não atende step → mover para coluna configurável
- [ ] atualizar score do lead conforme qualificação
- [ ] testes

### Step 5: Movimentação manual
- [ ] caso de uso: mover lead entre colunas manualmente
- [ ] caso de uso: mover lead entre funis
- [ ] registro de histórico de movimentações
- [ ] testes

### Step 6: Interface kanban (HTMX)
- [ ] template de visualização kanban com colunas
- [ ] cards de lead com informações resumidas
- [ ] drag-and-drop para movimentação manual
- [ ] filtros por produto, responsável, score
- [ ] visualização filtrada por perfil do grupo de permissão
- [ ] seletor de funil (quando há mais de um)
- [ ] clique no lead abre detalhes + conversa do WhatsApp

## Critérios de aceite
- funis são configuráveis com múltiplas colunas
- leads entram automaticamente ao receber mensagem
- movimentação automática por steps funciona
- movimentação manual por drag-and-drop
- visualização filtrada por perfil
- kanban é intuitivo e bonito
- cobertura >= 80%
