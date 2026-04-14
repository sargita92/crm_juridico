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

### Testes end-to-end (build tag `integration`)

Testes que sobem o pipeline inteiro (HTTP → router → use case → DB real → fixture completa) são marcados com:

```go
//go:build integration
```

Rodam apenas quando o tag é passado explicitamente:

```bash
go test -tags=integration ./internal/ai/application/... -run TestE2E -v -timeout 5m
```

Não entram no `go test ./...` default — evita lentidão na suíte unitária e torna explícito quando estamos exercitando o fluxo completo. Use este padrão para qualquer cenário que precise de:

- fixture carregada (`fixture/fixtures.sql`)
- múltiplos módulos wired via `NewModule`
- asserções sobre efeitos em cadeia (state → lead → coluna)

Exemplo: `internal/ai/application/e2e_flow_test.go` exercita o fluxo "mensagem inbound → especialista → steps → qualificação" usando o `FakeProvider` registrado no `ProviderRegistry`.

## Testes de segurança (OWASP)

Cada feature deve incluir testes automatizados que cubram os riscos relevantes do OWASP Top 10:

| OWASP | O que testar | Exemplo |
|-------|-------------|---------|
| A01 - Broken Access Control | endpoint sem token retorna 401, usuário comum não acessa admin (403), isolamento de tenant | `TestListTenants_WithoutToken_Returns401` |
| A02 - Cryptographic Failures | senha hasheada no banco, JWT não expõe dados sensíveis, segredos fora do código | `TestUserPassword_IsHashed_InDatabase` |
| A03 - Injection | SQL injection em parâmetros de busca, XSS em campos de texto renderizados em templates | `TestSearchConversations_SQLInjection_ReturnsEmpty` |
| A04 - Insecure Design | validação de payload (campos obrigatórios, tipos, limites), rate limiting em login | `TestCreateTenant_EmptyPayload_Returns400` |
| A05 - Security Misconfiguration | stack trace não exposto ao usuário, headers de segurança presentes | `TestErrorResponse_DoesNotLeakStackTrace` |
| A07 - Auth Failures | token expirado rejeitado, JWT manipulado rejeitado, brute force mitigado | `TestExpiredToken_Returns401` |
| A08 - Integrity Failures | JWT com alg:none rejeitado, payload adulterado rejeitado | `TestManipulatedJWT_Returns401` |
| A09 - Logging Failures | login falho gera log de auditoria, ações sensíveis são registradas | `TestFailedLogin_GeneratesAuditLog` |

### Regras

- todo endpoint HTTP deve ter pelo menos um teste de acesso não autorizado (401/403)
- todo endpoint de tenant deve ter teste de isolamento (não acessa dados de outro tenant)
- todo campo de entrada renderizado em template deve ter teste anti-XSS
- todo campo usado em query deve ter teste anti-SQL injection
- testes OWASP são obrigatórios na camada de integração

### Testes manuais

- arquivos `.http` para testes manuais em `rest/`
- `rest/99-seguranca-owasp.http` contém cenários de teste manual para cada categoria OWASP
- usar com JetBrains HTTP Client ou compatível

## Boas práticas

- nomes de teste descritivos (`TestCreateTenant_WhenDuplicateDocument_ReturnsError`)
- um assert por teste quando possível
- test fixtures para dados reutilizáveis
- não mockar banco nos testes de integração (usar testcontainers)
- mockar dependências externas (WhatsApp API, gateway de pagamento) nos testes unitários
