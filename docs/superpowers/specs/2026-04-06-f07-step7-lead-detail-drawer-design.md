# F07 Step 7 — Painel de Detalhes do Lead (Drawer)

## Resumo

Substituir o modal centrado atual por um drawer lateral (50% da tela, abrindo da direita) com overlay bloqueante. O drawer exibe dados enriquecidos do lead: contato, funil, mensagens WhatsApp, anotações manuais, histórico de movimentações e placeholders para features futuras.

## Decisões de design

| Decisao | Escolha |
|---------|---------|
| Tipo de painel | Drawer lateral direito com overlay bloqueante |
| Largura | ~50% da tela |
| Mensagens WhatsApp | 10 ultimas mensagens no preview |
| Anotacoes | Lista com historico (data/hora + conteudo) |
| Itens futuros (F08, F10, F14) | Secoes com placeholder "Nenhum associado" |
| Arquitetura cross-module | Interfaces no dominio funnel (ContactProvider, MessageProvider) |

## 1. Dominio

### Nova entidade: LeadNote

```go
type LeadNote struct {
    ID        string
    LeadID    string
    TenantID  string
    Content   string    // max 2000 chars
    CreatedBy string    // user_id
    CreatedAt time.Time
}
```

Validacoes:
- `LeadID`, `TenantID`, `Content`, `CreatedBy` obrigatorios
- `Content` max 2000 caracteres

### Novo repositorio: LeadNoteRepository

```go
type LeadNoteRepository interface {
    Create(ctx context.Context, note *LeadNote) error
    FindByLeadID(ctx context.Context, leadID string) ([]LeadNote, error)
}
```

### Novas interfaces cross-module

```go
type ContactProvider interface {
    FindByID(ctx context.Context, contactID string) (ContactInfo, error)
}

type ContactInfo struct {
    Name  string
    Phone string
}

type MessageProvider interface {
    FindRecentByConversationID(ctx context.Context, conversationID string, limit int) ([]MessageSummary, error)
}

type MessageSummary struct {
    Direction string    // "incoming" ou "outgoing"
    Content   string
    Timestamp time.Time
}
```

Estas interfaces vivem em `internal/funnel/domain/`. O dominio funnel nao importa o pacote whatsapp.

### Migration

Tabela `lead_notes`:

```sql
CREATE TABLE lead_notes (
    id VARCHAR(36) PRIMARY KEY,
    lead_id VARCHAR(36) NOT NULL,
    tenant_id VARCHAR(36) NOT NULL,
    content TEXT NOT NULL,
    created_by VARCHAR(36) NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE CASCADE,
    INDEX idx_lead_notes_lead_id (lead_id),
    INDEX idx_lead_notes_tenant_id (tenant_id)
);
```

## 2. Application

### GetLeadDetail (reformulado)

O use case atual depende apenas de `LeadRepository` e `LeadMovementRepository`. O novo adiciona:
- `FunnelRepository` (ja existe no dominio) — para nome do funil
- `ColumnRepository` (ja existe no dominio) — para nome da coluna atual e nomes nas movimentacoes
- `ContactProvider` (nova interface) — para nome e telefone do contato
- `MessageProvider` (nova interface) — para mensagens recentes
- `LeadNoteRepository` (novo) — para anotacoes

Fluxo:

1. Busca lead + valida tenant (como hoje)
2. Usa `FunnelRepository.FindByID` para nome do funil
3. Usa `ColumnRepository.FindByID` para nome da coluna atual
4. Usa `ContactProvider` para resolver nome e telefone
5. Usa `MessageProvider` para buscar 10 ultimas mensagens
6. Busca movimentacoes e resolve nomes das colunas via `ColumnRepository`
7. Busca anotacoes via `LeadNoteRepository`

Output enriquecido:

```go
type LeadDetailOutput struct {
    ID              string
    TenantID        string
    FunnelID        string
    FunnelName      string
    ColumnID        string
    ColumnName      string
    ContactID       string
    ContactName     string
    ContactPhone    string
    ConversationID  string
    Score           int
    Status          string
    ColumnEnteredAt time.Time
    CreatedAt       time.Time
    Messages        []MessageSummaryOutput
    Movements       []LeadMovementOutput  // com FromColumnName, ToColumnName
    Notes           []LeadNoteOutput
    // Placeholders futuros
    ProductName    string   // vazio ate F10
    AssignedToName string   // vazio ate F08
}

type MessageSummaryOutput struct {
    Direction string
    Content   string
    Timestamp time.Time
}

type LeadNoteOutput struct {
    ID        string
    Content   string
    CreatedBy string
    CreatedAt time.Time
}

type LeadMovementOutput struct {
    ID             string
    FromColumnID   string
    FromColumnName string
    ToColumnID     string
    ToColumnName   string
    MovedAt        time.Time
}
```

### CreateLeadNote (novo)

```go
type CreateLeadNoteInput struct {
    TenantID  string
    LeadID    string
    Content   string
    CreatedBy string
}
```

1. Valida que o lead existe e pertence ao tenant
2. Cria `LeadNote` via dominio
3. Persiste via `LeadNoteRepository`
4. Retorna a nota criada

## 3. Infrastructure

### Adaptadores cross-module

Em `internal/funnel/infrastructure/`:

**WhatsAppContactAdapter** — implementa `domain.ContactProvider`:
- Recebe `whatsapp.domain.ContactRepository`
- `FindByID` delega e converte para `domain.ContactInfo`

**WhatsAppMessageAdapter** — implementa `domain.MessageProvider`:
- Recebe `whatsapp.domain.MessageRepository`
- `FindRecentByConversationID` busca mensagens com limit, converte para `[]domain.MessageSummary`

### GormLeadNoteRepository

Em `internal/funnel/infrastructure/`:
- Implementa `domain.LeadNoteRepository`
- Model Gorm `LeadNoteModel` com conversao de/para dominio

## 4. Interfaces HTTP

### Rotas

Rota existente (sem mudanca de URL):
- `GET /tenant/leads/:id` — `RenderLeadDetail` — retorna drawer completo

Nova rota:
- `POST /tenant/leads/:id/notes` — `HandleCreateNote` — cria anotacao, retorna partial da secao de notas

### Handler

O handler recebe as novas dependencias no construtor:
- `createLeadNoteUC *application.CreateLeadNoteUseCase`

O `GetLeadDetailUseCase` ja recebe `ContactProvider`, `MessageProvider`, `LeadNoteRepository` via construtor.

**RenderLeadDetail:** chama use case enriquecido, renderiza `funnel/lead_drawer.html`.

**HandleCreateNote:** recebe content via form, chama `CreateLeadNoteUseCase`, renderiza partial `funnel/lead_notes_section.html` via HTMX swap.

## 5. Frontend

### Template: lead_drawer.html

Substitui `lead_detail.html`. Estrutura:

```
.lead-drawer-overlay (overlay escuro, clicavel para fechar)
  .lead-drawer (50% da tela, posicionado a direita, slide-in)
    .lead-drawer-header
      Nome do contato + botao X
    .lead-drawer-body (scroll interno)
      Secao: Contato (nome, telefone)
      Secao: Funil (nome, coluna, score, data criacao)
      Secao: Acoes (botoes "Abrir Conversa" e "Mover Lead")
      Secao: Mensagens WhatsApp (10 ultimas, bolhas incoming/outgoing, link "Ver conversa")
      Secao: Anotacoes (lista + form inline HTMX para adicionar)
      Secao: Historico de movimentacoes (timeline)
      Secao: Produto (placeholder)
      Secao: Responsavel (placeholder)
      Secao: Documentos (placeholder)
```

### Interacao HTMX

- Card no kanban: `hx-get="/tenant/leads/:id"` → `hx-target="#lead-modal"` (como hoje)
- Form de anotacao: `hx-post="/tenant/leads/:id/notes"` → `hx-target="#lead-notes-section"` → swap apenas a lista de notas
- Botao "Mover Lead": `hx-get="/tenant/leads/:id/move-form"` → `hx-target="#lead-modal"` (reutiliza fluxo existente)

### CSS (kanban.css)

Novas classes:

- `.lead-drawer-overlay` — fixed, inset 0, background rgba escuro, z-index 100
- `.lead-drawer` — fixed, right 0, top 0, bottom 0, width 50%, background branco, transform translateX com animacao
- `.lead-drawer-header`, `.lead-drawer-body` — layout interno
- `.lead-message-bubble`, `.lead-message-incoming`, `.lead-message-outgoing` — bolhas de mensagem
- `.lead-note-item`, `.lead-note-form` — anotacoes
- Media query: em telas < 768px, drawer ocupa 100% da largura

### JS (kanban.js)

Adaptar `closeLeadModal` para funcionar com o drawer (fechar no clique do overlay ou tecla Escape).

## 6. Composicao (module.go)

Atualizar `internal/funnel/module.go`:
- Criar adaptadores `WhatsAppContactAdapter` e `WhatsAppMessageAdapter` recebendo repos do WhatsApp
- Injetar no `GetLeadDetailUseCase`
- Criar `GormLeadNoteRepository`
- Criar `CreateLeadNoteUseCase`
- Injetar `CreateLeadNoteUseCase` no handler

## 7. Testes

### Unitarios — Dominio
- `LeadNote`: validacoes (content vazio, max length, campos obrigatorios)

### Unitarios — Application
- `GetLeadDetail`: mock de ContactProvider, MessageProvider, LeadNoteRepository; validar output enriquecido
- `CreateLeadNote`: criacao, validacao de tenant, erro se lead nao existe

### Unitarios — Infrastructure
- `WhatsAppContactAdapter`: conversao correta
- `WhatsAppMessageAdapter`: conversao e limite de mensagens

### HTTP — Handler
- `RenderLeadDetail`: 200 com dados completos
- `RenderLeadDetail`: 404 se lead nao existe / tenant errado
- `HandleCreateNote`: retorna secao de notas atualizada
- `HandleCreateNote`: 422 se content vazio

### OWASP
- Nao acessar lead de outro tenant (GET)
- Nao criar nota em lead de outro tenant (POST)
