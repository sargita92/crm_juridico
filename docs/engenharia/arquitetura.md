# Diretrizes de Arquitetura

## Camadas

```text
interfaces/http/   → handlers, rotas, middlewares (camada mais externa)
application/       → casos de uso, orquestração
domain/            → entidades, value objects, regras de negócio, interfaces de repositório
infrastructure/    → implementações de repositório, clients externos, integrações
```

## Regras

- separar domínio, aplicação, interfaces e infraestrutura
- manter regras de negócio no domínio e nos casos de uso
- manter handlers HTTP finos (recebe request, chama caso de uso, retorna response)
- definir composition root para wire-up de dependências
- isolar detalhes de ORM e framework da regra de negócio
- usar injeção de dependências por construtor
- declarar interfaces próximas de quem consome
- propagar `context.Context` nas operações relevantes
- manter `main` mínima e estável
- extrair roteamento e middlewares para módulos dedicados
- aplicar graceful shutdown em serviços long-running

## Estrutura de pastas

```text
cmd/
  api/                          # entry point da aplicação
    main.go                     # mínima: configura, wires e sobe servidor
internal/
  <feature>/                    # um módulo por feature
    domain/                     # entidades, value objects, interfaces de repo
    application/                # casos de uso
    infrastructure/             # implementação de repositórios, clients
    interfaces/http/            # handlers, rotas da feature
  shared/                       # código compartilhado entre features
    middleware/                  # middlewares transversais (auth, tenant, logging)
    config/                     # configuração (Viper)
    database/                   # conexão e helpers de banco
pkg/                            # pacotes exportáveis (se necessário)
web/
  templates/
    layouts/                    # layouts base (admin, tenant, auth)
    partials/                   # componentes reutilizáveis (header, sidebar, modals)
    <feature>/                  # templates específicos da feature
  static/
    css/                        # estilos
    js/                         # scripts (mínimo, HTMX resolve a maioria)
migrations/                     # arquivos de migration (golang-migrate)
docs/                           # documentação do projeto
```

## Regras para mini DI

- priorizar DI manual e explícita
- evitar frameworks pesados de injeção
- agrupar providers por feature
- facilitar substituição de dependências em testes
- composition root em `cmd/api/main.go` usando módulos

## Modularização do main.go

O main.go deve permanecer **estável e enxuto**. Cada feature encapsula seu próprio wire-up em um `Module` que sabe criar seus repositórios, use cases e handler.

### Interface Module

```go
// internal/shared/module/module.go

type Module interface {
    Name() string
    RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc)
}
```

### Padrão por feature

Cada feature expõe uma função `NewModule(deps)` que recebe apenas dependências externas (db, repos de outros módulos) e retorna um `Module`:

```go
// internal/document/module.go

func NewModule(db *gorm.DB) Module {
    // cria repos, use cases, handler internamente
    // retorna struct que implementa Module
}
```

### main.go enxuto

```go
modules := []module.Module{
    tenant.NewModule(db),
    specialist.NewModule(db, tenantRepo),
    document.NewModule(db, specialistRepo),
    mcp.NewModule(db, specialistRepo),
}

for _, mod := range modules {
    mod.RegisterRoutes(router, authMw, adminMw)
}
```

### Regras

- O main.go **não conhece** use cases nem repositórios individuais — só módulos
- Dependências entre módulos são explícitas via parâmetros do `NewModule`
- Adicionar nova feature = criar módulo + adicionar uma linha no main.go
- Cada módulo é responsável por seu wire-up interno

## Multitenancy

- banco de dados único com isolamento lógico
- toda entidade relevante tem `tenant_id`
- middleware extrai `tenant_id` do contexto do usuário autenticado
- queries são automaticamente filtradas por `tenant_id`
- admin pode operar sem filtro de tenant (acesso total)

## Frontend (HTMX)

- renderização server-side com Go `html/template`
- interatividade via atributos HTMX (`hx-get`, `hx-post`, `hx-swap`, `hx-trigger`, etc.)
- endpoints retornam fragmentos HTML (partials) para atualizações parciais
- evitar JavaScript customizado quando HTMX resolver
- layouts separados: admin, tenant, auth, landing page
- atualização em tempo real via SSE ou polling HTMX
