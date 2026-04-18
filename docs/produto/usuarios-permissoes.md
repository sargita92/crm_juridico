# Usuários e Permissões

## Visão geral

Cada tenant pode ter vários usuários com diferentes níveis de acesso. O sistema de permissões é baseado em grupos com perfis de visualização.

---

## Usuários

- cada tenant pode ter vários usuários
- cada usuário está ligado a permissões (por grupo ou individual)
- **owner**: permissão máxima dentro do tenant
- um usuário pode estar em mais de um tenant

---

## Grupos de permissão

### Descrição
Agrupam usuários com responsabilidades similares dentro do tenant.

### Regras
- grupos podem ser responsáveis por uma área do funil ou pelo funil inteiro
- configurável se só admin do sistema cria grupos de permissão ou se o tenant também pode
- permissões controlam acesso a:
  - aba de automações
  - configurações da conta
  - personalização de funis
  - criação de automações

### Perfis de visualização
- associados aos grupos
- filtram/simplificam as colunas exibidas para o usuário
- objetivo: evitar sobrecarga visual quando há muitas colunas no kanban
- cada grupo pode ter um perfil de visualização diferente

---

## Load balance de leads

### Descrição
Distribuição automática de leads entre membros de um grupo.

### Regras
- opção de balancear leads entre membros de um grupo
- ex: grupo "atendimento" → divide automaticamente os leads entre as pessoas
- distribuição configurável por grupo
- objetivo: distribuir carga de trabalho de forma equilibrada

---

## Hierarquia de permissões

```text
Admin do sistema (acesso total)
  └── Owner do tenant (permissão máxima no tenant)
       └── Grupos de permissão (configuráveis)
            └── Usuários individuais
```

---

## Onde encontrar na interface

Aba **Equipe** no sidebar do tenant (`/tenant/team`). Só aparece para quem
tem `users:read` ou `groups:manage`.

- **Aba Usuários**: lista de membros + convites pendentes, modal de convite
  com link copiável, modal para editar permissões individuais (somadas às
  herdadas do grupo), modal de WhatsApp ID, botão de remover
- **Aba Grupos**: lista de grupos + criação; abrir um grupo leva ao detail
  com 5 seções — Membros, Permissões (matriz recurso × ação), Funis
  atribuídos, Perfis de visualização e Load Balance
