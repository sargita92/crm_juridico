# Princípios de Engenharia

## Fundamentos

- aplicar DDD (Domain-Driven Design)
- aplicar Clean Architecture
- trabalhar com TDD (Test-Driven Development)
- organizar código por feature
- usar mini DI para simplificar gestão de dependências
- manter foco em system design
- usar padrões de projeto com parcimônia (ex.: `Factory`)

## Detalhamento

### DDD

- modelar o software a partir do domínio do negócio
- linguagem ubíqua entre código e regras de negócio
- entidades, value objects, aggregates e domain services no lugar correto
- separar domínio de infraestrutura

### Clean Architecture

- separar camadas: domínio, aplicação, interfaces e infraestrutura
- dependências apontam sempre para dentro (domínio no centro)
- regras de negócio nunca dependem de framework, banco ou HTTP

### TDD

1. escrever teste que falha
2. implementar o mínimo para passar
3. refatorar mantendo cobertura

### Organização por feature

- cada feature é um módulo isolado dentro de `internal/`
- cada módulo tem suas próprias camadas (domain, application, infrastructure, interfaces)
- compartilhamento entre features via `internal/shared/`

### Mini DI

- priorizar DI manual e explícita
- evitar frameworks pesados de injeção
- agrupar providers por feature
- facilitar substituição de dependências em testes
- usar injeção de dependências por construtor
- definir composition root para wire-up de dependências

### Padrões de projeto

- usar com parcimônia e propósito claro
- Factory para criação complexa de objetos
- Repository para abstração de persistência
- não adicionar padrões especulativamente
