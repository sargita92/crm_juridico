# F07 Step 7 — Lead Detail Drawer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the centered modal with a right-side drawer (50% width) showing enriched lead details: contact info, funnel position, WhatsApp messages, manual notes, movement history, and placeholders for future features.

**Architecture:** DDD + Clean Architecture. Cross-module data (WhatsApp contacts/messages) accessed via interfaces defined in funnel domain (`ContactProvider`, `MessageProvider`), implemented by adapters in funnel infrastructure that wrap WhatsApp repositories. New `LeadNote` entity for manual annotations. TDD throughout.

**Tech Stack:** Go, Gin, Gorm, HTMX, Go html/template, MySQL, testify

---

## File Structure

### New Files
- `internal/funnel/domain/lead_note.go` — LeadNote entity + constructor
- `internal/funnel/domain/lead_note_test.go` — LeadNote validation tests
- `internal/funnel/domain/providers.go` — ContactProvider, MessageProvider interfaces
- `internal/funnel/infrastructure/gorm_lead_note_repository.go` — Gorm implementation
- `internal/funnel/infrastructure/whatsapp_contact_adapter.go` — ContactProvider adapter
- `internal/funnel/infrastructure/whatsapp_message_adapter.go` — MessageProvider adapter
- `internal/funnel/application/create_lead_note.go` — CreateLeadNote use case
- `internal/funnel/application/create_lead_note_test.go` — Tests
- `web/templates/funnel/lead_drawer.html` — Drawer template (replaces lead_detail.html)
- `web/templates/funnel/lead_notes_section.html` — Notes partial for HTMX swap
- `migrations/000023_create_lead_notes_table.up.sql`
- `migrations/000023_create_lead_notes_table.down.sql`

### Modified Files
- `internal/funnel/domain/errors.go` — New error sentinels for LeadNote
- `internal/funnel/domain/repository.go` — Add LeadNoteRepository interface
- `internal/funnel/infrastructure/models.go` — Add leadNoteModel + mappers
- `internal/funnel/application/get_lead_detail.go` — Enrich with providers + notes
- `internal/funnel/application/get_lead_detail_test.go` — Update tests for enriched output
- `internal/funnel/application/mocks_test.go` — Add mocks for new interfaces
- `internal/funnel/interfaces/http/handler.go` — Update RenderLeadDetail, add HandleCreateNote
- `internal/funnel/interfaces/http/routes.go` — Add notes route
- `internal/funnel/module.go` — Wire new dependencies, accept providers
- `internal/whatsapp/module.go` — Expose ContactRepo() and MessageRepo()
- `cmd/api/main.go` — Wire cross-module adapters
- `web/static/css/kanban.css` — Drawer styles
- `web/static/js/kanban.js` — Drawer close behavior
- `web/templates/funnel/lead_detail.html` — Remove (replaced by lead_drawer.html)

---

### Task 1: Domain — LeadNote entity

**Files:**
- Create: `internal/funnel/domain/lead_note.go`
- Create: `internal/funnel/domain/lead_note_test.go`
- Modify: `internal/funnel/domain/errors.go`

- [ ] **Step 1: Add error sentinels**

In `internal/funnel/domain/errors.go`, add at the end:

```go
// Lead notes
var ErrNoteContentRequired = errors.New("note content is required")
var ErrNoteContentTooLong  = errors.New("note content exceeds 2000 characters")
var ErrNoteCreatedByRequired = errors.New("note created_by is required")
```

- [ ] **Step 2: Write failing tests**

Create `internal/funnel/domain/lead_note_test.go`:

```go
package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLeadNote_Success(t *testing.T) {
	note, err := NewLeadNote(uuid.New().String(), "lead-1", "tenant-1", "Ligar amanha", "user-1")
	require.NoError(t, err)
	assert.Equal(t, "lead-1", note.LeadID)
	assert.Equal(t, "tenant-1", note.TenantID)
	assert.Equal(t, "Ligar amanha", note.Content)
	assert.Equal(t, "user-1", note.CreatedBy)
	assert.False(t, note.CreatedAt.IsZero())
}

func TestNewLeadNote_ContentRequired(t *testing.T) {
	_, err := NewLeadNote(uuid.New().String(), "lead-1", "tenant-1", "", "user-1")
	assert.ErrorIs(t, err, ErrNoteContentRequired)
}

func TestNewLeadNote_ContentTooLong(t *testing.T) {
	long := make([]byte, 2001)
	for i := range long {
		long[i] = 'a'
	}
	_, err := NewLeadNote(uuid.New().String(), "lead-1", "tenant-1", string(long), "user-1")
	assert.ErrorIs(t, err, ErrNoteContentTooLong)
}

func TestNewLeadNote_CreatedByRequired(t *testing.T) {
	_, err := NewLeadNote(uuid.New().String(), "lead-1", "tenant-1", "note", "")
	assert.ErrorIs(t, err, ErrNoteCreatedByRequired)
}

func TestNewLeadNote_LeadIDRequired(t *testing.T) {
	_, err := NewLeadNote(uuid.New().String(), "", "tenant-1", "note", "user-1")
	assert.ErrorIs(t, err, ErrLeadNotFound)
}

func TestNewLeadNote_TenantIDRequired(t *testing.T) {
	_, err := NewLeadNote(uuid.New().String(), "lead-1", "", "note", "user-1")
	assert.ErrorIs(t, err, ErrTenantIDRequired)
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/funnel/domain/ -run TestNewLeadNote -v`
Expected: FAIL — `NewLeadNote` not defined

- [ ] **Step 4: Implement LeadNote entity**

Create `internal/funnel/domain/lead_note.go`:

```go
package domain

import "time"

const MaxNoteContentLength = 2000

type LeadNote struct {
	ID        string
	LeadID    string
	TenantID  string
	Content   string
	CreatedBy string
	CreatedAt time.Time
}

func NewLeadNote(id, leadID, tenantID, content, createdBy string) (*LeadNote, error) {
	if leadID == "" {
		return nil, ErrLeadNotFound
	}
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if content == "" {
		return nil, ErrNoteContentRequired
	}
	if len(content) > MaxNoteContentLength {
		return nil, ErrNoteContentTooLong
	}
	if createdBy == "" {
		return nil, ErrNoteCreatedByRequired
	}
	return &LeadNote{
		ID:        id,
		LeadID:    leadID,
		TenantID:  tenantID,
		Content:   content,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/funnel/domain/ -run TestNewLeadNote -v`
Expected: PASS (6 tests)

- [ ] **Step 6: Commit**

```bash
git add internal/funnel/domain/lead_note.go internal/funnel/domain/lead_note_test.go internal/funnel/domain/errors.go
git commit -m "feat(F07): add LeadNote domain entity with validation"
```

---

### Task 2: Domain — Cross-module interfaces + LeadNoteRepository

**Files:**
- Create: `internal/funnel/domain/providers.go`
- Modify: `internal/funnel/domain/repository.go`

- [ ] **Step 1: Create provider interfaces**

Create `internal/funnel/domain/providers.go`:

```go
package domain

import (
	"context"
	"time"
)

type ContactInfo struct {
	Name  string
	Phone string
}

type ContactProvider interface {
	FindByID(ctx context.Context, contactID string) (ContactInfo, error)
}

type MessageSummary struct {
	Direction string
	Content   string
	Timestamp time.Time
}

type MessageProvider interface {
	FindRecentByConversationID(ctx context.Context, conversationID string, limit int) ([]MessageSummary, error)
}
```

- [ ] **Step 2: Add LeadNoteRepository to repository.go**

In `internal/funnel/domain/repository.go`, add at the end (before closing):

```go
type LeadNoteRepository interface {
	Create(ctx context.Context, note *LeadNote) error
	FindByLeadID(ctx context.Context, leadID string) ([]LeadNote, error)
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./internal/funnel/...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add internal/funnel/domain/providers.go internal/funnel/domain/repository.go
git commit -m "feat(F07): add ContactProvider, MessageProvider interfaces and LeadNoteRepository"
```

---

### Task 3: Migration

**Files:**
- Create: `migrations/000023_create_lead_notes_table.up.sql`
- Create: `migrations/000023_create_lead_notes_table.down.sql`

- [ ] **Step 1: Create up migration**

Create `migrations/000023_create_lead_notes_table.up.sql`:

```sql
CREATE TABLE lead_notes (
    id CHAR(36) PRIMARY KEY,
    lead_id CHAR(36) NOT NULL,
    tenant_id CHAR(36) NOT NULL,
    content TEXT NOT NULL,
    created_by CHAR(36) NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE CASCADE,
    INDEX idx_lead_notes_lead_id (lead_id),
    INDEX idx_lead_notes_tenant_id (tenant_id)
);
```

- [ ] **Step 2: Create down migration**

Create `migrations/000023_create_lead_notes_table.down.sql`:

```sql
DROP TABLE IF EXISTS lead_notes;
```

- [ ] **Step 3: Commit**

```bash
git add migrations/000023_create_lead_notes_table.up.sql migrations/000023_create_lead_notes_table.down.sql
git commit -m "feat(F07): add lead_notes migration"
```

---

### Task 4: Infrastructure — LeadNote model + repository

**Files:**
- Modify: `internal/funnel/infrastructure/models.go`
- Create: `internal/funnel/infrastructure/gorm_lead_note_repository.go`

- [ ] **Step 1: Add Gorm model and mappers to models.go**

In `internal/funnel/infrastructure/models.go`, add at the end:

```go
// --- LeadNote model ---

type leadNoteModel struct {
	ID        string    `gorm:"primaryKey;column:id;type:char(36)"`
	LeadID    string    `gorm:"column:lead_id;type:char(36);not null"`
	TenantID  string    `gorm:"column:tenant_id;type:char(36);not null"`
	Content   string    `gorm:"column:content;type:text;not null"`
	CreatedBy string    `gorm:"column:created_by;type:char(36);not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

func (leadNoteModel) TableName() string { return "lead_notes" }

func noteToModel(n *domain.LeadNote) *leadNoteModel {
	return &leadNoteModel{
		ID:        n.ID,
		LeadID:    n.LeadID,
		TenantID:  n.TenantID,
		Content:   n.Content,
		CreatedBy: n.CreatedBy,
		CreatedAt: n.CreatedAt,
	}
}

func noteToDomain(m *leadNoteModel) *domain.LeadNote {
	return &domain.LeadNote{
		ID:        m.ID,
		LeadID:    m.LeadID,
		TenantID:  m.TenantID,
		Content:   m.Content,
		CreatedBy: m.CreatedBy,
		CreatedAt: m.CreatedAt,
	}
}
```

- [ ] **Step 2: Create Gorm repository**

Create `internal/funnel/infrastructure/gorm_lead_note_repository.go`:

```go
package infrastructure

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"gorm.io/gorm"
)

type GormLeadNoteRepository struct {
	db *gorm.DB
}

func NewGormLeadNoteRepository(db *gorm.DB) *GormLeadNoteRepository {
	return &GormLeadNoteRepository{db: db}
}

func (r *GormLeadNoteRepository) Create(ctx context.Context, note *domain.LeadNote) error {
	model := noteToModel(note)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormLeadNoteRepository) FindByLeadID(ctx context.Context, leadID string) ([]domain.LeadNote, error) {
	var models []leadNoteModel
	err := r.db.WithContext(ctx).
		Where("lead_id = ?", leadID).
		Order("created_at DESC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	notes := make([]domain.LeadNote, len(models))
	for i, m := range models {
		notes[i] = *noteToDomain(&m)
	}
	return notes, nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./internal/funnel/...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add internal/funnel/infrastructure/models.go internal/funnel/infrastructure/gorm_lead_note_repository.go
git commit -m "feat(F07): add GormLeadNoteRepository"
```

---

### Task 5: Infrastructure — WhatsApp adapters

**Files:**
- Create: `internal/funnel/infrastructure/whatsapp_contact_adapter.go`
- Create: `internal/funnel/infrastructure/whatsapp_message_adapter.go`

- [ ] **Step 1: Create contact adapter**

Create `internal/funnel/infrastructure/whatsapp_contact_adapter.go`:

```go
package infrastructure

import (
	"context"

	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type WhatsAppContactAdapter struct {
	contactRepo whatsappdomain.ContactRepository
}

func NewWhatsAppContactAdapter(contactRepo whatsappdomain.ContactRepository) *WhatsAppContactAdapter {
	return &WhatsAppContactAdapter{contactRepo: contactRepo}
}

func (a *WhatsAppContactAdapter) FindByID(ctx context.Context, contactID string) (funneldomain.ContactInfo, error) {
	contact, err := a.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return funneldomain.ContactInfo{}, err
	}
	return funneldomain.ContactInfo{
		Name:  contact.Name,
		Phone: contact.Phone,
	}, nil
}
```

- [ ] **Step 2: Create message adapter**

Create `internal/funnel/infrastructure/whatsapp_message_adapter.go`:

```go
package infrastructure

import (
	"context"

	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type WhatsAppMessageAdapter struct {
	messageRepo whatsappdomain.MessageRepository
}

func NewWhatsAppMessageAdapter(messageRepo whatsappdomain.MessageRepository) *WhatsAppMessageAdapter {
	return &WhatsAppMessageAdapter{messageRepo: messageRepo}
}

func (a *WhatsAppMessageAdapter) FindRecentByConversationID(ctx context.Context, conversationID string, limit int) ([]funneldomain.MessageSummary, error) {
	messages, err := a.messageRepo.FindByConversationID(ctx, conversationID, whatsappdomain.MessageFilter{
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	summaries := make([]funneldomain.MessageSummary, len(messages))
	for i, m := range messages {
		summaries[i] = funneldomain.MessageSummary{
			Direction: string(m.Direction),
			Content:   m.Content,
			Timestamp: m.Timestamp,
		}
	}
	return summaries, nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./internal/funnel/...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add internal/funnel/infrastructure/whatsapp_contact_adapter.go internal/funnel/infrastructure/whatsapp_message_adapter.go
git commit -m "feat(F07): add WhatsApp contact and message adapters"
```

---

### Task 6: Application — Add mocks for new interfaces

**Files:**
- Modify: `internal/funnel/application/mocks_test.go`

- [ ] **Step 1: Add mocks for ContactProvider, MessageProvider, and LeadNoteRepository**

In `internal/funnel/application/mocks_test.go`, add at the end:

```go
// --- Mock ContactProvider ---

type mockContactProvider struct {
	contacts map[string]domain.ContactInfo
}

func newMockContactProvider() *mockContactProvider {
	return &mockContactProvider{contacts: make(map[string]domain.ContactInfo)}
}

func (m *mockContactProvider) FindByID(_ context.Context, contactID string) (domain.ContactInfo, error) {
	if info, ok := m.contacts[contactID]; ok {
		return info, nil
	}
	return domain.ContactInfo{}, errors.New("contact not found")
}

// --- Mock MessageProvider ---

type mockMessageProvider struct {
	messages map[string][]domain.MessageSummary
}

func newMockMessageProvider() *mockMessageProvider {
	return &mockMessageProvider{messages: make(map[string][]domain.MessageSummary)}
}

func (m *mockMessageProvider) FindRecentByConversationID(_ context.Context, conversationID string, limit int) ([]domain.MessageSummary, error) {
	msgs := m.messages[conversationID]
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[:limit]
	}
	return msgs, nil
}

// --- Mock LeadNoteRepository ---

type mockLeadNoteRepo struct {
	notes     map[string][]*domain.LeadNote // by leadID
	createErr error
}

func newMockLeadNoteRepo() *mockLeadNoteRepo {
	return &mockLeadNoteRepo{notes: make(map[string][]*domain.LeadNote)}
}

func (m *mockLeadNoteRepo) Create(_ context.Context, note *domain.LeadNote) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.notes[note.LeadID] = append(m.notes[note.LeadID], note)
	return nil
}

func (m *mockLeadNoteRepo) FindByLeadID(_ context.Context, leadID string) ([]domain.LeadNote, error) {
	notes := m.notes[leadID]
	result := make([]domain.LeadNote, len(notes))
	for i, n := range notes {
		result[i] = *n
	}
	return result, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/funnel/application/ -run TestNothing -v 2>&1 | head -5`
Expected: compiles (test not found is OK)

- [ ] **Step 3: Commit**

```bash
git add internal/funnel/application/mocks_test.go
git commit -m "feat(F07): add mocks for ContactProvider, MessageProvider, LeadNoteRepository"
```

---

### Task 7: Application — Refactor GetLeadDetail use case

**Files:**
- Modify: `internal/funnel/application/get_lead_detail.go`
- Modify: `internal/funnel/application/get_lead_detail_test.go`

- [ ] **Step 1: Write updated failing tests**

Replace `internal/funnel/application/get_lead_detail_test.go` with:

```go
package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func setupLeadDetailTest() (*GetLeadDetailUseCase, *mockLeadRepo, *mockLeadMovementRepo, *mockFunnelRepo, *mockColumnRepo, *mockContactProvider, *mockMessageProvider, *mockLeadNoteRepo) {
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	contactProvider := newMockContactProvider()
	messageProvider := newMockMessageProvider()
	noteRepo := newMockLeadNoteRepo()

	uc := NewGetLeadDetailUseCase(leadRepo, movementRepo, funnelRepo, columnRepo, contactProvider, messageProvider, noteRepo)
	return uc, leadRepo, movementRepo, funnelRepo, columnRepo, contactProvider, messageProvider, noteRepo
}

func TestGetLeadDetail_Success_Enriched(t *testing.T) {
	uc, leadRepo, movementRepo, funnelRepo, columnRepo, contactProvider, messageProvider, noteRepo := setupLeadDetailTest()

	// Setup funnel + column
	funnel, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Vendas", "")
	_ = funnelRepo.Create(context.Background(), funnel)
	col, _ := domain.NewColumn(uuid.New().String(), funnel.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), col)

	// Setup lead
	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", funnel.ID, col.ID, "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	// Setup movement
	mv := domain.NewLeadMovement(uuid.New().String(), lead.ID, "", col.ID)
	_ = movementRepo.Create(context.Background(), mv)

	// Setup contact
	contactProvider.contacts["contact-1"] = domain.ContactInfo{Name: "Joao Silva", Phone: "+5511999990000"}

	// Setup messages
	messageProvider.messages["conv-1"] = []domain.MessageSummary{
		{Direction: "incoming", Content: "Ola, preciso de ajuda", Timestamp: time.Now()},
		{Direction: "outgoing", Content: "Como posso ajudar?", Timestamp: time.Now()},
	}

	// Setup note
	note, _ := domain.NewLeadNote(uuid.New().String(), lead.ID, "tenant-1", "Ligar amanha", "user-1")
	_ = noteRepo.Create(context.Background(), note)

	output, err := uc.Execute(context.Background(), GetLeadDetailInput{
		TenantID: "tenant-1", LeadID: lead.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, lead.ID, output.ID)
	assert.Equal(t, "Joao Silva", output.ContactName)
	assert.Equal(t, "+5511999990000", output.ContactPhone)
	assert.Equal(t, "Vendas", output.FunnelName)
	assert.Equal(t, "Novo", output.ColumnName)
	assert.Len(t, output.Messages, 2)
	assert.Equal(t, "incoming", output.Messages[0].Direction)
	assert.Len(t, output.Movements, 1)
	assert.Equal(t, "Novo", output.Movements[0].ToColumnName)
	assert.Len(t, output.Notes, 1)
	assert.Equal(t, "Ligar amanha", output.Notes[0].Content)
}

func TestGetLeadDetail_NotFound(t *testing.T) {
	uc, _, _, _, _, _, _, _ := setupLeadDetailTest()

	_, err := uc.Execute(context.Background(), GetLeadDetailInput{
		TenantID: "tenant-1", LeadID: "nope",
	})
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestGetLeadDetail_WrongTenant(t *testing.T) {
	uc, leadRepo, _, funnelRepo, columnRepo, _, _, _ := setupLeadDetailTest()

	funnel, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Vendas", "")
	_ = funnelRepo.Create(context.Background(), funnel)
	col, _ := domain.NewColumn(uuid.New().String(), funnel.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), col)
	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", funnel.ID, col.ID, "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	_, err := uc.Execute(context.Background(), GetLeadDetailInput{
		TenantID: "other", LeadID: lead.ID,
	})
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/funnel/application/ -run TestGetLeadDetail -v`
Expected: FAIL — constructor signature mismatch

- [ ] **Step 3: Rewrite GetLeadDetail use case**

Replace `internal/funnel/application/get_lead_detail.go` with:

```go
package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type GetLeadDetailInput struct {
	TenantID string
	LeadID   string
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
	Movements       []LeadMovementOutput
	Notes           []LeadNoteOutput
	ProductName     string
	AssignedToName  string
}

type GetLeadDetailUseCase struct {
	leadRepo        domain.LeadRepository
	movementRepo    domain.LeadMovementRepository
	funnelRepo      domain.FunnelRepository
	columnRepo      domain.ColumnRepository
	contactProvider domain.ContactProvider
	messageProvider domain.MessageProvider
	noteRepo        domain.LeadNoteRepository
}

func NewGetLeadDetailUseCase(
	leadRepo domain.LeadRepository,
	movementRepo domain.LeadMovementRepository,
	funnelRepo domain.FunnelRepository,
	columnRepo domain.ColumnRepository,
	contactProvider domain.ContactProvider,
	messageProvider domain.MessageProvider,
	noteRepo domain.LeadNoteRepository,
) *GetLeadDetailUseCase {
	return &GetLeadDetailUseCase{
		leadRepo:        leadRepo,
		movementRepo:    movementRepo,
		funnelRepo:      funnelRepo,
		columnRepo:      columnRepo,
		contactProvider: contactProvider,
		messageProvider: messageProvider,
		noteRepo:        noteRepo,
	}
}

func (uc *GetLeadDetailUseCase) Execute(ctx context.Context, input GetLeadDetailInput) (*LeadDetailOutput, error) {
	lead, err := uc.leadRepo.FindByID(ctx, input.LeadID)
	if err != nil {
		return nil, err
	}
	if lead.TenantID != input.TenantID {
		return nil, domain.ErrLeadNotFound
	}

	// Funnel name
	var funnelName string
	if funnel, err := uc.funnelRepo.FindByID(ctx, lead.FunnelID); err == nil {
		funnelName = funnel.Name
	}

	// Column name
	var columnName string
	if col, err := uc.columnRepo.FindByID(ctx, lead.ColumnID); err == nil {
		columnName = col.Name
	}

	// Contact info
	var contactName, contactPhone string
	if info, err := uc.contactProvider.FindByID(ctx, lead.ContactID); err == nil {
		contactName = info.Name
		contactPhone = info.Phone
	}

	// Messages
	var messages []MessageSummaryOutput
	if msgs, err := uc.messageProvider.FindRecentByConversationID(ctx, lead.ConversationID, 10); err == nil {
		messages = make([]MessageSummaryOutput, len(msgs))
		for i, m := range msgs {
			messages[i] = MessageSummaryOutput{
				Direction: m.Direction,
				Content:   m.Content,
				Timestamp: m.Timestamp,
			}
		}
	}

	// Movements with column names
	movements, err := uc.movementRepo.FindByLeadID(ctx, lead.ID)
	if err != nil {
		return nil, err
	}
	mvOutputs := make([]LeadMovementOutput, len(movements))
	for i, mv := range movements {
		var fromName, toName string
		if mv.FromColumnID != "" {
			if col, err := uc.columnRepo.FindByID(ctx, mv.FromColumnID); err == nil {
				fromName = col.Name
			}
		}
		if col, err := uc.columnRepo.FindByID(ctx, mv.ToColumnID); err == nil {
			toName = col.Name
		}
		mvOutputs[i] = LeadMovementOutput{
			ID:             mv.ID,
			FromColumnID:   mv.FromColumnID,
			FromColumnName: fromName,
			ToColumnID:     mv.ToColumnID,
			ToColumnName:   toName,
			MovedAt:        mv.MovedAt,
		}
	}

	// Notes
	var noteOutputs []LeadNoteOutput
	if notes, err := uc.noteRepo.FindByLeadID(ctx, lead.ID); err == nil {
		noteOutputs = make([]LeadNoteOutput, len(notes))
		for i, n := range notes {
			noteOutputs[i] = LeadNoteOutput{
				ID:        n.ID,
				Content:   n.Content,
				CreatedBy: n.CreatedBy,
				CreatedAt: n.CreatedAt,
			}
		}
	}

	return &LeadDetailOutput{
		ID:              lead.ID,
		TenantID:        lead.TenantID,
		FunnelID:        lead.FunnelID,
		FunnelName:      funnelName,
		ColumnID:        lead.ColumnID,
		ColumnName:      columnName,
		ContactID:       lead.ContactID,
		ContactName:     contactName,
		ContactPhone:    contactPhone,
		ConversationID:  lead.ConversationID,
		Score:           lead.Score,
		Status:          string(lead.Status),
		ColumnEnteredAt: lead.ColumnEnteredAt,
		CreatedAt:       lead.CreatedAt,
		Messages:        messages,
		Movements:       mvOutputs,
		Notes:           noteOutputs,
	}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/funnel/application/ -run TestGetLeadDetail -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Run ALL funnel application tests to check nothing broke**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/funnel/application/ -v`
Expected: Some tests may fail because `NewGetLeadDetailUseCase` signature changed. Fix callers in other test files if needed — any test that calls `NewGetLeadDetailUseCase(leadRepo, movementRepo)` must be updated to pass all 7 params. Check each failing test and update the constructor call.

- [ ] **Step 6: Commit**

```bash
git add internal/funnel/application/get_lead_detail.go internal/funnel/application/get_lead_detail_test.go
git commit -m "feat(F07): enrich GetLeadDetail with contact, messages, funnel names, notes"
```

---

### Task 8: Application — CreateLeadNote use case

**Files:**
- Create: `internal/funnel/application/create_lead_note.go`
- Create: `internal/funnel/application/create_lead_note_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/funnel/application/create_lead_note_test.go`:

```go
package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func TestCreateLeadNote_Success(t *testing.T) {
	leadRepo := newMockLeadRepo()
	noteRepo := newMockLeadNoteRepo()
	uc := NewCreateLeadNoteUseCase(leadRepo, noteRepo)

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", "funnel-1", "col-1", "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	output, err := uc.Execute(context.Background(), CreateLeadNoteInput{
		TenantID:  "tenant-1",
		LeadID:    lead.ID,
		Content:   "Ligar amanha",
		CreatedBy: "user-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "Ligar amanha", output.Content)
	assert.Equal(t, "user-1", output.CreatedBy)
	assert.NotEmpty(t, output.ID)
}

func TestCreateLeadNote_LeadNotFound(t *testing.T) {
	leadRepo := newMockLeadRepo()
	noteRepo := newMockLeadNoteRepo()
	uc := NewCreateLeadNoteUseCase(leadRepo, noteRepo)

	_, err := uc.Execute(context.Background(), CreateLeadNoteInput{
		TenantID:  "tenant-1",
		LeadID:    "nope",
		Content:   "Ligar amanha",
		CreatedBy: "user-1",
	})

	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestCreateLeadNote_WrongTenant(t *testing.T) {
	leadRepo := newMockLeadRepo()
	noteRepo := newMockLeadNoteRepo()
	uc := NewCreateLeadNoteUseCase(leadRepo, noteRepo)

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", "funnel-1", "col-1", "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	_, err := uc.Execute(context.Background(), CreateLeadNoteInput{
		TenantID:  "other-tenant",
		LeadID:    lead.ID,
		Content:   "Ligar amanha",
		CreatedBy: "user-1",
	})

	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestCreateLeadNote_EmptyContent(t *testing.T) {
	leadRepo := newMockLeadRepo()
	noteRepo := newMockLeadNoteRepo()
	uc := NewCreateLeadNoteUseCase(leadRepo, noteRepo)

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", "funnel-1", "col-1", "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	_, err := uc.Execute(context.Background(), CreateLeadNoteInput{
		TenantID:  "tenant-1",
		LeadID:    lead.ID,
		Content:   "",
		CreatedBy: "user-1",
	})

	assert.ErrorIs(t, err, domain.ErrNoteContentRequired)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/funnel/application/ -run TestCreateLeadNote -v`
Expected: FAIL — `NewCreateLeadNoteUseCase` not defined

- [ ] **Step 3: Implement CreateLeadNote use case**

Create `internal/funnel/application/create_lead_note.go`:

```go
package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type CreateLeadNoteInput struct {
	TenantID  string
	LeadID    string
	Content   string
	CreatedBy string
}

type CreateLeadNoteUseCase struct {
	leadRepo domain.LeadRepository
	noteRepo domain.LeadNoteRepository
}

func NewCreateLeadNoteUseCase(leadRepo domain.LeadRepository, noteRepo domain.LeadNoteRepository) *CreateLeadNoteUseCase {
	return &CreateLeadNoteUseCase{leadRepo: leadRepo, noteRepo: noteRepo}
}

func (uc *CreateLeadNoteUseCase) Execute(ctx context.Context, input CreateLeadNoteInput) (*LeadNoteOutput, error) {
	lead, err := uc.leadRepo.FindByID(ctx, input.LeadID)
	if err != nil {
		return nil, err
	}
	if lead.TenantID != input.TenantID {
		return nil, domain.ErrLeadNotFound
	}

	note, err := domain.NewLeadNote(uuid.New().String(), lead.ID, lead.TenantID, input.Content, input.CreatedBy)
	if err != nil {
		return nil, err
	}

	if err := uc.noteRepo.Create(ctx, note); err != nil {
		return nil, err
	}

	return &LeadNoteOutput{
		ID:        note.ID,
		Content:   note.Content,
		CreatedBy: note.CreatedBy,
		CreatedAt: note.CreatedAt,
	}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/funnel/application/ -run TestCreateLeadNote -v`
Expected: PASS (4 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/funnel/application/create_lead_note.go internal/funnel/application/create_lead_note_test.go
git commit -m "feat(F07): add CreateLeadNote use case with TDD"
```

---

### Task 9: WhatsApp module — Expose repositories

**Files:**
- Modify: `internal/whatsapp/module.go`

- [ ] **Step 1: Expose contact and message repositories**

In `internal/whatsapp/module.go`, add fields to the Module struct and getter methods.

Change the struct to:

```go
type Module struct {
	handler          *whatsapphttp.Handler
	provider         domain.WhatsAppProvider
	receiveMessageUC *application.ReceiveMessageUseCase
	contactRepo      domain.ContactRepository
	messageRepo      domain.MessageRepository
}
```

In `NewModule`, store the repos before returning:

Replace the return statement to include the new fields:
```go
return &Module{handler: handler, provider: provider, receiveMessageUC: receiveMessageUC, contactRepo: contactRepo, messageRepo: messageRepo}
```

Add getter methods at the end of the file:

```go
func (m *Module) ContactRepo() domain.ContactRepository {
	return m.contactRepo
}

func (m *Module) MessageRepo() domain.MessageRepository {
	return m.messageRepo
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./internal/whatsapp/...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add internal/whatsapp/module.go
git commit -m "feat(F07): expose ContactRepo and MessageRepo from WhatsApp module"
```

---

### Task 10: HTTP — Update handler and routes

**Files:**
- Modify: `internal/funnel/interfaces/http/handler.go`
- Modify: `internal/funnel/interfaces/http/routes.go`

- [ ] **Step 1: Add createLeadNoteUC to handler struct and constructor**

In `internal/funnel/interfaces/http/handler.go`, add field to Handler struct:

```go
createLeadNoteUC *application.CreateLeadNoteUseCase
```

Update `NewHandler` to accept the new param — add it after `getLeadDetailUC`:

```go
func NewHandler(
	getKanbanUC *application.GetKanbanUseCase,
	listFunnelsUC *application.ListFunnelsUseCase,
	getFunnelUC *application.GetFunnelUseCase,
	createFunnelUC *application.CreateFunnelUseCase,
	updateFunnelUC *application.UpdateFunnelUseCase,
	toggleFunnelUC *application.ToggleFunnelUseCase,
	createColumnUC *application.CreateColumnUseCase,
	deleteColumnUC *application.DeleteColumnUseCase,
	moveColumnUC *application.MoveColumnUseCase,
	createLeadUC *application.CreateLeadUseCase,
	moveLeadUC *application.MoveLeadUseCase,
	getLeadDetailUC *application.GetLeadDetailUseCase,
	createLeadNoteUC *application.CreateLeadNoteUseCase,
	leadRepo domain.LeadRepository,
	log *zap.Logger,
) *Handler {
```

And in the return, add `createLeadNoteUC: createLeadNoteUC`.

- [ ] **Step 2: Update RenderLeadDetail to use drawer template**

Replace the `RenderLeadDetail` method body. Change the template from `"funnel/lead_detail.html"` to `"funnel/lead_drawer.html"` and pass the `Lead` data at the top level (not nested):

```go
func (h *Handler) RenderLeadDetail(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	leadID := c.Param("id")

	detail, err := h.getLeadDetailUC.Execute(c.Request.Context(), application.GetLeadDetailInput{
		TenantID: tenantID,
		LeadID:   leadID,
	})
	if err != nil {
		h.log.Error("failed to get lead detail", zap.Error(err))
		c.HTML(http.StatusNotFound, "funnel/lead_drawer.html", gin.H{
			"Error": "Lead nao encontrado",
		})
		return
	}

	c.HTML(http.StatusOK, "funnel/lead_drawer.html", gin.H{
		"Lead": detail,
	})
}
```

- [ ] **Step 3: Add HandleCreateNote method**

Add at the end of the handler file (before any closing braces), after the existing lead methods:

```go
func (h *Handler) HandleCreateNote(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	leadID := c.Param("id")
	content := c.PostForm("content")
	userID := c.GetString("user_id")

	_, err := h.createLeadNoteUC.Execute(c.Request.Context(), application.CreateLeadNoteInput{
		TenantID:  tenantID,
		LeadID:    leadID,
		Content:   content,
		CreatedBy: userID,
	})
	if err != nil {
		h.log.Error("failed to create note", zap.Error(err))
		c.HTML(http.StatusUnprocessableEntity, "funnel/lead_notes_section.html", gin.H{
			"Error": "Erro ao adicionar anotacao: " + err.Error(),
		})
		return
	}

	// Re-fetch lead detail to get updated notes list
	detail, err := h.getLeadDetailUC.Execute(c.Request.Context(), application.GetLeadDetailInput{
		TenantID: tenantID,
		LeadID:   leadID,
	})
	if err != nil {
		h.log.Error("failed to reload lead detail", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.HTML(http.StatusOK, "funnel/lead_notes_section.html", gin.H{
		"Lead": detail,
	})
}
```

- [ ] **Step 4: Add notes route**

In `internal/funnel/interfaces/http/routes.go`, add after the existing lead routes:

```go
tenant.POST("/:id/notes", h.HandleCreateNote)
```

- [ ] **Step 5: Verify compilation**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./internal/funnel/...`
Expected: FAIL — `module.go` still passes old constructor. That's OK, we'll fix it in Task 12.

- [ ] **Step 6: Commit**

```bash
git add internal/funnel/interfaces/http/handler.go internal/funnel/interfaces/http/routes.go
git commit -m "feat(F07): add HandleCreateNote and update handler for drawer"
```

---

### Task 11: Module wiring

**Files:**
- Modify: `internal/funnel/module.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Update funnel module to accept providers**

Replace `internal/funnel/module.go`:

```go
package funnel

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/funnel/application"
	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/funnel/infrastructure"
	funnelhttp "github.com/sasrgita/crm-juridico/internal/funnel/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

type Module struct {
	handler     *funnelhttp.Handler
	leadCreator *application.CreateLeadUseCase
}

func NewModule(db *gorm.DB, contactProvider domain.ContactProvider, messageProvider domain.MessageProvider, log *zap.Logger) *Module {
	funnelRepo := infrastructure.NewGormFunnelRepository(db)
	columnRepo := infrastructure.NewGormColumnRepository(db)
	leadRepo := infrastructure.NewGormLeadRepository(db)
	movementRepo := infrastructure.NewGormLeadMovementRepository(db)
	noteRepo := infrastructure.NewGormLeadNoteRepository(db)

	// Use cases
	getKanbanUC := application.NewGetKanbanUseCase(funnelRepo, columnRepo, leadRepo)
	listFunnelsUC := application.NewListFunnelsUseCase(funnelRepo, columnRepo, leadRepo)
	getFunnelUC := application.NewGetFunnelUseCase(funnelRepo, columnRepo)
	createFunnelUC := application.NewCreateFunnelUseCase(funnelRepo, columnRepo)
	updateFunnelUC := application.NewUpdateFunnelUseCase(funnelRepo)
	toggleFunnelUC := application.NewToggleFunnelUseCase(funnelRepo)
	createColumnUC := application.NewCreateColumnUseCase(funnelRepo, columnRepo)
	deleteColumnUC := application.NewDeleteColumnUseCase(funnelRepo, columnRepo, leadRepo)
	moveColumnUC := application.NewMoveColumnUseCase(funnelRepo, columnRepo)
	createLeadUC := application.NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo)
	moveLeadUC := application.NewMoveLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo)
	getLeadDetailUC := application.NewGetLeadDetailUseCase(leadRepo, movementRepo, funnelRepo, columnRepo, contactProvider, messageProvider, noteRepo)
	createLeadNoteUC := application.NewCreateLeadNoteUseCase(leadRepo, noteRepo)

	handler := funnelhttp.NewHandler(
		getKanbanUC, listFunnelsUC, getFunnelUC,
		createFunnelUC, updateFunnelUC, toggleFunnelUC,
		createColumnUC, deleteColumnUC, moveColumnUC,
		createLeadUC, moveLeadUC, getLeadDetailUC,
		createLeadNoteUC,
		leadRepo, log,
	)

	return &Module{
		handler:     handler,
		leadCreator: createLeadUC,
	}
}

func (m *Module) Name() string { return "funnel" }

func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant)
}

func (m *Module) LeadCreator() *application.CreateLeadUseCase {
	return m.leadCreator
}
```

- [ ] **Step 2: Update main.go wiring**

In `cmd/api/main.go`, find the line:

```go
funnelMod := funnel.NewModule(db, log)
```

Replace with:

```go
// Cross-module adapters
contactAdapter := funnelinfra.NewWhatsAppContactAdapter(whatsappMod.ContactRepo())
messageAdapter := funnelinfra.NewWhatsAppMessageAdapter(whatsappMod.MessageRepo())

funnelMod := funnel.NewModule(db, contactAdapter, messageAdapter, log)
```

This requires moving `whatsappMod` creation BEFORE `funnelMod`. The order becomes:

```go
whatsappMod := whatsapp.NewModule(db, whatsmeowProvider, log)

// Cross-module adapters
contactAdapter := funnelinfra.NewWhatsAppContactAdapter(whatsappMod.ContactRepo())
messageAdapter := funnelinfra.NewWhatsAppMessageAdapter(whatsappMod.MessageRepo())

funnelMod := funnel.NewModule(db, contactAdapter, messageAdapter, log)
whatsappMod.SetLeadCreator(funnelMod.LeadCreator())
```

Add the import for funnel infrastructure:

```go
funnelinfra "github.com/sasrgita/crm-juridico/internal/funnel/infrastructure"
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./cmd/api/`
Expected: success

- [ ] **Step 4: Run all funnel tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/funnel/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/funnel/module.go cmd/api/main.go
git commit -m "feat(F07): wire cross-module adapters and LeadNote in module"
```

---

### Task 12: Frontend — Drawer template

**Files:**
- Create: `web/templates/funnel/lead_drawer.html`
- Create: `web/templates/funnel/lead_notes_section.html`
- Delete: `web/templates/funnel/lead_detail.html` (replaced)

- [ ] **Step 1: Create drawer template**

Create `web/templates/funnel/lead_drawer.html`:

```html
{{define "funnel/lead_drawer.html"}}
<div class="lead-drawer-overlay" onclick="closeLeadDrawer(event)">
    <div class="lead-drawer" onclick="event.stopPropagation()">
        <div class="lead-drawer-header">
            <h3>{{if .Lead}}{{.Lead.ContactName}}{{if not .Lead.ContactName}}Lead{{end}}{{else}}Erro{{end}}</h3>
            <button class="modal-close" onclick="document.getElementById('lead-modal').innerHTML=''">&times;</button>
        </div>
        {{if .Error}}
        <div class="lead-drawer-body">
            <p style="color:#ef4444">{{.Error}}</p>
        </div>
        {{else}}
        <div class="lead-drawer-body">
            <!-- Contato -->
            <div class="lead-detail-section">
                <h4>Contato</h4>
                <p><strong>Nome:</strong> {{.Lead.ContactName}}</p>
                <p><strong>Telefone:</strong> {{.Lead.ContactPhone}}</p>
            </div>

            <!-- Funil -->
            <div class="lead-detail-section">
                <h4>Funil</h4>
                <p><strong>Funil:</strong> {{.Lead.FunnelName}}</p>
                <p><strong>Coluna:</strong> {{.Lead.ColumnName}}</p>
                <p><strong>Score:</strong> <span style="color:#eab308;font-weight:600">&#9733; {{.Lead.Score}}</span></p>
                <p><strong>Status:</strong> <span class="badge {{if eq .Lead.Status "won"}}badge-green{{else if eq .Lead.Status "lost"}}badge-gray{{else}}badge-blue{{end}}">{{.Lead.Status}}</span></p>
                <p><strong>Criado em:</strong> {{.Lead.CreatedAt.Format "02/01/2006 15:04"}}</p>
            </div>

            <!-- Acoes -->
            <div class="lead-detail-actions">
                <a href="/tenant/whatsapp" class="btn btn-primary btn-sm">Abrir Conversa</a>
                <button class="btn btn-outline btn-sm"
                        hx-get="/tenant/leads/{{.Lead.ID}}/move-form"
                        hx-target="#lead-modal"
                        hx-swap="innerHTML">Mover Lead</button>
            </div>

            <!-- Mensagens WhatsApp -->
            <div class="lead-detail-section">
                <h4>Mensagens recentes</h4>
                {{if .Lead.Messages}}
                <div class="lead-messages">
                    {{range .Lead.Messages}}
                    <div class="lead-message-bubble lead-message-{{.Direction}}">
                        <p>{{.Content}}</p>
                        <span class="lead-message-time">{{.Timestamp.Format "02/01 15:04"}}</span>
                    </div>
                    {{end}}
                </div>
                <a href="/tenant/whatsapp" class="lead-messages-link">Ver conversa completa &rarr;</a>
                {{else}}
                <p class="lead-empty-text">Nenhuma mensagem</p>
                {{end}}
            </div>

            <!-- Anotacoes -->
            <div class="lead-detail-section" id="lead-notes-section">
                {{template "funnel/lead_notes_section.html" .}}
            </div>

            <!-- Historico de movimentacoes -->
            <div class="lead-detail-section">
                <h4>Historico de movimentacoes</h4>
                {{if .Lead.Movements}}
                <div class="lead-movements">
                    {{range .Lead.Movements}}
                    <div class="lead-movement-item">
                        <span class="lead-movement-date">{{.MovedAt.Format "02/01 15:04"}}</span>
                        <span class="lead-movement-desc">
                            {{if .FromColumnName}}{{.FromColumnName}} &rarr; {{end}}{{.ToColumnName}}
                            {{if not .FromColumnName}}(criacao automatica){{end}}
                        </span>
                    </div>
                    {{end}}
                </div>
                {{else}}
                <p class="lead-empty-text">Nenhuma movimentacao</p>
                {{end}}
            </div>

            <!-- Produto (placeholder F10) -->
            <div class="lead-detail-section">
                <h4>Produto</h4>
                {{if .Lead.ProductName}}
                <p>{{.Lead.ProductName}}</p>
                {{else}}
                <p class="lead-empty-text">Nenhum produto associado</p>
                {{end}}
            </div>

            <!-- Responsavel (placeholder F08) -->
            <div class="lead-detail-section">
                <h4>Responsavel</h4>
                {{if .Lead.AssignedToName}}
                <p>{{.Lead.AssignedToName}}</p>
                {{else}}
                <p class="lead-empty-text">Nenhum responsavel atribuido</p>
                {{end}}
            </div>

            <!-- Documentos (placeholder F14) -->
            <div class="lead-detail-section">
                <h4>Documentos</h4>
                <p class="lead-empty-text">Nenhum documento anexado</p>
            </div>
        </div>
        {{end}}
    </div>
</div>
{{end}}
```

- [ ] **Step 2: Create notes section partial**

Create `web/templates/funnel/lead_notes_section.html`:

```html
{{define "funnel/lead_notes_section.html"}}
<h4>Anotacoes</h4>
{{if .Error}}
<p style="color:#ef4444;font-size:0.8125rem;margin-bottom:0.5rem">{{.Error}}</p>
{{end}}
{{if .Lead}}
<form class="lead-note-form"
      hx-post="/tenant/leads/{{.Lead.ID}}/notes"
      hx-target="#lead-notes-section"
      hx-swap="innerHTML">
    <textarea name="content" placeholder="Adicionar anotacao..." rows="2" class="lead-note-input" required></textarea>
    <button type="submit" class="btn btn-sm btn-primary">Adicionar</button>
</form>
{{if .Lead.Notes}}
<div class="lead-notes-list">
    {{range .Lead.Notes}}
    <div class="lead-note-item">
        <div class="lead-note-meta">
            <span class="lead-note-date">{{.CreatedAt.Format "02/01/2006 15:04"}}</span>
        </div>
        <p class="lead-note-content">{{.Content}}</p>
    </div>
    {{end}}
</div>
{{else}}
<p class="lead-empty-text">Nenhuma anotacao</p>
{{end}}
{{end}}
{{end}}
```

- [ ] **Step 3: Delete old lead_detail.html**

```bash
rm web/templates/funnel/lead_detail.html
```

- [ ] **Step 4: Commit**

```bash
git add web/templates/funnel/lead_drawer.html web/templates/funnel/lead_notes_section.html
git rm web/templates/funnel/lead_detail.html
git commit -m "feat(F07): replace lead modal with drawer template + notes section"
```

---

### Task 13: Frontend — CSS

**Files:**
- Modify: `web/static/css/kanban.css`

- [ ] **Step 1: Replace modal styles with drawer styles**

In `web/static/css/kanban.css`, replace the `/* --- Modal --- */` section and `/* --- Lead detail --- */` section. Keep existing modal classes (they're used by funnel forms) and ADD the drawer classes.

Add these new classes after the existing `/* --- Lead detail --- */` section:

```css
/* --- Lead Drawer --- */
.lead-drawer-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.5);
    z-index: 100;
    animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
}

.lead-drawer {
    position: fixed;
    right: 0;
    top: 0;
    bottom: 0;
    width: 50%;
    background: #fff;
    box-shadow: -4px 0 16px rgba(0,0,0,0.1);
    display: flex;
    flex-direction: column;
    animation: slideIn 0.25s ease;
    z-index: 101;
}

@keyframes slideIn {
    from { transform: translateX(100%); }
    to { transform: translateX(0); }
}

.lead-drawer-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.25rem;
    border-bottom: 1px solid #e5e7eb;
    flex-shrink: 0;
}

.lead-drawer-header h3 {
    font-size: 1.1rem;
    margin: 0;
}

.lead-drawer-body {
    flex: 1;
    overflow-y: auto;
    padding: 1.25rem;
}

/* --- Lead Messages --- */
.lead-messages {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    max-height: 320px;
    overflow-y: auto;
    padding: 0.5rem;
    background: #f0f2f5;
    border-radius: 8px;
}

.lead-message-bubble {
    max-width: 80%;
    padding: 0.5rem 0.75rem;
    border-radius: 8px;
    font-size: 0.8125rem;
    line-height: 1.4;
}

.lead-message-bubble p {
    margin: 0;
}

.lead-message-incoming {
    background: #fff;
    align-self: flex-start;
    border-top-left-radius: 2px;
}

.lead-message-outgoing {
    background: #d9fdd3;
    align-self: flex-end;
    border-top-right-radius: 2px;
}

.lead-message-time {
    display: block;
    font-size: 0.6875rem;
    color: #9ca3af;
    margin-top: 0.125rem;
    text-align: right;
}

.lead-messages-link {
    display: block;
    font-size: 0.8125rem;
    color: #3b82f6;
    margin-top: 0.5rem;
    text-decoration: none;
}

.lead-messages-link:hover {
    text-decoration: underline;
}

/* --- Lead Notes --- */
.lead-note-form {
    display: flex;
    gap: 0.5rem;
    align-items: flex-start;
    margin-bottom: 0.75rem;
}

.lead-note-input {
    flex: 1;
    padding: 0.5rem 0.75rem;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 0.8125rem;
    resize: vertical;
    min-height: 2.5rem;
    font-family: inherit;
}

.lead-notes-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
}

.lead-note-item {
    padding: 0.5rem 0.75rem;
    background: #fefce8;
    border-radius: 6px;
    border-left: 3px solid #eab308;
}

.lead-note-meta {
    margin-bottom: 0.25rem;
}

.lead-note-date {
    font-size: 0.6875rem;
    color: #9ca3af;
}

.lead-note-content {
    font-size: 0.8125rem;
    margin: 0;
    line-height: 1.4;
}

/* --- Lead Empty Text --- */
.lead-empty-text {
    font-size: 0.8125rem;
    color: #9ca3af;
    font-style: italic;
}

/* --- Drawer Responsive --- */
@media (max-width: 768px) {
    .lead-drawer {
        width: 100%;
    }
}
```

- [ ] **Step 2: Commit**

```bash
git add web/static/css/kanban.css
git commit -m "feat(F07): add drawer CSS styles with messages and notes"
```

---

### Task 14: Frontend — JavaScript

**Files:**
- Modify: `web/static/js/kanban.js`

- [ ] **Step 1: Add drawer close function and Escape key handler**

In `web/static/js/kanban.js`, replace the `closeLeadModal` function with:

```javascript
function closeLeadDrawer(event) {
    if (event.target === event.currentTarget) {
        document.getElementById('lead-modal').innerHTML = '';
    }
}

// Keep backward compat
function closeLeadModal(event) {
    closeLeadDrawer(event);
}
```

Add Escape key handler (if not already present from admin.js — check admin.js first; if admin.js handles it generically for `#lead-modal`, this step can be skipped). Add after the close functions:

```javascript
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
        var modal = document.getElementById('lead-modal');
        if (modal && modal.innerHTML.trim() !== '') {
            modal.innerHTML = '';
        }
    }
});
```

- [ ] **Step 2: Commit**

```bash
git add web/static/js/kanban.js
git commit -m "feat(F07): add drawer close behavior with Escape key"
```

---

### Task 15: Verification — Build + Tests

- [ ] **Step 1: Run full build**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./...`
Expected: success

- [ ] **Step 2: Run all funnel tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/funnel/... -v`
Expected: all pass

- [ ] **Step 3: Run all project tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./... -count=1 2>&1 | tail -30`
Expected: all pass (or only unrelated failures)

- [ ] **Step 4: Check test coverage**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/funnel/... -coverprofile=coverage.out && go tool cover -func=coverage.out | grep total`
Expected: coverage >= 80%

- [ ] **Step 5: If coverage < 80%, add tests for uncovered code before proceeding**

- [ ] **Step 6: Update F07 feature doc**

In `docs/features/F07-funis-kanban.md`, mark Step 7 items as done:

```markdown
### Step 7: Painel de detalhes do lead
- [x] ao clicar no card, abrir painel lateral ou modal com:
  - [x] dados do contato (nome, telefone)
  - [x] funil e coluna atual, score
  - [x] conversa do WhatsApp embutida (últimas mensagens + link para abrir)
  - [x] histórico de movimentações no funil
  - [ ] documentos/arquivos do lead (F14)
  - [ ] produto associado (F10)
  - [ ] responsável atribuído (F08)
  - [x] anotações manuais
  - [x] botão "Mover Lead" (funil + coluna)
  - [x] botão "Abrir Conversa" (navega para WhatsApp)
- [x] design: painel lateral estilo drawer (abre da direita, não bloqueia o kanban)
```

- [ ] **Step 7: Commit**

```bash
git add docs/features/F07-funis-kanban.md
git commit -m "docs(F07): mark Step 7 items as completed"
```

---

### Task 16: Update rest/ HTTP files

**Files:**
- Modify or create: `rest/funnel.http`

- [ ] **Step 1: Add lead detail and notes endpoints**

Add to `rest/funnel.http` (create if not exists):

```http
### Get Lead Detail (drawer)
GET {{host}}/tenant/leads/{{lead_id}}
Cookie: token={{token}}

### Create Lead Note
POST {{host}}/tenant/leads/{{lead_id}}/notes
Cookie: token={{token}}
Content-Type: application/x-www-form-urlencoded

content=Ligar amanha para agendar reuniao
```

- [ ] **Step 2: Commit**

```bash
git add rest/
git commit -m "docs(F07): add lead detail and notes endpoints to rest files"
```

---

### Task 17: OWASP tests

**Files:**
- Create: `internal/funnel/interfaces/http/owasp_test.go`

- [ ] **Step 1: Write OWASP tests**

Create `internal/funnel/interfaces/http/owasp_test.go` following the WhatsApp OWASP pattern:

```go
package http

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/funnel/application"
	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

type owaspEnv struct {
	router      *gin.Engine
	provider    *authinfra.JWTProvider
	leadRepo    *owaspMockLeadRepo
	funnelRepo  *owaspMockFunnelRepo
	columnRepo  *owaspMockColumnRepo
	noteRepo    *owaspMockNoteRepo
}

// Minimal mocks for OWASP tests (only what's needed for routing)

type owaspMockFunnelRepo struct{ funnels map[string]*domain.Funnel }
func (m *owaspMockFunnelRepo) Create(_ context.Context, f *domain.Funnel) error { m.funnels[f.ID] = f; return nil }
func (m *owaspMockFunnelRepo) FindByID(_ context.Context, id string) (*domain.Funnel, error) {
	if f, ok := m.funnels[id]; ok { return f, nil }; return nil, domain.ErrFunnelNotFound
}
func (m *owaspMockFunnelRepo) Update(_ context.Context, f *domain.Funnel) error { return nil }
func (m *owaspMockFunnelRepo) FindByTenantID(_ context.Context, tenantID string) ([]domain.Funnel, error) { return nil, nil }
func (m *owaspMockFunnelRepo) FindDefaultByTenantID(_ context.Context, tenantID string) (*domain.Funnel, error) { return nil, domain.ErrFunnelNotFound }

type owaspMockColumnRepo struct{ columns map[string]*domain.Column }
func (m *owaspMockColumnRepo) Create(_ context.Context, c *domain.Column) error { m.columns[c.ID] = c; return nil }
func (m *owaspMockColumnRepo) FindByID(_ context.Context, id string) (*domain.Column, error) {
	if c, ok := m.columns[id]; ok { return c, nil }; return nil, domain.ErrColumnNotFound
}
func (m *owaspMockColumnRepo) Update(_ context.Context, c *domain.Column) error { return nil }
func (m *owaspMockColumnRepo) Delete(_ context.Context, id string) error { return nil }
func (m *owaspMockColumnRepo) FindByFunnelID(_ context.Context, funnelID string) ([]domain.Column, error) { return nil, nil }
func (m *owaspMockColumnRepo) FindEntryByFunnelID(_ context.Context, funnelID string) (*domain.Column, error) { return nil, domain.ErrColumnNotFound }
func (m *owaspMockColumnRepo) CountByFunnelID(_ context.Context, funnelID string) (int, error) { return 0, nil }
func (m *owaspMockColumnRepo) GetMaxOrderIndex(_ context.Context, funnelID string) (int, error) { return 0, nil }
func (m *owaspMockColumnRepo) SwapOrder(_ context.Context, col1ID string, order1 int, col2ID string, order2 int) error { return nil }

type owaspMockLeadRepo struct{ leads map[string]*domain.Lead }
func (m *owaspMockLeadRepo) Create(_ context.Context, l *domain.Lead) error { m.leads[l.ID] = l; return nil }
func (m *owaspMockLeadRepo) FindByID(_ context.Context, id string) (*domain.Lead, error) {
	if l, ok := m.leads[id]; ok { return l, nil }; return nil, domain.ErrLeadNotFound
}
func (m *owaspMockLeadRepo) Update(_ context.Context, l *domain.Lead) error { return nil }
func (m *owaspMockLeadRepo) FindByContactAndTenant(_ context.Context, tenantID, contactID string) (*domain.Lead, error) { return nil, domain.ErrLeadNotFound }
func (m *owaspMockLeadRepo) FindByFunnelID(_ context.Context, funnelID string, filter domain.LeadFilter) (*domain.LeadList, error) { return &domain.LeadList{}, nil }
func (m *owaspMockLeadRepo) CountByColumnID(_ context.Context, columnID string) (int, error) { return 0, nil }

type owaspMockMovementRepo struct{}
func (m *owaspMockMovementRepo) Create(_ context.Context, mv *domain.LeadMovement) error { return nil }
func (m *owaspMockMovementRepo) FindByLeadID(_ context.Context, leadID string) ([]domain.LeadMovement, error) { return nil, nil }

type owaspMockContactProvider struct{}
func (m *owaspMockContactProvider) FindByID(_ context.Context, id string) (domain.ContactInfo, error) { return domain.ContactInfo{}, nil }

type owaspMockMessageProvider struct{}
func (m *owaspMockMessageProvider) FindRecentByConversationID(_ context.Context, id string, limit int) ([]domain.MessageSummary, error) { return nil, nil }

type owaspMockNoteRepo struct{ notes map[string][]*domain.LeadNote }
func (m *owaspMockNoteRepo) Create(_ context.Context, n *domain.LeadNote) error { m.notes[n.LeadID] = append(m.notes[n.LeadID], n); return nil }
func (m *owaspMockNoteRepo) FindByLeadID(_ context.Context, leadID string) ([]domain.LeadNote, error) { return nil, nil }

func setupOwaspEnv() *owaspEnv {
	gin.SetMode(gin.TestMode)

	funnelRepo := &owaspMockFunnelRepo{funnels: make(map[string]*domain.Funnel)}
	columnRepo := &owaspMockColumnRepo{columns: make(map[string]*domain.Column)}
	leadRepo := &owaspMockLeadRepo{leads: make(map[string]*domain.Lead)}
	movementRepo := &owaspMockMovementRepo{}
	contactProvider := &owaspMockContactProvider{}
	messageProvider := &owaspMockMessageProvider{}
	noteRepo := &owaspMockNoteRepo{notes: make(map[string][]*domain.LeadNote)}

	getKanbanUC := application.NewGetKanbanUseCase(funnelRepo, columnRepo, leadRepo)
	listFunnelsUC := application.NewListFunnelsUseCase(funnelRepo, columnRepo, leadRepo)
	getFunnelUC := application.NewGetFunnelUseCase(funnelRepo, columnRepo)
	createFunnelUC := application.NewCreateFunnelUseCase(funnelRepo, columnRepo)
	updateFunnelUC := application.NewUpdateFunnelUseCase(funnelRepo)
	toggleFunnelUC := application.NewToggleFunnelUseCase(funnelRepo)
	createColumnUC := application.NewCreateColumnUseCase(funnelRepo, columnRepo)
	deleteColumnUC := application.NewDeleteColumnUseCase(funnelRepo, columnRepo, leadRepo)
	moveColumnUC := application.NewMoveColumnUseCase(funnelRepo, columnRepo)
	createLeadUC := application.NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo)
	moveLeadUC := application.NewMoveLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo)
	getLeadDetailUC := application.NewGetLeadDetailUseCase(leadRepo, movementRepo, funnelRepo, columnRepo, contactProvider, messageProvider, noteRepo)
	createLeadNoteUC := application.NewCreateLeadNoteUseCase(leadRepo, noteRepo)

	testLog, _ := zap.NewDevelopment()
	handler := NewHandler(
		getKanbanUC, listFunnelsUC, getFunnelUC,
		createFunnelUC, updateFunnelUC, toggleFunnelUC,
		createColumnUC, deleteColumnUC, moveColumnUC,
		createLeadUC, moveLeadUC, getLeadDetailUC,
		createLeadNoteUC,
		leadRepo, testLog,
	)

	router := gin.New()

	tmpl := template.New("")
	for _, name := range []string{
		"funnel/kanban.html", "funnel/kanban_content.html",
		"funnel/lead_drawer.html", "funnel/lead_notes_section.html",
		"funnel/lead_move.html", "funnel/funnel_list.html",
		"funnel/funnel_detail.html", "funnel/funnel_form.html",
		"funnel/columns_section.html", "funnel/column_form.html",
	} {
		template.Must(tmpl.New(name).Parse("ok"))
	}
	router.SetHTMLTemplate(tmpl)

	jwtProvider := authinfra.NewJWTProvider("test-secret-owasp", 24*time.Hour)
	authMw := middleware.Auth(jwtProvider)
	tenantMw := middleware.RequireTenant()
	handler.RegisterRoutes(router, authMw, tenantMw)

	return &owaspEnv{
		router:     router,
		provider:   jwtProvider,
		leadRepo:   leadRepo,
		funnelRepo: funnelRepo,
		columnRepo: columnRepo,
		noteRepo:   noteRepo,
	}
}

func (e *owaspEnv) tenantToken(t *testing.T, tenantID string) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(), Role: authdomain.UserRoleUser, TenantID: tenantID,
	})
	require.NoError(t, err)
	return token
}

func (e *owaspEnv) tokenWithoutTenant(t *testing.T) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(), Role: authdomain.UserRoleUser,
	})
	require.NoError(t, err)
	return token
}

func owaspCookie(token string) *http.Cookie {
	return &http.Cookie{Name: "token", Value: token}
}

func TestOWASP_A01_NoToken_Returns401(t *testing.T) {
	env := setupOwaspEnv()
	routes := []struct{ method, path string }{
		{http.MethodGet, "/tenant/leads"},
		{http.MethodGet, "/tenant/leads/some-id"},
		{http.MethodPost, "/tenant/leads/some-id/notes"},
		{http.MethodPost, "/tenant/leads/some-id/move"},
	}
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(r.method, r.path, nil)
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestOWASP_A01_NoTenant_Returns403(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tokenWithoutTenant(t)
	routes := []struct{ method, path string }{
		{http.MethodGet, "/tenant/leads"},
		{http.MethodGet, "/tenant/leads/some-id"},
		{http.MethodPost, "/tenant/leads/some-id/notes"},
	}
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(r.method, r.path, nil)
			req.AddCookie(owaspCookie(token))
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

func TestOWASP_A01_TenantIsolation_LeadNotVisible(t *testing.T) {
	env := setupOwaspEnv()

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-a", "funnel-1", "col-1", "contact-1", "conv-1")
	_ = env.leadRepo.Create(context.Background(), lead)

	tokenB := env.tenantToken(t, "tenant-b")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenant/leads/"+lead.ID, nil)
	req.AddCookie(owaspCookie(tokenB))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOWASP_A01_TenantIsolation_CreateNoteDenied(t *testing.T) {
	env := setupOwaspEnv()

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-a", "funnel-1", "col-1", "contact-1", "conv-1")
	_ = env.leadRepo.Create(context.Background(), lead)

	tokenB := env.tenantToken(t, "tenant-b")
	body := strings.NewReader("content=test+note")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tenant/leads/"+lead.ID+"/notes", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(owaspCookie(tokenB))
	env.router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}
```

- [ ] **Step 2: Run OWASP tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/funnel/interfaces/http/ -run TestOWASP -v`
Expected: PASS (4 tests)

- [ ] **Step 3: Commit**

```bash
git add internal/funnel/interfaces/http/owasp_test.go
git commit -m "test(F07): add OWASP tests for lead detail and notes endpoints"
```
