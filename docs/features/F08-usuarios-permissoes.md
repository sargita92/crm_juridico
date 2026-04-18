# F08 - Usuários e Permissões do Tenant

## Objetivo
Implementar o sistema de usuários, grupos de permissão, perfis de visualização e load balance de leads.

## Pré-requisitos
- F07 (funis/kanban)

## Steps

### Step 1: Grupos de permissão
- [x] criar entidade PermissionGroup (id, tenant_id, nome, descricao, permissoes, created_at)
- [x] criar entidade UserGroup (user_id, group_id) para N:N
- [x] migration
- [x] caso de uso: CRUD de grupos de permissão
- [x] permissões: acesso a automações, configurações, personalização de funis, etc.
- [x] permissionamento: configurável se só admin ou tenant pode criar grupos
- [x] testes

### Step 2: Perfis de visualização
- [x] criar entidade ViewProfile (id, group_id, funnel_id, colunas_visiveis)
- [x] migration
- [x] caso de uso: configurar perfil de visualização para um grupo
- [x] caso de uso: aplicar filtro de colunas no kanban baseado no perfil
- [x] testes

### Step 3: Associação grupo ↔ funil
- [x] grupos podem ser responsáveis por uma área do funil ou todo funil
- [x] caso de uso: associar grupo a funil/colunas específicas
- [x] testes

### Step 4: Load balance de leads
- [x] caso de uso: configurar balanceamento para um grupo
- [x] ao entrar lead no funil → distribuir automaticamente entre membros do grupo responsável
- [x] algoritmos: round-robin, menor carga, aleatório (configurável)
- [x] caso de uso: reatribuir lead manualmente
- [x] testes

### Step 4.1: Responsável do lead e notificações
- [x] todo lead deve ter um humano responsável associado (campo responsible_user_id no Lead)
- [x] ao criar lead (via IA/WhatsApp ou manual), atribuir responsável via load balance ou manualmente
- [x] notificar responsável quando:
  - lead é criado e atribuído a ele
  - lead é movido de coluna (pela IA ou manualmente)
  - IA faz handoff para humano
  - lead é qualificado/desqualificado pelo motor de IA (F16)
- [x] canal de notificação: na interface (badge/toast) + WhatsApp (opcional, configurável)
- [x] testes

### Step 5: Gestão de usuários do tenant
- [x] caso de uso: convidar usuário ao tenant
- [x] caso de uso: listar usuários do tenant
- [x] caso de uso: editar permissões do usuário
- [x] caso de uso: remover usuário do tenant
- [x] testes

### Step 6: Telas (HTMX)
- [x] template de gestão de usuários
- [x] template de gestão de grupos de permissão
- [x] template de configuração de perfis de visualização
- [x] template de configuração de load balance
- [x] interações via HTMX

## Critérios de aceite
- grupos de permissão funcionam e controlam acesso
- perfis filtram colunas visíveis no kanban
- load balance distribui leads automaticamente
- todo lead tem um responsável humano associado
- responsável é notificado em eventos relevantes (criação, movimentação, handoff, qualificação)
- owner tem permissão máxima no tenant
- cobertura >= 80%
