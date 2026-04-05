# Área Administrativa

## Visão geral

A área administrativa é acessível apenas por administradores do sistema. Centraliza a gestão de tenants, especialistas de IA, pagamentos, bloqueios e configurações.

---

## Tenants

### Descrição
Cadastro e gestão de clientes do sistema. Cada tenant representa uma organização (PF ou PJ) que utiliza o CRM.

### Regras
- multitenancy com banco de dados único (isolamento lógico)
- tenant pode ser PF ou PJ
- CRUD completo (criar, listar, editar, desativar)

### Bloqueio/Desbloqueio
- admin pode bloquear um tenant de usar o sistema
- motivo de bloqueio é obrigatório e registrado
- admin pode desbloquear com motivo registrado
- tenant bloqueado não acessa a plataforma

### Pagamentos
- controle de pagamento por tenant
- pode ser automático (integração com gateway) ou manual (registrado por fora)
- histórico de pagamentos visível no admin

---

## Especialistas (Agentes de IA)

### Descrição
Agentes de IA chamados "especialistas" que atendem leads via WhatsApp. Todo treinamento é feito pela interface, sem alterar código.

### CRUD
- criar, editar, listar e excluir especialistas
- associar especialista a um ou mais tenants

### Configurações do especialista

#### Prompt editável
- campo de texto para definir o comportamento do especialista
- editável pela interface

#### RAG configurável
- associar documentos ao especialista
- documentos de RAG são reutilizáveis entre vários especialistas
- upload e gestão de documentos pela interface

#### MCPs
- associar especialista a MCPs (Model Context Protocol servers)
- configurável pela interface

#### Guardrails
- regras e limites configuráveis para controlar o comportamento
- evitar que o especialista fuja do escopo definido

#### Passo a passo (script de atendimento)
- fluxo sequencial que o especialista segue
- ex: pedir nome → pedir documento X → pedir documento Y → até completar
- cada step é configurável (texto, tipo de dado esperado, obrigatoriedade)

### Sistema de qualificação por pontuação
- cada item/documento/etapa tem uma pontuação atribuída
- total de pontos é configurável por especialista
- threshold configurável:
  - abaixo do threshold → lead qualificado
  - acima ou igual ao threshold → lead desqualificado
- pontuações e thresholds editáveis pela interface

---

## Associação especialistas ↔ tenants

- um especialista pode ser associado a vários tenants
- um tenant pode ter vários especialistas
- associação gerenciada pelo admin

---

## Configuração dos tenants

- perfis de permissão
- grupos de permissão
- definições de acesso
- configurável se só admin cria grupos ou se o tenant também pode

---

## Tela de seleção de tenant

- admin tem acesso a todos os tenants e pode navegar pela visão de qualquer um
- usuários com mais de um tenant veem tela de seleção
- se o usuário tem apenas um tenant, vai direto (sem tela de seleção)

---

## Logs

- área centralizada de logs do sistema
- visível apenas para admin

---

## Fluxo resumido

```text
1. Criar tenant
2. Criar especialista(s)
3. Associar especialista(s) ao tenant
4. Configurar permissões e perfis do tenant
5. Gerenciar pagamentos
6. Gerenciar bloqueios (quando necessário)
```
