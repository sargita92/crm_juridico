# F07 - Funis de Vendas (Kanban)

## Objetivo
Implementar o sistema de funis de vendas em formato kanban, conectado às conversas do WhatsApp.

## Pré-requisitos
- F06 (integração WhatsApp)

## Status: em andamento

## Steps

### Step 1: Domínio de funis e colunas
- [x] criar entidade Funnel (id, tenant_id, nome, descricao, ativo, created_at)
- [x] criar entidade Column (id, funnel_id, nome, ordem, tipo, cor, created_at)
- [x] tipos de coluna: entry, intermediate, won, lost
- [x] migrations (000019-000022)
- [x] testes unitários

### Step 2: Domínio de leads
- [x] criar entidade Lead (id, tenant_id, funnel_id, column_id, contact_id, conversation_id, score, status, created_at, updated_at)
- [x] criar entidade LeadMovement (histórico de movimentações)
- [x] migrations
- [x] testes unitários

> **Nota**: campos `product_id` (F10) e `assigned_to` (F08) serão adicionados nas features correspondentes.

### Step 3: Casos de uso de funis
- [x] criar funil (com 5 colunas padrão automáticas)
- [x] listar funis do tenant
- [x] editar funil (nome, descrição)
- [x] ativar/desativar funil
- [x] adicionar/remover/reordenar colunas
- [ ] permissionamento: configurável se só admin ou tenant pode personalizar (F08)
- [x] testes

### Step 4: Fluxo automático de leads
- [x] ao receber primeira mensagem no WhatsApp → criar lead na coluna inicial
- [ ] conforme lead atende steps do especialista → mover entre colunas (F16)
- [ ] se não atende step → mover para coluna configurável (F16)
- [ ] atualizar score do lead conforme qualificação (F16)
- [x] testes (para o que foi implementado)

### Step 5: Movimentação manual
- [x] caso de uso: mover lead entre colunas manualmente
- [x] caso de uso: mover lead entre funis
- [x] registro de histórico de movimentações
- [x] testes

### Step 6: Interface kanban (HTMX)
- [x] template de visualização kanban com colunas
- [x] cards de lead com informações resumidas
- [x] drag-and-drop para movimentação manual (Sortable.js)
- [ ] filtros por produto, responsável, score (F08, F10)
- [ ] visualização filtrada por perfil do grupo de permissão (F08)
- [x] seletor de funil (quando há mais de um)
- [ ] clique no lead abre painel de detalhes (Step 7)

### Step 7: Painel de detalhes do lead (futuro)
- [ ] ao clicar no card, abrir painel lateral ou modal com:
  - [ ] dados do contato (nome, telefone)
  - [ ] funil e coluna atual, score
  - [ ] conversa do WhatsApp embutida (últimas mensagens + link para abrir)
  - [ ] histórico de movimentações no funil
  - [ ] documentos/arquivos do lead (F14)
  - [ ] produto associado (F10)
  - [ ] responsável atribuído (F08)
  - [ ] anotações manuais
  - [ ] botão "Mover Lead" (funil + coluna)
  - [ ] botão "Abrir Conversa" (navega para WhatsApp)
- [ ] design: painel lateral estilo drawer (abre da direita, não bloqueia o kanban)

## Decisões técnicas
- Módulo `internal/funnel/` com DDD + Clean Architecture
- Interface `LeadCreator` integra WhatsApp → Funnel (cria lead ao receber primeira msg)
- Colunas padrão criadas automaticamente ao criar funil (Novo, Em Atendimento, Qualificado, Ganho, Perdido)
- Drag-and-drop via Sortable.js (lib leve, sem framework)
- Sessão WhatsApp persistida em SQLite (volume Docker)
- Templates inline no kanban (evita problema de sub-templates Go)

## Critérios de aceite
- [x] funis são configuráveis com múltiplas colunas
- [x] leads entram automaticamente ao receber mensagem
- [ ] movimentação automática por steps funciona (F16)
- [x] movimentação manual por drag-and-drop
- [ ] visualização filtrada por perfil (F08)
- [x] kanban é intuitivo e bonito
- [x] cobertura >= 80%

## Melhorias futuras
- Painel de detalhes do lead (Step 7) — ver dados, conversa, documentos, histórico
- Filtros avançados por produto (F10), responsável (F08), score
- Visualização por perfil de permissão (F08)
- Movimentação automática por IA (F16)
- Anotações manuais no lead
- Indicador de tempo na coluna nos cards
