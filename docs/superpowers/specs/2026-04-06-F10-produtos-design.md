# F10 — Produtos: Design Spec

## Resumo

Cadastro de produtos por tenant com detecção automática por palavras-chave, associação manual/automática a leads, vínculo N:N entre produto e funil com prioridade para roteamento de entrada, e preparação para detecção por número de WhatsApp (Meta futura).

## Decisões de design

### Relação Funil <-> Produto (N:N com prioridade)

- Um produto pode estar vinculado a múltiplos funis (ex: "Venda" e "Pós-venda" do mesmo produto)
- Cada vínculo tem uma **prioridade** (int, maior número = maior prioridade)
- Ao criar lead, o funil de **maior prioridade** para aquele produto é escolhido
- Funis sem produtos vinculados (ou com "todos") funcionam como **catch-all** para leads sem produto detectado

### Cascata de detecção de produto

1. **Número de WhatsApp** — se o número de entrada tem produto vinculado, usa esse (futuro, quando Meta multi-número)
2. **Palavras-chave** — analisa conteúdo da primeira mensagem contra keywords dos produtos do tenant (case-insensitive, match parcial por palavra)
3. **Catch-all** — sem match, lead fica sem produto (`product_id = NULL`), entra no funil padrão (`is_default`)

Na implementação atual (whatsmeow, 1 número por tenant), apenas os passos 2 e 3 são funcionais. O modelo de dados já suporta o passo 1 via tabela `product_phone_numbers`.

### Roteamento de entrada do lead

```
Mensagem chega no WhatsApp
  -> detectar produto (cascata acima)
  -> se produto detectado:
       -> buscar funnel_products WHERE product_id = X ORDER BY priority DESC
       -> lead entra no primeiro funil encontrado (maior prioridade), coluna entry
  -> se produto não detectado:
       -> lead entra no funil padrão do tenant (is_default = true)
```

### Convenção de prioridade

`priority` usa escala onde **maior número = maior prioridade** (10 vence sobre 5). Query: `ORDER BY priority DESC`.

## Modelo de dados

### Tabela `products`

| Campo | Tipo | Notas |
|-------|------|-------|
| id | VARCHAR(36) PK | UUID |
| tenant_id | VARCHAR(36) FK | NOT NULL, referencia tenants |
| name | VARCHAR(255) | NOT NULL |
| description | TEXT | Opcional |
| keywords | JSON | Array de strings, ex: `["trabalhista","CLT","rescisão"]` |
| active | BOOLEAN | Default true |
| created_at | DATETIME | |
| updated_at | DATETIME | |

Indexes: `(tenant_id)`, `(tenant_id, active)`.

### Tabela `funnel_products`

| Campo | Tipo | Notas |
|-------|------|-------|
| id | VARCHAR(36) PK | UUID |
| funnel_id | VARCHAR(36) FK | NOT NULL, referencia funnels |
| product_id | VARCHAR(36) FK | NOT NULL, referencia products |
| priority | INT | NOT NULL, default 1, maior = mais prioritário |
| created_at | DATETIME | |

Unique constraint: `(funnel_id, product_id)`.
Index: `(product_id, priority DESC)` para roteamento.

### Tabela `product_phone_numbers`

| Campo | Tipo | Notas |
|-------|------|-------|
| id | VARCHAR(36) PK | UUID |
| product_id | VARCHAR(36) FK | NOT NULL |
| phone_number | VARCHAR(20) | NOT NULL |
| created_at | DATETIME | |

Unique constraint: `(phone_number)` — um número pertence a no máximo 1 produto.
Nota: sem uso funcional até integração Meta. Apenas modelo e migration.

### Alteração em `leads`

Adicionar coluna `product_id VARCHAR(36) NULL` com FK para `products(id)`.
Index: `(product_id)`.

## Arquitetura do módulo

### Estrutura de pastas

```
internal/product/
  domain/
    product.go              — entidade Product (name max 255, keywords []string)
    funnel_product.go       — entidade FunnelProduct (priority int)
    phone_number.go         — entidade ProductPhoneNumber
    repository.go           — interfaces: ProductRepository, FunnelProductRepository, PhoneNumberRepository
    errors.go               — erros de domínio
  application/
    create_product.go       — criar produto com validações
    update_product.go       — editar nome, descrição, keywords, active
    list_products.go        — listar produtos ativos do tenant
    delete_product.go       — desativar produto (soft delete)
    detect_product.go       — lógica de detecção: keywords match na mensagem
    manage_funnel_products.go — vincular/desvincular produto<->funil, setar prioridade
  infrastructure/
    gorm_product_repository.go
    models.go               — modelos GORM com mappers to/from domain
  interfaces/http/
    handler.go              — handlers HTMX (listagem, form, associação)
    routes.go               — rotas registradas via RegisterRoutes
  module.go                 — NewModule(db, log) + RegisterRoutes
```

### Cross-module interfaces

O módulo `funnel` consome produto via duas interfaces:

```go
// ProductDetector — detecta produto a partir do conteúdo da mensagem
type ProductDetector interface {
    DetectFromMessage(ctx context.Context, tenantID, messageText string) (productID string, found bool, err error)
}

// ProductProvider — busca dados de produto para exibição
type ProductProvider interface {
    GetByID(ctx context.Context, id string) (name string, err error)
}
```

O módulo `product` consome dados de funil via interface:

```go
// FunnelLister — lista funis do tenant (para UI de vínculo)
type FunnelLister interface {
    ListByTenantID(ctx context.Context, tenantID string) ([]FunnelInfo, error)
}
```

### Roteamento no create_lead

O `create_lead.go` do módulo funnel será alterado para:

1. Chamar `ProductDetector.DetectFromMessage(tenantID, messageText)`
2. Se `found=true`: buscar funil via `FunnelProductRepository.FindTopPriorityFunnel(productID)` e setar `lead.ProductID`
3. Se `found=false`: manter comportamento atual (funil padrão)

### Registro no main.go

```go
productMod := product.NewModule(db, log)
// Passa detector e provider para o módulo funnel
funnelMod := funnel.NewModule(db, contactAdapter, messageAdapter, userNameAdapter, productDetector, productProvider, log)
```

## Interface (HTMX)

### 1. Listagem de produtos (`/products`)

- Tabela com colunas: Nome, Palavras-chave (tags), Funis vinculados (com prioridade), Status, Ações
- Busca por nome
- Botão "+ Novo Produto"
- Ações: Editar, Ativar/Desativar
- Produtos inativos aparecem com opacidade reduzida

### 2. Formulário de criação/edição (`/products/new`, `/products/:id/edit`)

- Campos: nome, descrição, palavras-chave (input de tags com Enter para adicionar)
- Seção "Vincular a funis": lista de funis vinculados com campo de prioridade, botão adicionar, botão remover
- Botões: Salvar, Cancelar

### 3. Kanban — filtro por produto

- Dropdown "Filtrar por produto" no toolbar do kanban (acima das colunas)
- Opções: "Todos os produtos", lista de produtos ativos do tenant
- Badge de produto no card do lead (tag colorida, ou "Sem produto" em cinza)
- Filtro via HTMX: `hx-get` com query param `product_id`

### 4. Drawer do lead — produto associado

- Nova seção "Produto" no drawer de detalhes do lead
- Mostra badge do produto atual (ou "Sem produto")
- Botão "Alterar" que abre seletor dropdown para associação manual
- Ao alterar, `hx-put` para `/leads/:id/product`

## Endpoints

### CRUD de Produtos (HTMX — retornam HTML)
- `GET /products` — página de listagem
- `GET /products/new` — formulário de criação
- `POST /products` — criar produto
- `GET /products/:id/edit` — formulário de edição
- `PUT /products/:id` — atualizar produto
- `PUT /products/:id/toggle` — ativar/desativar

### Vínculo funil<->produto (HTMX)
- `POST /products/:id/funnels` — vincular funil ao produto
- `DELETE /products/:id/funnels/:funnelId` — desvincular
- `PUT /products/:id/funnels/:funnelId/priority` — alterar prioridade

### Associação lead<->produto (HTMX)
- `PUT /leads/:id/product` — associar/alterar produto do lead (retorna partial do drawer)

### Filtro no kanban
- `GET /funnels/:id/kanban?product_id=xxx` — kanban filtrado por produto (já existente, adicionar query param)

## Testes

- Testes unitários para todas as entidades de domínio e validações
- Testes unitários para todos os use cases (mock de repositórios)
- Testes de integração com testcontainers (repositórios GORM)
- Testes OWASP: acesso não autorizado (401/403), isolamento de tenant, anti-injection
- Testes do fluxo de detecção automática (keywords match, sem match, case insensitive)
- Testes do roteamento: produto detectado -> funil correto por prioridade
- Meta: cobertura >= 80%

## Fora de escopo

- Detecção por número de WhatsApp (modelo pronto, lógica em F10 futura ou Meta)
- Especialista "secretária" como fallback (F16 — motor de IA)
- Automação de mover lead entre funis (F09 — automações)
- Permissionamento de quem pode gerenciar produtos (F08)
