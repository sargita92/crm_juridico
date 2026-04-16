# Internal Tool Registry para Especialistas IA

**Data**: 2026-04-16
**Status**: Aprovado
**Abordagem**: B — Tool Registry interno customizado

## Contexto

Os especialistas IA do CRM conversam com leads via WhatsApp usando o `ConversationEngine`. Hoje, o especialista apenas gera texto. Este design adiciona a capacidade dos especialistas executarem **ações** durante a conversa: consultar dados, agir no CRM e disparar automacoes.

## Decisoes

| Decisao | Escolha | Motivo |
|---|---|---|
| Consumidor | Especialistas IA internos (via ConversationEngine) | Nao ha necessidade de exposicao externa |
| Categorias de tools | Consulta de dados, acoes no CRM, automacoes | Cobrem os casos de uso identificados |
| Configuracao | Por especialista, via admin | Mesmo padrao de documentos e MCPs |
| Execucao | Hibrida (function calling nativo + steps restringem/forcam) | Flexibilidade com controle |
| Provedor | Agnostico (OpenAI, Claude, Gemini, qualquer um) | Futuro-proof sem over-engineering |
| Protocolo | Tool Registry Go interno (sem MCP Protocol) | Zero overhead in-process, fit com arquitetura DDD |

## 1. Domain Layer — Novos Tipos

### ToolDefinition

```go
// internal/ai/domain/tool.go

type ToolDefinition struct {
    Name        string
    Description string
    Parameters  map[string]ParameterDef
    Category    ToolCategory
}

type ParameterDef struct {
    Type        string   // "string", "number", "boolean"
    Description string
    Required    bool
    Enum        []string // valores possiveis (opcional)
}

type ToolCategory string

const (
    ToolCategoryDataQuery  ToolCategory = "data_query"
    ToolCategoryCRMAction  ToolCategory = "crm_action"
    ToolCategoryAutomation ToolCategory = "automation"
)
```

### ToolCall e ToolResult

```go
type ToolCall struct {
    ID        string
    ToolName  string
    Arguments map[string]interface{}
}

type ToolResult struct {
    ToolCallID string
    Content    string
    IsError    bool
}
```

### Extensao de AIRequest e AIResponse

```go
// AIRequest ganha:
    Tools       []ToolDefinition
    ToolResults []ToolResult

// AIResponse ganha:
    ToolCalls []ToolCall
```

O `FinishReason` existente diferencia `"tool_calls"` de `"stop"`.

## 2. Tool Interface e Tools Concretas

### Interface

```go
type Tool interface {
    Definition() ToolDefinition
    Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*ToolResult, error)
}
```

### Categoria 1: Consulta de Dados

| Tool | Descricao | Parametros |
|---|---|---|
| `SearchLeadsTool` | Busca leads por nome, telefone ou status | `query` (string), `status` (enum, opcional) |
| `GetLeadDetailTool` | Detalhe completo de um lead | `lead_id` (string) |
| `GetConversationHistoryTool` | Historico de conversa WhatsApp | `conversation_id` (string), `limit` (number, opcional) |
| `ListProductsTool` | Lista produtos do tenant | — |
| `GetPipelineTool` | Estado do kanban/funil | `funnel_id` (string, opcional) |

### Categoria 2: Acoes no CRM

| Tool | Descricao | Parametros |
|---|---|---|
| `MoveLeadTool` | Move lead para outra coluna | `lead_id` (string), `column_id` (string) |
| `CreateLeadNoteTool` | Adiciona nota a um lead | `lead_id` (string), `content` (string) |
| `UpdateLeadScoreTool` | Altera score do lead | `lead_id` (string), `score` (number) |

### Categoria 3: Automacoes

| Tool | Descricao | Parametros |
|---|---|---|
| `TriggerAutomationTool` | Dispara automacao manualmente | `automation_id` (string), `lead_id` (string) |
| `SwitchSpecialistTool` | Troca especialista mid-conversation | `specialist_id` (string) |

### Localizacao no projeto

```
internal/ai/
  domain/
    tool.go
  infrastructure/
    tools/
      search_leads.go
      get_lead_detail.go
      get_history.go
      list_products.go
      get_pipeline.go
      move_lead.go
      create_note.go
      update_score.go
      trigger_automation.go
      switch_specialist.go
```

## 3. Tool Registry e Filtragem por Especialista

### ToolRegistry

```go
// internal/ai/application/tool_registry.go

type ToolRegistry struct {
    tools map[string]domain.Tool
}

func NewToolRegistry() *ToolRegistry
func (r *ToolRegistry) Register(tool domain.Tool)
func (r *ToolRegistry) Get(name string) (domain.Tool, error)
func (r *ToolRegistry) All() []domain.Tool
```

### Associacao especialista-tool

Nova entidade no modulo specialist:

```go
// internal/specialist/domain/specialist_tool.go

type SpecialistTool struct {
    ID           string
    SpecialistID string
    ToolName     string
    CreatedAt    time.Time
}
```

### ToolResolver

```go
// internal/ai/application/tool_resolver.go

type SpecialistToolFinder interface {
    FindBySpecialistID(ctx context.Context, specialistID string) ([]string, error)
}

type ToolResolver struct {
    registry             *ToolRegistry
    specialistToolFinder SpecialistToolFinder
}

func (r *ToolResolver) ResolveForSpecialist(ctx context.Context, specialistID string) ([]domain.Tool, error)
func (r *ToolResolver) ApplyStepConstraints(tools []domain.Tool, step *specDomain.Step) []domain.Tool
```

### Integracao com steps (hibrido)

Novos campos opcionais no `Step`:

```go
type Step struct {
    // ... campos existentes ...
    ForcedTools     []string  // tools que DEVEM ser enviadas neste step
    RestrictedTools []string  // tools BLOQUEADAS neste step
}
```

Persistencia: ambos os campos sao armazenados como JSON string na coluna (`forced_tools`, `restricted_tools`). Serializado/deserializado no repository GORM com `datatypes.JSON` ou marshal/unmarshal manual. Mesmo padrao usado em `McpServer.Config`.

Logica:

```
tools disponiveis = tools do especialista
if step.ForcedTools nao vazio:
    tools disponiveis = intersecao(tools do especialista, step.ForcedTools)
if step.RestrictedTools nao vazio:
    tools disponiveis = tools disponiveis - step.RestrictedTools
```

## 4. Provider Adapter e Tool Calling Loop

### Interface AIProvider — nao muda

```go
type AIProvider interface {
    Name() string
    GenerateResponse(ctx context.Context, req *AIRequest) (*AIResponse, error)
}
```

Cada provider converte `ToolDefinition` para seu formato internamente. O `ConversationEngine` so trabalha com tipos do dominio.

### OpenAI Provider — adaptacao

Novos metodos internos:

```go
func (p *OpenAIProvider) buildTools(tools []domain.ToolDefinition) []openAITool
func (p *OpenAIProvider) parseToolCalls(choices []openAIChoice) []domain.ToolCall
```

### Tool Calling Loop

```go
// internal/ai/application/conversation_engine.go

func (e *ConversationEngine) executeToolLoop(
    ctx context.Context,
    provider domain.AIProvider,
    req *domain.AIRequest,
    tenantID string,
    maxIterations int,
) (*domain.AIResponse, error) {
    for i := 0; i < maxIterations; i++ {
        resp, err := provider.GenerateResponse(ctx, req)
        if err != nil {
            return nil, err
        }

        if len(resp.ToolCalls) == 0 {
            return resp, nil
        }

        var results []domain.ToolResult
        for _, call := range resp.ToolCalls {
            tool, tErr := e.toolRegistry.Get(call.ToolName)
            if tErr != nil {
                results = append(results, domain.ToolResult{
                    ToolCallID: call.ID,
                    Content:    "tool not found: " + call.ToolName,
                    IsError:    true,
                })
                continue
            }
            result, execErr := tool.Execute(ctx, tenantID, call.Arguments)
            if execErr != nil {
                results = append(results, domain.ToolResult{
                    ToolCallID: call.ID,
                    Content:    "error: " + execErr.Error(),
                    IsError:    true,
                })
                continue
            }
            results = append(results, *result)
        }

        req.ToolResults = results
    }

    return nil, fmt.Errorf("tool loop exceeded max iterations (%d)", maxIterations)
}
```

### Integracao no HandleMessages

Substituicao pontual na linha 131 do `conversation_engine.go`:

```go
// Antes:
resp, err := provider.GenerateResponse(ctx, req)

// Depois:
resp, err := e.executeToolLoop(ctx, provider, req, tenantID, e.cfg.ToolLoopMaxIterations)
```

## 5. Integracao no Module Wiring e Admin UI

### ModuleDeps estendido

```go
type ModuleDeps struct {
    // ... deps existentes ...
    LeadRepository      funnel.LeadRepository
    LeadNoteRepository  funnel.LeadNoteRepository
    ColumnRepository    funnel.ColumnRepository
    FunnelRepository    funnel.FunnelRepository
    ProductRepository   product.ProductRepository
    AutomationExecutor  automation.Executor
    SpecialistToolRepo  specialist.SpecialistToolRepository
}
```

### Inicializacao no Module

```go
func NewModule(...) *Module {
    // 1. Cria tools concretas
    // 2. Registra no ToolRegistry
    // 3. Cria ToolResolver
    // 4. Injeta no ConversationEngine
}
```

### Nova tabela

```sql
CREATE TABLE specialist_tools (
    id            VARCHAR(36) PRIMARY KEY,
    specialist_id VARCHAR(36) NOT NULL,
    tool_name     VARCHAR(100) NOT NULL,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (specialist_id) REFERENCES specialists(id) ON DELETE CASCADE,
    UNIQUE KEY idx_specialist_tool (specialist_id, tool_name)
);
```

### Admin UI — rotas

```
GET    /admin/specialists/:id/tools         -> lista tools associadas
POST   /admin/specialists/:id/tools         -> associa tool
DELETE /admin/specialists/:id/tools/:name   -> desassocia tool
```

Mesma UX de documentos/MCPs: checkboxes agrupados por categoria, HTMX submit parcial.

### ContextBuilder — novo passo

```go
func (b *ContextBuilder) Build(...) (*domain.AIRequest, error) {
    // ... passos existentes (1-8) ...
    // 9. Resolve tools para especialista + step
    // 10. Filtra por step constraints (forced/restricted)
    // Retorna AIRequest com Tools preenchido
}
```

## 6. Seguranca, Limites e Observabilidade

### Seguranca

- **Tenant isolation**: `tenantID` vem do middleware, nunca do LLM. Toda tool usa `TenantScope(ctx)`.
- **Validacao de argumentos**: cada tool valida antes de executar. Falha retorna `ToolResult{IsError: true}`.
- **Escopo restrito**: tools nao criam tenants, nao alteram permissoes, nao acessam dados cross-tenant.

### Limites

| Limite | Default | Env var |
|---|---|---|
| Max iteracoes do tool loop | 5 | `AI_TOOL_LOOP_MAX_ITERATIONS` |
| Max tool calls por iteracao | 10 | `AI_TOOL_CALL_MAX_PER_ITERATION` |
| Timeout por tool execution | 10s | `AI_TOOL_EXECUTION_TIMEOUT_SECONDS` |
| Max tamanho do ToolResult | 4000 chars | `AI_TOOL_RESULT_MAX_LENGTH` |

### Metricas Prometheus

```
ai_tool_calls_total{tenant_id, specialist_id, tool_name, status}
ai_tool_call_duration_seconds{tenant_id, tool_name}
ai_tool_loop_iterations{tenant_id, specialist_id}
ai_tool_result_truncated_total{tenant_id, tool_name}
```

### Logs estruturados (Zap)

```go
// A cada tool call
e.log.Info("tool_call_executed",
    zap.String("tenant_id", tenantID),
    zap.String("specialist_id", specialistID),
    zap.String("conversation_id", conversationID),
    zap.String("tool_name", call.ToolName),
    zap.Duration("duration", elapsed),
    zap.Bool("is_error", result.IsError),
)
```

### Traces

Cada tool call gera um span filho do span de `HandleMessages`, com attributes de tool name e status. Segue setup OpenTelemetry existente.

### Tratamento de erros

Erros de tool nao crasham a conversa. O LLM recebe `ToolResult{IsError: true}` e decide o que fazer. Unico cenario que interrompe: falha do provider (rede, auth).

## Resumo de arquivos novos/modificados

### Novos

| Arquivo | Descricao |
|---|---|
| `internal/ai/domain/tool.go` | ToolDefinition, Tool, ToolCall, ToolResult, ToolCategory, ParameterDef |
| `internal/ai/application/tool_registry.go` | ToolRegistry |
| `internal/ai/application/tool_resolver.go` | ToolResolver, SpecialistToolFinder, ApplyStepConstraints |
| `internal/ai/infrastructure/tools/*.go` | 10 tools concretas |
| `internal/specialist/domain/specialist_tool.go` | SpecialistTool entity |
| `internal/specialist/infrastructure/gorm_specialist_tool_repository.go` | Repositorio GORM |
| `internal/specialist/interfaces/http/tool_handler.go` | Endpoints admin |
| `web/templates/specialist/tools.html` | UI de associacao |
| `migrations/XXXXXX_create_specialist_tools.up.sql` | Migration: tabela specialist_tools |
| `migrations/XXXXXX_add_step_tool_fields.up.sql` | Migration: forced_tools e restricted_tools no steps |

### Modificados

| Arquivo | Mudanca |
|---|---|
| `internal/ai/domain/provider.go` | AIRequest + Tools/ToolResults, AIResponse + ToolCalls |
| `internal/ai/application/conversation_engine.go` | executeToolLoop, injecao de ToolRegistry |
| `internal/ai/application/context_builder.go` | Passo 9-10: resolve tools por especialista/step |
| `internal/ai/infrastructure/openai_provider.go` | buildTools, parseToolCalls, mensagens tool role |
| `internal/ai/module.go` | ModuleDeps estendido, wiring de tools |
| `internal/specialist/domain/step.go` | ForcedTools, RestrictedTools |
| `internal/shared/config/config.go` | AIConfigEnv com limites de tools |
| `cmd/api/main.go` | Wiring dos novos adapters |
