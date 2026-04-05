# Estratégia de Testes

## Fluxo TDD (obrigatório)

1. escrever teste que falha
2. implementar o mínimo para passar
3. refatorar mantendo cobertura

## Camadas mínimas de teste

| Camada | O que testa | Exemplo |
|--------|-------------|---------|
| Unitário de domínio | Entidades, value objects, regras de negócio | Validação de tenant, cálculo de score |
| Caso de uso | Orquestração, regras de aplicação | Criar tenant, mover lead no funil |
| Integração | Repositórios, HTTP handlers, banco | CRUD real no MySQL, chamadas HTTP |

## Meta de cobertura

- **mínimo 80%** de cobertura de testes no projeto
- priorizar cobertura real de regra de negócio, não volume superficial
- quando a cobertura cair abaixo de 80%, a entrega deve voltar para correção
- medir cobertura por feature, não apenas global

## Testes de integração

- usar `testcontainers-go` para dependências externas (MySQL, Redis, etc.)
- evitar dependência de ambientes compartilhados
- garantir isolamento e repetibilidade da suíte
- cada teste cria e destrói seu próprio container/estado
- helper compartilhado para setup/teardown em `internal/shared/testhelper/`

## Boas práticas

- nomes de teste descritivos (`TestCreateTenant_WhenDuplicateDocument_ReturnsError`)
- um assert por teste quando possível
- test fixtures para dados reutilizáveis
- não mockar banco nos testes de integração (usar testcontainers)
- mockar dependências externas (WhatsApp API, gateway de pagamento) nos testes unitários
