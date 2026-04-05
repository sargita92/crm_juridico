# Área do Tenant

## Visão geral

A área do tenant é o espaço onde os usuários de cada organização operam o CRM no dia a dia. Contém as abas de WhatsApp, Leads/Kanban, Automações e Configurações.

---

## Aba WhatsApp

### Descrição
Interface de conversas similar ao WhatsApp Web.

### Funcionalidades
- lista todas as conversas do tenant
- exibe todas as pessoas que entraram em contato
- usabilidade familiar (parecida com WhatsApp Web)
- conversa vinculada ao lead no kanban

---

## Aba Leads / Funis de Vendas (Kanban)

### Descrição
Funil de vendas visual em formato kanban, ligado diretamente às conversas do WhatsApp.

### Regras
- pode ter mais de um funil por tenant
- personalização do funil (colunas, regras) é configurável:
  - somente admin pode alterar, ou
  - o tenant também pode alterar (configurável por tenant)

### Fluxo do lead no kanban

```text
1. Pessoa chama no WhatsApp
   → entra automaticamente na coluna inicial do funil (lead)

2. Conforme atende aos steps definidos
   → move automaticamente pelas colunas

3. Se não atende a um step
   → vai para uma coluna específica (configurável)

4. Em alguns momentos o lead fica parado
   → movido manualmente por uma pessoa

5. Automações podem mover leads entre colunas
```

### Visualização por perfil
- perfis de visualização filtram/simplificam as colunas exibidas
- evita sobrecarga visual quando há muitas colunas
- associados aos grupos de permissão

---

## Aba Automações (visível com permissão)

### Descrição
Aba dedicada para configurar automações do tenant. Visível apenas para usuários com permissão.

### Regras
- personalizáveis por funil e/ou coluna
- permissionamento: configurável se só admin cria ou se o tenant também pode
- requisito: simples e intuitivo de configurar pela interface

### Tipos de automações

| Tipo | Descrição |
|------|-----------|
| Exclusão por tempo | Excluir lead após período configurável |
| Mover para outro funil | Ao cair numa coluna, lead é enviado para outro funil |
| Envio de mensagens automáticas | Disparo de mensagens ao atingir condições |
| Anotações | Registros automáticos ao mudar de coluna |

---

## Configurações da conta (visível com permissão)

- ajustes gerais do tenant
- visível apenas para usuários com permissão adequada
