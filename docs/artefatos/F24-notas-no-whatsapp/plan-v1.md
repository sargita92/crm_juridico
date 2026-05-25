# F24 — Notas na tela de WhatsApp — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans or
> superpowers:subagent-driven-development to implement this plan task-by-task.
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Permitir ver e adicionar notas do cliente direto na tela de WhatsApp, num
drawer deslizante acionado por um botão no cabeçalho do chat, operando sobre o lead
atual da conversa.

**Architecture:** Porta `LeadNotesService` no `whatsapp/domain` implementada por um
adapter no `funnel` (padrão `LeadCreator`, wired em `main.go`). Reusa `LeadNote`,
tabela `lead_notes` e `CreateLeadNoteUseCase`. Query nova `FindCurrentByConversationID`
(lead mais recente da conversa, filtrado por tenant). Frontend HTMX puro.

**Tech Stack:** Go, Gin, Gorm, html/template, HTMX, testify, testcontainers-go.

---

## Fase A — Backend funnel: query do lead atual

### Task A1: `FindCurrentByConversationID` no repositório de leads

**Files:**
- Modify: `internal/funnel/domain/repository.go` (interface `LeadRepository`)
- Modify: `internal/funnel/infrastructure/gorm_lead_repository.go`
- Test: `internal/funnel/infrastructure/gorm_lead_repository_test.go`

- [ ] **Step 1: Teste falhando** — adicionar ao test file:

```go
func TestGormLead_FindCurrentByConversationID_ReturnsMostRecent(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	ctx := context.Background()
	fx := newLeadFixture(t, repos, db)

	// Lead antigo na conversa.
	old, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, err)
	old.CreatedAt = time.Now().Add(-1 * time.Hour)
	require.NoError(t, repos.leads.Create(ctx, old))

	// Lead novo (destino de cross-sell) na MESMA conversa.
	recent, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, err)
	recent.CreatedAt = time.Now()
	require.NoError(t, repos.leads.Create(ctx, recent))

	got, err := repos.leads.FindCurrentByConversationID(ctx, fx.tenantID, fx.conversationID)
	require.NoError(t, err)
	assert.Equal(t, recent.ID, got.ID)
}

func TestGormLead_FindCurrentByConversationID_TenantIsolation(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	ctx := context.Background()
	fx := newLeadFixture(t, repos, db)

	lead, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, err)
	require.NoError(t, repos.leads.Create(ctx, lead))

	_, err = repos.leads.FindCurrentByConversationID(ctx, "outro-tenant", fx.conversationID)
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestGormLead_FindCurrentByConversationID_NotFound(t *testing.T) {
	repos, _ := setupFunnelRepos(t)
	_, err := repos.leads.FindCurrentByConversationID(context.Background(), "t", uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}
```

- [ ] **Step 2: Rodar e ver falhar** (compila? não — método inexistente):
  `go test ./internal/funnel/infrastructure/ -run FindCurrentByConversationID` → FAIL/compile error.

- [ ] **Step 3: Adicionar à interface** em `repository.go`, dentro de `LeadRepository`:

```go
	FindByConversationID(ctx context.Context, conversationID string) (*Lead, error)
	// FindCurrentByConversationID returns the most recent lead (by created_at) of a
	// conversation within a tenant. After a cross-sell, the newest lead is the one the
	// conversation is currently on. ErrLeadNotFound when none.
	FindCurrentByConversationID(ctx context.Context, tenantID, conversationID string) (*Lead, error)
```

- [ ] **Step 4: Implementar** no `gorm_lead_repository.go` (após `FindByConversationID`):

```go
func (r *GormLeadRepository) FindCurrentByConversationID(ctx context.Context, tenantID, conversationID string) (*domain.Lead, error) {
	var model leadModel
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND tenant_id = ?", conversationID, tenantID).
		Order("created_at DESC").
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrLeadNotFound
		}
		return nil, err
	}
	return leadToDomain(&model), nil
}
```

- [ ] **Step 5: Rodar e passar:**
  `go test ./internal/funnel/infrastructure/ -run FindCurrentByConversationID` → PASS.

- [ ] **Step 6: Atualizar mocks** que implementam `LeadRepository` (compilação dos outros pacotes):
  `internal/funnel/application/mocks_test.go` e `internal/funnel/interfaces/http/owasp_test.go` ganham:

```go
func (m *mockLeadRepo) FindCurrentByConversationID(_ context.Context, _, conversationID string) (*domain.Lead, error) {
	// mesma lógica do FindByConversationID do mock
	for _, l := range m.leads {
		if l.ConversationID == conversationID {
			return l, nil
		}
	}
	return nil, domain.ErrLeadNotFound
}
```
  (ajustar nome do campo/coleção ao mock existente.)

- [ ] **Step 7: Build** `go build ./...` → ok. **Commit:**
  `git commit -am "feat(F24): FindCurrentByConversationID no repo de leads"`

---

## Fase B — Porta + adapter de notas

### Task B1: Porta `LeadNotesService` no whatsapp/domain

**Files:**
- Create: `internal/whatsapp/domain/lead_notes_service.go`

- [ ] **Step 1: Criar arquivo:**

```go
package domain

import (
	"context"
	"time"
)

// ConversationNote is a note shown in the WhatsApp chat notes drawer.
type ConversationNote struct {
	ID         string
	Content    string
	AuthorName string
	CreatedAt  time.Time
}

// LeadNotesService gives the WhatsApp screen access to the notes of the lead that a
// conversation is currently on. Implemented by the funnel module.
type LeadNotesService interface {
	// NotesForConversation resolves the current lead of the conversation and lists its
	// notes. hasLead=false (no error) when the conversation has no lead in this tenant.
	NotesForConversation(ctx context.Context, tenantID, conversationID string) (hasLead bool, notes []ConversationNote, err error)
	// AddNote creates a note on the current lead and returns the refreshed list.
	AddNote(ctx context.Context, tenantID, conversationID, content, createdBy string) (notes []ConversationNote, err error)
}
```

- [ ] **Step 2: Build** `go build ./internal/whatsapp/...` → ok. **Commit.**

### Task B2: Adapter no funnel

**Files:**
- Create: `internal/funnel/infrastructure/whatsapp_notes_adapter.go`
- Test: `internal/funnel/infrastructure/whatsapp_notes_adapter_test.go`

- [ ] **Step 1: Teste falhando (integração):**

```go
package infrastructure

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/application"
	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type stubUserName struct{ name string }

func (s stubUserName) FindNameByID(_ context.Context, _ string) (string, error) { return s.name, nil }

func newNotesAdapter(repos *funnelTestRepos) *WhatsAppNotesAdapter {
	createUC := application.NewCreateLeadNoteUseCase(repos.leads, repos.notes)
	return NewWhatsAppNotesAdapter(repos.leads, repos.notes, stubUserName{name: "Maria"}, createUC)
}

func TestNotesAdapter_AddAndList(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	ctx := context.Background()
	fx := newLeadFixture(t, repos, db)

	lead, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, err)
	require.NoError(t, repos.leads.Create(ctx, lead))

	a := newNotesAdapter(repos)

	notes, err := a.AddNote(ctx, fx.tenantID, fx.conversationID, "primeira nota", uuid.New().String())
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "primeira nota", notes[0].Content)
	assert.Equal(t, "Maria", notes[0].AuthorName)

	has, list, err := a.NotesForConversation(ctx, fx.tenantID, fx.conversationID)
	require.NoError(t, err)
	assert.True(t, has)
	require.Len(t, list, 1)
}

func TestNotesAdapter_NoLead(t *testing.T) {
	repos, _ := setupFunnelRepos(t)
	a := newNotesAdapter(repos)

	has, list, err := a.NotesForConversation(context.Background(), "t", uuid.New().String())
	require.NoError(t, err)
	assert.False(t, has)
	assert.Empty(t, list)
}
```

- [ ] **Step 2: Rodar e ver falhar** (tipo inexistente): compile error.

- [ ] **Step 3: Implementar adapter:**

```go
package infrastructure

import (
	"context"
	"errors"

	"github.com/sasrgita/crm-juridico/internal/funnel/application"
	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// WhatsAppNotesAdapter implements whatsapp/domain.LeadNotesService over the funnel
// lead/note repositories, operating on the current lead of a conversation.
type WhatsAppNotesAdapter struct {
	leadRepo  domain.LeadRepository
	noteRepo  domain.LeadNoteRepository
	userNames domain.UserNameProvider
	createUC  *application.CreateLeadNoteUseCase
}

func NewWhatsAppNotesAdapter(
	leadRepo domain.LeadRepository,
	noteRepo domain.LeadNoteRepository,
	userNames domain.UserNameProvider,
	createUC *application.CreateLeadNoteUseCase,
) *WhatsAppNotesAdapter {
	return &WhatsAppNotesAdapter{leadRepo: leadRepo, noteRepo: noteRepo, userNames: userNames, createUC: createUC}
}

func (a *WhatsAppNotesAdapter) NotesForConversation(ctx context.Context, tenantID, conversationID string) (bool, []whatsappdomain.ConversationNote, error) {
	lead, err := a.leadRepo.FindCurrentByConversationID(ctx, tenantID, conversationID)
	if err != nil {
		if errors.Is(err, domain.ErrLeadNotFound) {
			return false, nil, nil
		}
		return false, nil, err
	}
	notes, err := a.listNotes(ctx, lead.ID)
	if err != nil {
		return false, nil, err
	}
	return true, notes, nil
}

func (a *WhatsAppNotesAdapter) AddNote(ctx context.Context, tenantID, conversationID, content, createdBy string) ([]whatsappdomain.ConversationNote, error) {
	lead, err := a.leadRepo.FindCurrentByConversationID(ctx, tenantID, conversationID)
	if err != nil {
		return nil, err
	}
	if _, err := a.createUC.Execute(ctx, application.CreateLeadNoteInput{
		TenantID:  tenantID,
		LeadID:    lead.ID,
		Content:   content,
		CreatedBy: createdBy,
	}); err != nil {
		return nil, err
	}
	return a.listNotes(ctx, lead.ID)
}

func (a *WhatsAppNotesAdapter) listNotes(ctx context.Context, leadID string) ([]whatsappdomain.ConversationNote, error) {
	notes, err := a.noteRepo.FindByLeadID(ctx, leadID)
	if err != nil {
		return nil, err
	}
	out := make([]whatsappdomain.ConversationNote, 0, len(notes))
	for _, n := range notes {
		name, _ := a.userNames.FindNameByID(ctx, n.CreatedBy)
		out = append(out, whatsappdomain.ConversationNote{
			ID:         n.ID,
			Content:    n.Content,
			AuthorName: name,
			CreatedAt:  n.CreatedAt,
		})
	}
	return out, nil
}
```

- [ ] **Step 4: Rodar e passar:** `go test ./internal/funnel/infrastructure/ -run NotesAdapter` → PASS.

- [ ] **Step 5: Build + Commit** `feat(F24): adapter LeadNotesService no funnel`.

### Task B3: Expor no funnel module

**Files:**
- Modify: `internal/funnel/module.go`

- [ ] **Step 1:** guardar `createLeadNoteUC` no struct `Module` e expor:

```go
// no struct Module:
	createLeadNoteUC *application.CreateLeadNoteUseCase
// no return de NewModule: createLeadNoteUC: createLeadNoteUC,
// novo método:
func (m *Module) LeadNotesService() whatsappdomain.LeadNotesService {
	return infrastructure.NewWhatsAppNotesAdapter(m.leadRepo, m.noteRepo, m.userNameProvider, m.createLeadNoteUC)
}
```
  Guardar também `userNameProvider` no struct (hoje só é parâmetro de `NewModule`). Import:
  `whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"`.

- [ ] **Step 2: Build** `go build ./...` → ok. **Commit.**

---

## Fase C — Wiring + handlers + rotas

### Task C1: whatsapp module + handler aceitam o serviço

**Files:**
- Modify: `internal/whatsapp/interfaces/http/handler.go`
- Modify: `internal/whatsapp/module.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1:** handler ganha campo + setter:

```go
// struct Handler: notesService domain.LeadNotesService
func (h *Handler) SetNotesService(s domain.LeadNotesService) { h.notesService = s }
```

- [ ] **Step 2:** module:

```go
func (m *Module) SetLeadNotesService(s domain.LeadNotesService) {
	m.handler.SetNotesService(s)
}
```

- [ ] **Step 3:** `cmd/api/main.go`, após a linha `whatsappMod.SetLeadCreator(funnelMod.LeadCreator())`:

```go
	whatsappMod.SetLeadNotesService(funnelMod.LeadNotesService())
```

- [ ] **Step 4: Build + Commit.**

### Task C2: Handlers RenderNotesPanel + HandleCreateNote (TDD)

**Files:**
- Modify: `internal/whatsapp/interfaces/http/handler.go`
- Modify: `internal/whatsapp/interfaces/http/routes.go`
- Test: `internal/whatsapp/interfaces/http/notes_handler_test.go`
- Modify (stub template): `internal/whatsapp/interfaces/http/handler_test.go` e `owasp_test.go`
  (adicionar `"whatsapp/notes_panel.html"` à lista de templates stub; em `setupTest`,
  fazer o `authMw` no-op setar user: `c.Set("user_id", "user-1")`).

- [ ] **Step 1: Teste falhando** em `notes_handler_test.go`:

```go
package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type mockNotesService struct {
	hasLead bool
	notes   []domain.ConversationNote
	addErr  error
}

func (m *mockNotesService) NotesForConversation(_ context.Context, _, _ string) (bool, []domain.ConversationNote, error) {
	return m.hasLead, m.notes, nil
}
func (m *mockNotesService) AddNote(_ context.Context, _, _, content, _ string) ([]domain.ConversationNote, error) {
	if m.addErr != nil {
		return nil, m.addErr
	}
	m.notes = append(m.notes, domain.ConversationNote{Content: content, AuthorName: "User", CreatedAt: time.Now()})
	return m.notes, nil
}

func TestRenderNotesPanel_HasLead(t *testing.T) {
	deps := setupTest()
	svc := &mockNotesService{hasLead: true, notes: []domain.ConversationNote{{Content: "oi"}}}
	deps.handler.SetNotesService(svc)

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/conversations/c1/notes"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleCreateNote_Success(t *testing.T) {
	deps := setupTest()
	svc := &mockNotesService{hasLead: true}
	deps.handler.SetNotesService(svc)

	form := url.Values{"content": {"nota nova"}}
	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodPost, "/tenant/whatsapp/conversations/c1/notes", form.Encode()))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, svc.notes, 1)
}

func TestHandleCreateNote_Empty(t *testing.T) {
	deps := setupTest()
	deps.handler.SetNotesService(&mockNotesService{hasLead: true})

	form := url.Values{"content": {""}}
	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodPost, "/tenant/whatsapp/conversations/c1/notes", form.Encode()))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
```

- [ ] **Step 2: Rodar e ver falhar** (rota/handler inexistente → 404).

- [ ] **Step 3: Rotas** em `routes.go` (após a linha de messages):

```go
	tenant.GET("/conversations/:id/notes", h.RenderNotesPanel)
	tenant.POST("/conversations/:id/notes", h.HandleCreateNote)
```

- [ ] **Step 4: Handlers** em `handler.go`:

```go
func (h *Handler) RenderNotesPanel(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	convID := c.Param("id")

	hasLead, notes, err := h.notesService.NotesForConversation(c.Request.Context(), tenantID, convID)
	if err != nil {
		h.log.Error("failed to load notes", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "whatsapp/notes_panel.html", gin.H{"Error": "Erro ao carregar notas"})
		return
	}
	c.HTML(http.StatusOK, "whatsapp/notes_panel.html", gin.H{
		"ConversationID": convID,
		"HasLead":        hasLead,
		"Notes":          notes,
	})
}

func (h *Handler) HandleCreateNote(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	convID := c.Param("id")
	content := c.PostForm("content")
	userID := c.GetString("user_id")

	if content == "" {
		c.HTML(http.StatusBadRequest, "whatsapp/notes_panel.html", gin.H{
			"ConversationID": convID, "HasLead": true,
			"Error": "A nota nao pode ser vazia",
		})
		return
	}

	notes, err := h.notesService.AddNote(c.Request.Context(), tenantID, convID, content, userID)
	if err != nil {
		h.log.Error("failed to create note", zap.Error(err))
		c.HTML(http.StatusUnprocessableEntity, "whatsapp/notes_panel.html", gin.H{
			"ConversationID": convID, "HasLead": true,
			"Error": "Erro ao adicionar nota",
		})
		return
	}
	c.HTML(http.StatusOK, "whatsapp/notes_panel.html", gin.H{
		"ConversationID": convID, "HasLead": true, "Notes": notes,
	})
}
```

- [ ] **Step 5: Rodar e passar** `go test ./internal/whatsapp/...` → PASS.
- [ ] **Step 6: Build + Commit.**

### Task C3: OWASP para as rotas de notas

**Files:**
- Modify: `internal/whatsapp/interfaces/http/owasp_test.go`

- [ ] **Step 1:** adicionar as 2 rotas à lista de `TestOWASP_A01_NoToken_Returns401`
  (`GET` e `POST` em `/tenant/whatsapp/conversations/some-id/notes`) e ao
  `TestOWASP_A01_NoTenant_Returns403` (o POST). No `setupOwaspEnv`, setar
  `handler.SetNotesService(&mockNotesService{})` e adicionar `notes_panel.html` ao stub.
  Adicionar teste XSS:

```go
func TestOWASP_A03_XSS_NoteContent(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")
	form := url.Values{"content": {"<script>alert('xss')</script>"}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tenant/whatsapp/conversations/c1/notes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(tokenCookieOwasp(token))
	env.router.ServeHTTP(w, req)
	assert.NotContains(t, w.Body.String(), "<script>alert")
}
```

- [ ] **Step 2: Rodar e passar** `go test ./internal/whatsapp/...` → PASS. **Commit.**

---

## Fase D — Frontend

### Task D1: Partial notes_panel.html

**Files:**
- Create: `web/templates/whatsapp/notes_panel.html`

- [ ] **Step 1: Criar** (estilo amarelo do Kanban; cabeçalho + ✕ que fecha o drawer):

```html
{{define "whatsapp/notes_panel.html"}}
<div class="wa-notes-header">
    <h3>📝 Notas do cliente</h3>
    <button class="wa-notes-close" type="button"
            hx-on:click="document.getElementById('wa-notes-drawer').classList.remove('open')">✕</button>
</div>
{{if .Error}}
<p class="wa-notes-error">{{.Error}}</p>
{{end}}
{{if .HasLead}}
<form class="wa-notes-form"
      hx-post="/tenant/whatsapp/conversations/{{.ConversationID}}/notes"
      hx-target="#wa-notes-drawer-body"
      hx-swap="innerHTML"
      hx-on::after-request="if(event.detail.successful) this.reset()">
    <textarea name="content" rows="2" required placeholder="Adicionar anotacao..."></textarea>
    <button type="submit">Adicionar</button>
</form>
{{if .Notes}}
<div class="wa-notes-list">
    {{range .Notes}}
    <div class="wa-note">
        <div class="wa-note-meta">
            <span class="wa-note-author">{{if .AuthorName}}{{.AuthorName}}{{else}}Usuario{{end}}</span>
            <span class="wa-note-time">{{.CreatedAt.Format "02/01/2006 15:04"}}</span>
        </div>
        <p class="wa-note-content">{{.Content}}</p>
    </div>
    {{end}}
</div>
{{else}}
<p class="wa-notes-empty">Nenhuma anotacao</p>
{{end}}
{{else}}
<p class="wa-notes-empty">Nenhum lead associado a esta conversa ainda.</p>
{{end}}
{{end}}
```

- [ ] **Step 2: Commit.**

### Task D2: Botão + drawer no chat.html

**Files:**
- Modify: `web/templates/whatsapp/chat.html`

- [ ] **Step 1:** no `.wa-chat-header`, após `wa-chat-header-info`, adicionar o botão:

```html
        <button class="wa-notes-btn" type="button"
                hx-get="/tenant/whatsapp/conversations/{{.ConversationID}}/notes"
                hx-target="#wa-notes-drawer-body"
                hx-swap="innerHTML"
                hx-on:click="document.getElementById('wa-notes-drawer').classList.add('open')">
            📝 Notas
        </button>
```
  (o header vira `justify-content: space-between` via CSS — ver D3.)

- [ ] **Step 2:** antes de fechar `.wa-chat-container` (após `.wa-input-area`), adicionar:

```html
    <aside id="wa-notes-drawer" class="wa-notes-drawer" aria-label="Notas do cliente">
        <div id="wa-notes-drawer-body"></div>
    </aside>
```

- [ ] **Step 3: Commit.**

### Task D3: CSS do drawer

**Files:**
- Modify: `web/static/css/whatsapp.css`

- [ ] **Step 1:** alterar `.wa-chat-container` para `position: relative; overflow: hidden;`
  e o `.wa-chat-header` para `justify-content: space-between;`. Acrescentar ao fim:

```css
/* --- Notas (drawer) --- */
.wa-notes-btn {
    margin-left: auto;
    background: #fff;
    border: 1px solid #d1d5db;
    border-radius: 8px;
    padding: 0.375rem 0.75rem;
    font-size: 0.8125rem;
    font-weight: 600;
    color: #41525d;
    cursor: pointer;
}
.wa-notes-btn:hover { background: #e9edef; }

.wa-notes-drawer {
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    width: 320px;
    max-width: 85%;
    background: #fff;
    border-left: 1px solid #e0e0e0;
    box-shadow: -4px 0 12px rgba(0, 0, 0, 0.12);
    transform: translateX(100%);
    transition: transform 0.2s ease;
    display: flex;
    flex-direction: column;
    z-index: 5;
    overflow-y: auto;
}
.wa-notes-drawer.open { transform: translateX(0); }

.wa-notes-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid #e0e0e0;
    background: #f0f2f5;
}
.wa-notes-header h3 { font-size: 0.9375rem; font-weight: 600; color: #111b21; }
.wa-notes-close { background: none; border: none; font-size: 1rem; cursor: pointer; color: #54656f; }

.wa-notes-form { display: flex; flex-direction: column; gap: 0.5rem; padding: 0.75rem 1rem; }
.wa-notes-form textarea {
    width: 100%; box-sizing: border-box; padding: 0.5rem 0.75rem;
    border: 1px solid #d1d5db; border-radius: 6px; font-size: 0.8125rem;
    resize: vertical; min-height: 2.5rem; font-family: inherit;
}
.wa-notes-form button {
    align-self: flex-end; padding: 0.5rem 1rem; font-size: 0.8125rem; font-weight: 600;
    color: #fff; background: #00a884; border: none; border-radius: 6px; cursor: pointer;
}
.wa-notes-list { display: flex; flex-direction: column; gap: 0.5rem; padding: 0 1rem 1rem; }
.wa-note { padding: 0.5rem 0.75rem; background: #fefce8; border-radius: 6px; border-left: 3px solid #eab308; }
.wa-note-meta { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.25rem; }
.wa-note-author { font-size: 0.6875rem; color: #667781; font-weight: 500; }
.wa-note-time { font-size: 0.6875rem; color: #9ca3af; }
.wa-note-content { font-size: 0.8125rem; margin: 0; line-height: 1.4; white-space: pre-wrap; }
.wa-notes-empty { font-size: 0.8125rem; color: #9ca3af; font-style: italic; padding: 0 1rem 1rem; }
.wa-notes-error { font-size: 0.8125rem; color: #ef4444; padding: 0.5rem 1rem 0; }
```

- [ ] **Step 2: Commit.**

---

## Fase E — Docs e verificação

### Task E1: rest/ (.http)

**Files:**
- Modify/Create: arquivo `.http` de whatsapp em `rest/`

- [ ] **Step 1:** acrescentar os 2 endpoints (GET painel, POST criar nota) seguindo o
  formato dos `.http` existentes. **Commit.**

### Task E2: Verificação final (DoD)

- [ ] **Step 1:** `go build ./...` → ok.
- [ ] **Step 2:** `go test ./...` → todos verdes.
- [ ] **Step 3:** cobertura dos pacotes tocados ≥ 80%:
  `go test -cover ./internal/funnel/infrastructure/... ./internal/whatsapp/interfaces/http/...`
- [ ] **Step 4:** lint `golangci-lint run` (ou o alvo do Makefile) → 0 issues; `gofmt`/`goimports` ok.
- [ ] **Step 5:** atualizar `status.md` (commits + agentes) e abrir PR.

---

## Notas de cobertura do spec

- Escopo (ver+adicionar): C2 (handlers) + D (frontend).
- Layout drawer: D1/D2/D3.
- Cross-sell (lead atual): A1 (query) + B2 (adapter).
- Isolamento de tenant: A1 (filtro) + C3 (OWASP).
- Anti-injection: C3 (XSS) — `html/template` auto-escapa.
- Sem migration: nenhuma task de schema (reuso de `lead_notes`).
- Observabilidade: `CreateLeadNoteUseCase` já abre span; logs com zap nos handlers.
