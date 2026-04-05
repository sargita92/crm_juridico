# F08 - Usuários e Permissões do Tenant

## Objetivo
Implementar o sistema de usuários, grupos de permissão, perfis de visualização e load balance de leads.

## Pré-requisitos
- F07 (funis/kanban)

## Steps

### Step 1: Grupos de permissão
- [ ] criar entidade PermissionGroup (id, tenant_id, nome, descricao, permissoes, created_at)
- [ ] criar entidade UserGroup (user_id, group_id) para N:N
- [ ] migration
- [ ] caso de uso: CRUD de grupos de permissão
- [ ] permissões: acesso a automações, configurações, personalização de funis, etc.
- [ ] permissionamento: configurável se só admin ou tenant pode criar grupos
- [ ] testes

### Step 2: Perfis de visualização
- [ ] criar entidade ViewProfile (id, group_id, funnel_id, colunas_visiveis)
- [ ] migration
- [ ] caso de uso: configurar perfil de visualização para um grupo
- [ ] caso de uso: aplicar filtro de colunas no kanban baseado no perfil
- [ ] testes

### Step 3: Associação grupo ↔ funil
- [ ] grupos podem ser responsáveis por uma área do funil ou todo funil
- [ ] caso de uso: associar grupo a funil/colunas específicas
- [ ] testes

### Step 4: Load balance de leads
- [ ] caso de uso: configurar balanceamento para um grupo
- [ ] ao entrar lead no funil → distribuir automaticamente entre membros do grupo responsável
- [ ] algoritmos: round-robin, menor carga, aleatório (configurável)
- [ ] caso de uso: reatribuir lead manualmente
- [ ] testes

### Step 5: Gestão de usuários do tenant
- [ ] caso de uso: convidar usuário ao tenant
- [ ] caso de uso: listar usuários do tenant
- [ ] caso de uso: editar permissões do usuário
- [ ] caso de uso: remover usuário do tenant
- [ ] testes

### Step 6: Telas (HTMX)
- [ ] template de gestão de usuários
- [ ] template de gestão de grupos de permissão
- [ ] template de configuração de perfis de visualização
- [ ] template de configuração de load balance
- [ ] interações via HTMX

## Critérios de aceite
- grupos de permissão funcionam e controlam acesso
- perfis filtram colunas visíveis no kanban
- load balance distribui leads automaticamente
- owner tem permissão máxima no tenant
- cobertura >= 80%
