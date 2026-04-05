# Feature em Andamento

> Este documento deve ser atualizado ao iniciar uma feature e limpo ao concluí-la.
> Serve como ponto central de contexto para todos os agentes durante o desenvolvimento.

---

## Feature

- **Código**: F02
- **Nome**: Autenticação e Multitenancy
- **Branch**: feature/F02-autenticacao-multitenancy
- **Referência**: docs/features/F02-autenticacao-multitenancy.md

## Stories (definidas pelo PO)

### Story 1: Cadastro de Tenants no sistema
- **Objetivo de negócio**: O sistema precisa conhecer os escritórios/clientes (tenants) para isolar seus dados
- **Escopo**: Entidade tenant com nome, tipo (PF/PJ), documento, status e motivo de bloqueio; persistência no banco
- **Fora do escopo**: CRUD completo via interface (isso é F03), associação com especialistas
- **Critérios de aceite**:
  - [ ] Tenant pode ser criado com todos os campos obrigatórios
  - [ ] Tenant pode ser consultado por ID
  - [ ] Tenants inativos/bloqueados são identificáveis pelo status
  - [ ] Dados persistem corretamente no banco

### Story 2: Cadastro de Usuários com vínculo a tenant
- **Objetivo de negócio**: Cada pessoa que acessa o sistema deve ter um usuário vinculado a pelo menos um tenant
- **Escopo**: Entidade usuário com nome, email, senha (hash), tenant, role e status; persistência no banco
- **Fora do escopo**: Tela de cadastro de usuários, recuperação de senha, convite por email
- **Critérios de aceite**:
  - [ ] Usuário pode ser criado com email e senha
  - [ ] Senha é armazenada como hash (nunca em texto plano)
  - [ ] Usuário está vinculado a um tenant
  - [ ] Não é possível criar dois usuários com o mesmo email

### Story 3: Login com email e senha
- **Objetivo de negócio**: O usuário precisa se autenticar para acessar o sistema de forma segura
- **Escopo**: Endpoint de login, validação de credenciais, geração de JWT, tela de login
- **Fora do escopo**: Login social, 2FA, "esqueci minha senha"
- **Critérios de aceite**:
  - [ ] Usuário faz login com email e senha válidos e recebe um token
  - [ ] Credenciais inválidas retornam erro claro (sem revelar qual campo está errado)
  - [ ] Token tem tempo de expiração
  - [ ] Endpoints protegidos rejeitam requisições sem token válido

### Story 4: Isolamento de dados por tenant
- **Objetivo de negócio**: Cada escritório/cliente deve ver apenas seus próprios dados — segurança e privacidade são fundamentais
- **Escopo**: Middleware que extrai tenant do usuário autenticado, scoping automático de queries por tenant
- **Fora do escopo**: Permissões granulares dentro do tenant (isso é F08)
- **Critérios de aceite**:
  - [ ] Todas as consultas são filtradas automaticamente pelo tenant do usuário
  - [ ] Um usuário do tenant A não consegue acessar dados do tenant B
  - [ ] Admin consegue acessar dados de qualquer tenant
  - [ ] O isolamento funciona sem que o desenvolvedor precise filtrar manualmente em cada query

### Story 5: Seleção de tenant após login
- **Objetivo de negócio**: Usuários com acesso a mais de um escritório precisam escolher em qual contexto vão trabalhar
- **Escopo**: Listagem de tenants do usuário, tela de seleção, redirecionamento automático quando há só um tenant
- **Fora do escopo**: Trocar de tenant sem fazer logout
- **Critérios de aceite**:
  - [ ] Usuário com 1 tenant é redirecionado direto ao dashboard
  - [ ] Usuário com mais de 1 tenant vê a tela de seleção
  - [ ] Admin vê todos os tenants do sistema
  - [ ] Após selecionar, o tenant fica no contexto da sessão

## Wireframes e fluxos (definidos pelo UI/UX)

<!-- Pendente -->

## Design técnico (definido pelo Arquiteto)

<!-- Pendente -->

## Cenários de teste (definidos pelo QA)

<!-- Pendente -->

## Status

| Etapa | Status | Observações |
|-------|--------|-------------|
| PO — stories | ✅ concluído | 5 stories definidas |
| UI/UX — wireframes | pendente | |
| Arquiteto — design | pendente | |
| QA — cenários | pendente | |
| Dev Backend | pendente | |
| Dev Front-end | pendente | |
| QA — validação | pendente | |
| Segurança — revisão | pendente | |

## Blockers

Nenhum no momento.
