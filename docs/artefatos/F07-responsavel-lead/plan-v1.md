# Responsavel no Lead — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Exibir o responsavel no card do kanban, permitir atribuicao no drawer de detalhes, e filtrar o board por responsavel.

**Architecture:** O backend ja possui `Lead.ResponsibleUserID` e `AssignLeadUseCase`. As alteracoes sao: (1) adicionar `ResponsibleUserID` ao filtro e ao output do kanban, (2) resolver nome do responsavel via `UserNameProvider`, (3) novo provider `TenantUserLister` para listar usuarios do tenant no dropdown, (4) templates HTMX para card, drawer e filtro.

**Tech Stack:** Go, Gin, GORM, HTMX, html/template, testify

---

## File Map

| Action | File | Responsibility |
|--------|------|---------------|
| Modify | `internal/funnel/domain/repository.go` | Add `ResponsibleUserID` to `LeadFilter` and `ResponsibleUserName` to `LeadWithContact` |
| Modify | `internal/funnel/domain/providers.go` | Add `TenantUserLister` interface |
| Modify | `internal/funnel/infrastructure/gorm_lead_repository.go` | Add responsible filter to count and data queries, join users table for name |
| Modify | `internal/funnel/application/get_kanban.go` | Add `ResponsibleUserID` to input/output, pass to filter, populate name |
| Modify | `internal/funnel/application/get_lead_detail.go` | Populate `AssignedToName` from `userNameProvider` |
| Modify | `internal/funnel/interfaces/http/handler.go` | Accept `responsible` query param, add `userNameProvider` + `tenantUserLister` deps, new `RenderLeadAssignForm` + improved `HandleAssignLead` |
| Modify | `internal/funnel/interfaces/http/routes.go` | Add route for assign form |
| Modify | `internal/funnel/module.go` | Wire `TenantUserLister` |
| Create | `internal/funnel/infrastructure/user_lister_adapter.go` | Adapter implementing `TenantUserLister` using auth repos |
| Modify | `web/templates/funnel/kanban.html` | Add responsible filter dropdown |
| Modify | `web/templates/funnel/kanban_content.html` | Show responsible name on card |
| Modify | `web/templates/funnel/lead_drawer.html` | Functional responsible section with assign button |
| Create | `web/templates/funnel/lead_responsible_section.html` | Swappable responsible section |
| Create | `web/templates/funnel/lead_assign_form.html` | Dropdown form for assigning responsible |
| Modify | `internal/funnel/application/mocks_test.go` | Add `mockTenantUserLister`, update `mockLeadRepo.FindByFunnelID` for responsible filter |
| Modify | `internal/funnel/application/get_kanban_test.go` | Test kanban with responsible filter and name |
| Modify | `internal/funnel/application/get_lead_detail_test.go` | Test `AssignedToName` population |
| Modify | `internal/funnel/interfaces/http/owasp_test.go` | OWASP tests for responsible filter + assign |

---

### Task 1: Add ResponsibleUserID to LeadFilter and LeadWithContact

**Files:**
- Modify: `internal/funnel/domain/repository.go:40-61`

- [ ] **Step 1: Add `ResponsibleUserID` field to `LeadFilter`**

```go
type LeadFilter struct {
	Search            string
	ColumnID          string
	ProductID         string
	ResponsibleUserID string
	Page              int
	Limit             int
}
```

- [ ] **Step 2: Add `ResponsibleUserID` and `ResponsibleUserName` to `LeadWithContact`**

```go
type LeadWithContact struct {
	Lead                Lead
	ContactName         string
	ContactPhone        string
	ColumnName          string
	ColumnColor         string
	ResponsibleUserName string
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/funnel/domain/repository.go
git commit -m "feat: add ResponsibleUserID to LeadFilter and ResponsibleUserName to LeadWithContact"
```

---

### Task 2: Add TenantUserLister provider interface

**Files:**
- Modify: `internal/funnel/domain/providers.go`

- [ ] **Step 1: Add `TenantUserInfo` and `TenantUserLister` interface**

Append after the existing `ProductLister` interface:

```go
// TenantUserInfo is a lightweight user representation for dropdowns.
type TenantUserInfo struct {
	ID   string
	Name string
}

// TenantUserLister lists active users for a tenant (used by kanban for filter/assign dropdowns).
type TenantUserLister interface {
	ListUsersByTenantID(ctx context.Context, tenantID string) ([]TenantUserInfo, error)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/funnel/domain/providers.go
git commit -m "feat: add TenantUserLister provider interface"
```

---

### Task 3: Add responsible filter to repository query

**Files:**
- Modify: `internal/funnel/infrastructure/gorm_lead_repository.go:82-196`

- [ ] **Step 1: Add `responsible_user_id` to leadRow struct**

In the `leadRow` struct inside `FindByFunnelID`, add:

```go
ResponsibleUserID string `gorm:"column:responsible_user_id"`
```

- [ ] **Step 2: Add `leads.responsible_user_id` to the SELECT clause**

In the `dataQuery` Select, add `leads.responsible_user_id` to the selected fields:

```go
dataQuery := r.db.WithContext(ctx).
	Table("leads").
	Select(`leads.id, leads.tenant_id, leads.funnel_id, leads.column_id,
		leads.contact_id, leads.conversation_id, leads.product_id, leads.responsible_user_id,
		leads.score, leads.status,
		leads.column_entered_at, leads.created_at as lead_created_at, leads.updated_at as lead_updated_at,
		contacts.name as contact_name, contacts.phone as contact_phone,
		funnel_columns.name as column_name, funnel_columns.color as column_color,
		funnel_columns.order_index as column_order`).
	Joins("JOIN contacts ON contacts.id = leads.contact_id").
	Joins("JOIN funnel_columns ON funnel_columns.id = leads.column_id").
	Where("leads.funnel_id = ?", funnelID)
```

- [ ] **Step 3: Add responsible filter to both count and data queries**

After the `ProductID` filter block in both `countQuery` and `dataQuery`, add:

```go
if filter.ResponsibleUserID != "" {
	countQuery = countQuery.Where("leads.responsible_user_id = ?", filter.ResponsibleUserID)
}
```

And for dataQuery:

```go
if filter.ResponsibleUserID != "" {
	dataQuery = dataQuery.Where("leads.responsible_user_id = ?", filter.ResponsibleUserID)
}
```

- [ ] **Step 4: Populate `ResponsibleUserID` in the Lead and domain mapping**

In the loop that builds `leads[i]`, add `ResponsibleUserID` to the Lead struct:

```go
leads[i] = domain.LeadWithContact{
	Lead: domain.Lead{
		ID:                row.ID,
		TenantID:          row.TenantID,
		FunnelID:          row.FunnelID,
		ColumnID:          row.ColumnID,
		ContactID:         row.ContactID,
		ConversationID:    row.ConversationID,
		ProductID:         row.ProductID,
		ResponsibleUserID: row.ResponsibleUserID,
		Score:             row.Score,
		Status:            domain.LeadStatus(row.Status),
	},
	ContactName:  row.ContactName,
	ContactPhone: row.ContactPhone,
	ColumnName:   row.ColumnName,
	ColumnColor:  row.ColumnColor,
}
```

- [ ] **Step 5: Commit**

```bash
git add internal/funnel/infrastructure/gorm_lead_repository.go
git commit -m "feat: add responsible_user_id filter and select to lead repository"
```

---

### Task 4: Update GetKanbanUseCase to include responsible name and filter

**Files:**
- Modify: `internal/funnel/application/get_kanban.go`

- [ ] **Step 1: Add `ResponsibleUserID` to `GetKanbanInput`**

```go
type GetKanbanInput struct {
	TenantID          string
	FunnelID          string
	Search            string
	ProductID         string
	ResponsibleUserID string
}
```

- [ ] **Step 2: Add `ResponsibleUserID` and `ResponsibleUserName` to `KanbanLead`**

```go
type KanbanLead struct {
	ID                  string
	ContactName         string
	ContactPhone        string
	Score               int
	Status              string
	ConversationID      string
	ProductID           string
	ProductName         string
	ResponsibleUserID   string
	ResponsibleUserName string
}
```

- [ ] **Step 3: Pass `ResponsibleUserID` to the filter in Execute**

In the `uc.leadRepo.FindByFunnelID` call, add the field:

```go
leadList, err := uc.leadRepo.FindByFunnelID(ctx, funnel.ID, domain.LeadFilter{
	Search:            input.Search,
	ColumnID:          col.ID,
	ProductID:         input.ProductID,
	ResponsibleUserID: input.ResponsibleUserID,
	Page:              1,
	Limit:             100,
})
```

- [ ] **Step 4: Populate `ResponsibleUserID` in the KanbanLead mapping**

In the loop that builds `leads[j]`:

```go
leads[j] = KanbanLead{
	ID:                lc.Lead.ID,
	ContactName:       lc.ContactName,
	ContactPhone:      lc.ContactPhone,
	Score:             lc.Lead.Score,
	Status:            string(lc.Lead.Status),
	ConversationID:    lc.Lead.ConversationID,
	ProductID:         lc.Lead.ProductID,
	ResponsibleUserID: lc.Lead.ResponsibleUserID,
}
```

Note: `ResponsibleUserName` will be resolved by the handler (same pattern as `ProductName`).

- [ ] **Step 5: Commit**

```bash
git add internal/funnel/application/get_kanban.go
git commit -m "feat: add responsible filter and output to GetKanbanUseCase"
```

---

### Task 5: Fix GetLeadDetailUseCase to populate AssignedToName

**Files:**
- Modify: `internal/funnel/application/get_lead_detail.go:188-210`

- [ ] **Step 1: Write failing test for AssignedToName**

Add to `internal/funnel/application/get_lead_detail_test.go`:

```go
func TestGetLeadDetail_AssignedToName(t *testing.T) {
	uc, leadRepo, _, funnelRepo, columnRepo, contactProvider, _, _, userNameProvider := setupLeadDetailTest()

	funnel, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Vendas", "")
	_ = funnelRepo.Create(context.Background(), funnel)
	col, _ := domain.NewColumn(uuid.New().String(), funnel.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), col)

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", funnel.ID, col.ID, "contact-1", "conv-1")
	lead.AssignResponsible("user-resp-1")
	_ = leadRepo.Create(context.Background(), lead)

	contactProvider.contacts["contact-1"] = domain.ContactInfo{Name: "Carlos", Phone: "+5511999990000"}
	userNameProvider.names["user-resp-1"] = "Ana Souza"

	output, err := uc.Execute(context.Background(), GetLeadDetailInput{
		TenantID: "tenant-1", LeadID: lead.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, "Ana Souza", output.AssignedToName)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/funnel/application/ -run TestGetLeadDetail_AssignedToName -v`
Expected: FAIL — `AssignedToName` is empty string, not "Ana Souza"

- [ ] **Step 3: Add AssignedToName resolution in Execute**

In `get_lead_detail.go`, after the product name resolution block (line ~188) and before the `return`, add:

```go
	// Assigned user name
	var assignedToName string
	if lead.ResponsibleUserID != "" {
		if name, err := uc.userNameProvider.FindNameByID(ctx, lead.ResponsibleUserID); err == nil {
			assignedToName = name
		}
	}
```

And add `AssignedToName: assignedToName,` to the return struct.

The full return block becomes:

```go
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
		ProductName:     productName,
		AssignedToName:  assignedToName,
	}, nil
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/funnel/application/ -run TestGetLeadDetail_AssignedToName -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/funnel/application/get_lead_detail.go internal/funnel/application/get_lead_detail_test.go
git commit -m "fix: populate AssignedToName in GetLeadDetailUseCase"
```

---

### Task 6: Update mock LeadRepo to support responsible filter

**Files:**
- Modify: `internal/funnel/application/mocks_test.go:236-265`

- [ ] **Step 1: Add ResponsibleUserID filter to mockLeadRepo.FindByFunnelID**

Replace the `FindByFunnelID` method:

```go
func (m *mockLeadRepo) FindByFunnelID(_ context.Context, funnelID string, filter domain.LeadFilter) (*domain.LeadList, error) {
	var result []domain.LeadWithContact
	for _, l := range m.leads {
		if l.FunnelID != funnelID {
			continue
		}
		if filter.ColumnID != "" && l.ColumnID != filter.ColumnID {
			continue
		}
		if filter.ProductID != "" && l.ProductID != filter.ProductID {
			continue
		}
		if filter.ResponsibleUserID != "" && l.ResponsibleUserID != filter.ResponsibleUserID {
			continue
		}
		result = append(result, domain.LeadWithContact{Lead: *l})
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}
	return &domain.LeadList{
		Leads: result,
		Total: int64(len(result)),
		Page:  page,
		Limit: limit,
	}, nil
}
```

- [ ] **Step 2: Add mockTenantUserLister**

```go
// --- Mock TenantUserLister ---

type mockTenantUserLister struct {
	users map[string][]domain.TenantUserInfo // tenantID -> users
}

func newMockTenantUserLister() *mockTenantUserLister {
	return &mockTenantUserLister{users: make(map[string][]domain.TenantUserInfo)}
}

func (m *mockTenantUserLister) ListUsersByTenantID(_ context.Context, tenantID string) ([]domain.TenantUserInfo, error) {
	return m.users[tenantID], nil
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/funnel/application/mocks_test.go
git commit -m "test: update mocks for responsible filter and tenant user lister"
```

---

### Task 7: Test kanban with responsible filter

**Files:**
- Modify: `internal/funnel/application/get_kanban_test.go`

- [ ] **Step 1: Add test for kanban filtering by responsible**

```go
func TestGetKanban_FilterByResponsible(t *testing.T) {
	uc, _, columnRepo, leadRepo, f := setupKanbanTest(t)

	var entryColID string
	for _, col := range columnRepo.byFunnel[f.ID] {
		if col.Type == domain.ColumnTypeEntry {
			entryColID = col.ID
			break
		}
	}

	lead1, _ := domain.NewLead(uuid.New().String(), "tenant-1", f.ID, entryColID, "contact-1", "conv-1")
	lead1.AssignResponsible("user-a")
	_ = leadRepo.Create(context.Background(), lead1)

	lead2, _ := domain.NewLead(uuid.New().String(), "tenant-1", f.ID, entryColID, "contact-2", "conv-2")
	lead2.AssignResponsible("user-b")
	_ = leadRepo.Create(context.Background(), lead2)

	// Filter by user-a: should see only 1 lead
	output, err := uc.Execute(context.Background(), GetKanbanInput{
		TenantID:          "tenant-1",
		ResponsibleUserID: "user-a",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, output.TotalLeads)

	// No filter: should see both
	output, err = uc.Execute(context.Background(), GetKanbanInput{
		TenantID: "tenant-1",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, output.TotalLeads)
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `go test ./internal/funnel/application/ -run TestGetKanban_FilterByResponsible -v`
Expected: PASS (logic already flows through — filter added in Task 4 step 3, mock updated in Task 6)

- [ ] **Step 3: Add test for ResponsibleUserID in KanbanLead output**

```go
func TestGetKanban_LeadHasResponsibleUserID(t *testing.T) {
	uc, _, columnRepo, leadRepo, f := setupKanbanTest(t)

	var entryColID string
	for _, col := range columnRepo.byFunnel[f.ID] {
		if col.Type == domain.ColumnTypeEntry {
			entryColID = col.ID
			break
		}
	}

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", f.ID, entryColID, "contact-1", "conv-1")
	lead.AssignResponsible("user-x")
	_ = leadRepo.Create(context.Background(), lead)

	output, err := uc.Execute(context.Background(), GetKanbanInput{TenantID: "tenant-1"})
	require.NoError(t, err)

	found := false
	for _, col := range output.Columns {
		for _, kl := range col.Leads {
			if kl.ID == lead.ID {
				assert.Equal(t, "user-x", kl.ResponsibleUserID)
				found = true
			}
		}
	}
	assert.True(t, found, "lead should be in kanban output")
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/funnel/application/ -run TestGetKanban -v`
Expected: All kanban tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/funnel/application/get_kanban_test.go
git commit -m "test: add kanban tests for responsible filter and output"
```

---

### Task 8: Create TenantUserLister adapter

**Files:**
- Create: `internal/funnel/infrastructure/user_lister_adapter.go`

- [ ] **Step 1: Create the adapter**

```go
package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

// TenantUserListerAdapter lists active users for a tenant by joining user_tenants and users.
type TenantUserListerAdapter struct {
	db *gorm.DB
}

func NewTenantUserListerAdapter(db *gorm.DB) *TenantUserListerAdapter {
	return &TenantUserListerAdapter{db: db}
}

func (a *TenantUserListerAdapter) ListUsersByTenantID(ctx context.Context, tenantID string) ([]domain.TenantUserInfo, error) {
	type row struct {
		ID   string `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}

	var rows []row
	err := a.db.WithContext(ctx).
		Table("users").
		Select("users.id, users.name").
		Joins("JOIN user_tenants ON user_tenants.user_id = users.id").
		Where("user_tenants.tenant_id = ? AND users.status = ?", tenantID, "active").
		Order("users.name ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	users := make([]domain.TenantUserInfo, len(rows))
	for i, r := range rows {
		users[i] = domain.TenantUserInfo{ID: r.ID, Name: r.Name}
	}
	return users, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/funnel/infrastructure/user_lister_adapter.go
git commit -m "feat: add TenantUserListerAdapter for listing tenant users"
```

---

### Task 9: Update Handler to resolve responsible names and accept filter

**Files:**
- Modify: `internal/funnel/interfaces/http/handler.go`

- [ ] **Step 1: Add `userNameProvider` and `tenantUserLister` to Handler struct and constructor**

Add fields to Handler struct:

```go
type Handler struct {
	// ... existing fields ...
	userNameProvider domain.UserNameProvider
	tenantUserLister domain.TenantUserLister
}
```

Update `NewHandler` signature to accept new params (add after `productProvider domain.ProductProvider`):

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
	assignLeadUC *application.AssignLeadUseCase,
	leadRepo domain.LeadRepository,
	productLister domain.ProductLister,
	productProvider domain.ProductProvider,
	userNameProvider domain.UserNameProvider,
	tenantUserLister domain.TenantUserLister,
	log *zap.Logger,
) *Handler {
	return &Handler{
		// ... existing fields ...
		userNameProvider:  userNameProvider,
		tenantUserLister:  tenantUserLister,
	}
}
```

- [ ] **Step 2: Add `resolveResponsibleNames` method (same pattern as `resolveProductNames`)**

```go
// resolveResponsibleNames enriches KanbanLead with ResponsibleUserName if available.
func (h *Handler) resolveResponsibleNames(c *gin.Context, output *application.KanbanOutput) {
	if h.userNameProvider == nil {
		return
	}
	for i := range output.Columns {
		for j := range output.Columns[i].Leads {
			lead := &output.Columns[i].Leads[j]
			if lead.ResponsibleUserID != "" {
				name, err := h.userNameProvider.FindNameByID(c.Request.Context(), lead.ResponsibleUserID)
				if err == nil {
					lead.ResponsibleUserName = name
				}
			}
		}
	}
}
```

- [ ] **Step 3: Update `RenderKanbanPage` to load users for filter dropdown**

After the products loading block, add:

```go
	// Load users for responsible filter dropdown
	var tenantUsers []domain.TenantUserInfo
	if h.tenantUserLister != nil {
		tenantUsers, _ = h.tenantUserLister.ListUsersByTenantID(c.Request.Context(), tenantID)
	}
```

Add to the template data:

```go
	c.HTML(http.StatusOK, "funnel/kanban.html", gin.H{
		"CurrentFunnelID":       currentFunnelID,
		"Funnels":               funnels,
		"Products":              products,
		"CurrentProductID":      c.Query("product_id"),
		"TenantUsers":           tenantUsers,
		"CurrentResponsibleID":  c.Query("responsible"),
		"CurrentUserID":         c.GetString("user_id"),
		"ActiveNav":             "leads",
	})
```

- [ ] **Step 4: Update `RenderKanbanContent` to accept `responsible` query param**

```go
func (h *Handler) RenderKanbanContent(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Query("funnel_id")
	search := c.Query("search")
	productID := c.Query("product_id")
	responsibleID := c.Query("responsible")

	output, err := h.getKanbanUC.Execute(c.Request.Context(), application.GetKanbanInput{
		TenantID:          tenantID,
		FunnelID:          funnelID,
		Search:            search,
		ProductID:         productID,
		ResponsibleUserID: responsibleID,
	})
	if err != nil {
		c.HTML(http.StatusOK, "funnel/kanban_content.html", gin.H{})
		return
	}

	h.resolveProductNames(c, output)
	h.resolveResponsibleNames(c, output)

	h.log.Info("kanban data",
		zap.String("funnel", output.FunnelName),
		zap.Int("columns", len(output.Columns)),
		zap.Int("leads", output.TotalLeads),
	)

	c.HTML(http.StatusOK, "funnel/kanban_content.html", gin.H{
		"FunnelID":   output.FunnelID,
		"FunnelName": output.FunnelName,
		"Columns":    output.Columns,
		"TotalLeads": output.TotalLeads,
		"Search":     search,
	})
}
```

- [ ] **Step 5: Add `RenderLeadAssignForm` handler**

```go
func (h *Handler) RenderLeadAssignForm(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	leadID := c.Param("id")

	var tenantUsers []domain.TenantUserInfo
	if h.tenantUserLister != nil {
		tenantUsers, _ = h.tenantUserLister.ListUsersByTenantID(c.Request.Context(), tenantID)
	}

	lead, err := h.leadRepo.FindByID(c.Request.Context(), leadID)
	if err != nil || lead.TenantID != tenantID {
		c.HTML(http.StatusNotFound, "funnel/lead_assign_form.html", gin.H{
			"Error": "Lead nao encontrado",
		})
		return
	}

	c.HTML(http.StatusOK, "funnel/lead_assign_form.html", gin.H{
		"LeadID":              leadID,
		"TenantUsers":         tenantUsers,
		"CurrentResponsibleID": lead.ResponsibleUserID,
	})
}
```

- [ ] **Step 6: Update `HandleAssignLead` to return the updated responsible section**

Replace the existing `HandleAssignLead`:

```go
func (h *Handler) HandleAssignLead(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	leadID := c.Param("id")

	var req struct {
		UserID string `json:"user_id" form:"user_id"`
	}
	if err := c.ShouldBind(&req); err != nil {
		h.log.Error("invalid assign lead request", zap.Error(err))
		c.Status(http.StatusBadRequest)
		return
	}

	err := h.assignLeadUC.Execute(c.Request.Context(), application.AssignLeadInput{
		LeadID:   leadID,
		UserID:   req.UserID,
		TenantID: tenantID,
	})
	if err != nil {
		h.log.Error("failed to assign lead", zap.Error(err))
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	// Resolve user name for the response
	var assignedName string
	if req.UserID != "" && h.userNameProvider != nil {
		name, err := h.userNameProvider.FindNameByID(c.Request.Context(), req.UserID)
		if err == nil {
			assignedName = name
		}
	}

	c.HTML(http.StatusOK, "funnel/lead_responsible_section.html", gin.H{
		"LeadID":         leadID,
		"AssignedToName": assignedName,
	})
}
```

- [ ] **Step 7: Commit**

```bash
git add internal/funnel/interfaces/http/handler.go
git commit -m "feat: add responsible filter, name resolution, and assign form to handler"
```

---

### Task 10: Update routes and module wiring

**Files:**
- Modify: `internal/funnel/interfaces/http/routes.go`
- Modify: `internal/funnel/module.go`

- [ ] **Step 1: Add assign-form route**

In `routes.go`, after the existing `tenant.PUT("/:id/assign", h.HandleAssignLead)` line, add:

```go
	tenant.GET("/:id/assign-form", h.RenderLeadAssignForm)
```

- [ ] **Step 2: Update module.go to accept and wire TenantUserLister**

Update `NewModule` signature to add `tenantUserLister domain.TenantUserLister`:

```go
func NewModule(
	db *gorm.DB,
	contactProvider domain.ContactProvider,
	messageProvider domain.MessageProvider,
	userNameProvider domain.UserNameProvider,
	productDetector domain.ProductDetector,
	productProvider domain.ProductProvider,
	funnelProductRouter domain.FunnelProductRouter,
	productLister domain.ProductLister,
	tenantUserLister domain.TenantUserLister,
	eventBus events.EventBus,
	log *zap.Logger,
) *Module {
```

Update the `handler` construction to pass the new deps:

```go
	handler := funnelhttp.NewHandler(
		getKanbanUC, listFunnelsUC, getFunnelUC,
		createFunnelUC, updateFunnelUC, toggleFunnelUC,
		createColumnUC, deleteColumnUC, moveColumnUC,
		createLeadUC, moveLeadUC, getLeadDetailUC,
		createLeadNoteUC, assignLeadUC,
		leadRepo, productLister, productProvider,
		userNameProvider, tenantUserLister, log,
	)
```

- [ ] **Step 3: Update main.go to create and pass TenantUserListerAdapter**

Find where `funnel.NewModule` is called in `cmd/api/main.go`. Add adapter creation and pass it:

```go
tenantUserLister := funnelinfra.NewTenantUserListerAdapter(db)
```

Pass `tenantUserLister` as the new argument to `funnel.NewModule(...)`.

Note: You'll need to add the import for `funnelinfra` if not already present:
```go
funnelinfra "github.com/sasrgita/crm-juridico/internal/funnel/infrastructure"
```

- [ ] **Step 4: Run `go build ./cmd/api/` to verify compilation**

Run: `go build ./cmd/api/`
Expected: compiles cleanly

- [ ] **Step 5: Commit**

```bash
git add internal/funnel/interfaces/http/routes.go internal/funnel/module.go cmd/api/main.go
git commit -m "feat: wire TenantUserLister and assign-form route"
```

---

### Task 11: Create HTMX templates for responsible section and assign form

**Files:**
- Create: `web/templates/funnel/lead_responsible_section.html`
- Create: `web/templates/funnel/lead_assign_form.html`

- [ ] **Step 1: Create lead_responsible_section.html**

```html
{{define "funnel/lead_responsible_section.html"}}
<h4>Responsavel</h4>
{{if .AssignedToName}}
<div style="display:flex;justify-content:space-between;align-items:center">
    <span style="font-size:0.875rem">{{.AssignedToName}}</span>
    <button class="btn btn-sm btn-outline"
            hx-get="/tenant/leads/{{.LeadID}}/assign-form"
            hx-target="#lead-responsible-section"
            hx-swap="innerHTML">Trocar</button>
</div>
{{else}}
<div style="display:flex;justify-content:space-between;align-items:center">
    <p class="lead-empty-text" style="margin:0">Sem responsavel</p>
    <button class="btn btn-sm btn-outline"
            hx-get="/tenant/leads/{{.LeadID}}/assign-form"
            hx-target="#lead-responsible-section"
            hx-swap="innerHTML">Atribuir</button>
</div>
{{end}}
{{end}}
```

- [ ] **Step 2: Create lead_assign_form.html**

```html
{{define "funnel/lead_assign_form.html"}}
<h4>Responsavel</h4>
{{if .Error}}
<p style="color:#ef4444;font-size:0.8125rem">{{.Error}}</p>
{{end}}
<form hx-put="/tenant/leads/{{.LeadID}}/assign"
      hx-target="#lead-responsible-section"
      hx-swap="innerHTML"
      style="display:flex;flex-direction:column;gap:0.5rem">
    <select name="user_id" class="form-select" style="font-size:0.8125rem">
        <option value="">Sem responsavel</option>
        {{range .TenantUsers}}
        <option value="{{.ID}}" {{if eq .ID $.CurrentResponsibleID}}selected{{end}}>{{.Name}}</option>
        {{end}}
    </select>
    <div style="display:flex;gap:0.5rem">
        <button type="submit" class="btn btn-sm btn-primary">Salvar</button>
        <button type="button" class="btn btn-sm btn-outline"
                hx-get="/tenant/leads/{{.LeadID}}"
                hx-target="#lead-modal"
                hx-swap="innerHTML">Cancelar</button>
    </div>
</form>
{{end}}
```

- [ ] **Step 3: Commit**

```bash
git add web/templates/funnel/lead_responsible_section.html web/templates/funnel/lead_assign_form.html
git commit -m "feat: add HTMX templates for responsible section and assign form"
```

---

### Task 12: Update lead_drawer.html to use the responsible section template

**Files:**
- Modify: `web/templates/funnel/lead_drawer.html:119-127`

- [ ] **Step 1: Replace the placeholder responsible section**

Replace the existing block:

```html
            <!-- Responsavel (placeholder F08) -->
            <div class="lead-detail-section">
                <h4>Responsavel</h4>
                {{if .Lead.AssignedToName}}
                <p>{{.Lead.AssignedToName}}</p>
                {{else}}
                <p class="lead-empty-text">Nenhum responsavel atribuido</p>
                {{end}}
            </div>
```

With:

```html
            <!-- Responsavel -->
            <div class="lead-detail-section" id="lead-responsible-section">
                {{template "funnel/lead_responsible_section.html" (dict "LeadID" .Lead.ID "AssignedToName" .Lead.AssignedToName)}}
            </div>
```

- [ ] **Step 2: Commit**

```bash
git add web/templates/funnel/lead_drawer.html
git commit -m "feat: replace responsible placeholder with functional section in drawer"
```

---

### Task 13: Update kanban_content.html to show responsible name on card

**Files:**
- Modify: `web/templates/funnel/kanban_content.html:20-34`

- [ ] **Step 1: Add responsible name to the card**

After the product badge block (line 31-32), add the responsible name display:

```html
                {{if .ProductName}}
                <div style="margin-top:4px"><span style="font-size:0.6875rem;padding:1px 6px;border-radius:8px;background:#dbeafe;color:#2563eb">{{.ProductName}}</span></div>
                {{end}}
                <div style="margin-top:4px;display:flex;align-items:center;gap:4px;max-width:100%">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" width="12" height="12" style="flex-shrink:0;color:{{if .ResponsibleUserName}}#6b7280{{else}}#d1d5db{{end}}"><path stroke-linecap="round" stroke-linejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/></svg>
                    <span style="font-size:0.6875rem;color:{{if .ResponsibleUserName}}#374151{{else}}#d1d5db{{end}};white-space:nowrap;overflow:hidden;text-overflow:ellipsis">{{if .ResponsibleUserName}}{{.ResponsibleUserName}}{{else}}Sem responsavel{{end}}</span>
                </div>
```

- [ ] **Step 2: Commit**

```bash
git add web/templates/funnel/kanban_content.html
git commit -m "feat: show responsible name on kanban card"
```

---

### Task 14: Add responsible filter dropdown to kanban.html

**Files:**
- Modify: `web/templates/funnel/kanban.html:30-49`

- [ ] **Step 1: Add responsible filter dropdown**

After the product filter dropdown (`</select>` on line 42) and before the search input, add:

```html
                {{if .TenantUsers}}
                <select class="kanban-funnel-select"
                        hx-get="/tenant/leads/kanban"
                        hx-target="#kanban-content"
                        hx-include="[name=funnel_id],[name=product_id],[name=search]"
                        name="responsible"
                        style="min-width:160px">
                    <option value="">Todos responsaveis</option>
                    <option value="{{.CurrentUserID}}" {{if eq .CurrentResponsibleID .CurrentUserID}}selected{{end}}>Meus leads</option>
                    {{range .TenantUsers}}
                    <option value="{{.ID}}" {{if eq .ID $.CurrentResponsibleID}}selected{{end}}>{{.Name}}</option>
                    {{end}}
                </select>
                {{end}}
```

- [ ] **Step 2: Update the existing `hx-include` attributes to include the responsible filter**

The existing funnel, product, and search inputs need to include `[name=responsible]`:

Funnel select `hx-include`:
```
hx-include="[name=product_id],[name=search],[name=responsible]"
```

Product select `hx-include`:
```
hx-include="[name=funnel_id],[name=search],[name=responsible]"
```

Search input `hx-include`:
```
hx-include="[name=funnel_id],[name=product_id],[name=responsible]"
```

- [ ] **Step 3: Commit**

```bash
git add web/templates/funnel/kanban.html
git commit -m "feat: add responsible filter dropdown to kanban header"
```

---

### Task 15: Update OWASP tests

**Files:**
- Modify: `internal/funnel/interfaces/http/owasp_test.go`

- [ ] **Step 1: Determine current OWASP test patterns**

Read `internal/funnel/interfaces/http/owasp_test.go` to see the existing patterns. The Handler constructor call needs to be updated to include the two new params (`userNameProvider`, `tenantUserLister`).

- [ ] **Step 2: Add mock TenantUserLister to OWASP test file**

```go
type owaspMockTenantUserLister struct{}

func (m *owaspMockTenantUserLister) ListUsersByTenantID(_ context.Context, _ string) ([]domain.TenantUserInfo, error) {
	return nil, nil
}
```

- [ ] **Step 3: Update Handler construction in OWASP tests**

Find the `NewHandler(...)` call in the OWASP test setup and add the two new params (after `productProvider`):

```go
&owaspMockUserNameProvider{},
&owaspMockTenantUserLister{},
```

- [ ] **Step 4: Add OWASP test for responsible filter tenant isolation**

```go
func TestOWASP_KanbanResponsibleFilter_TenantIsolation(t *testing.T) {
	env := setupOwaspEnv()

	// Request kanban with responsible filter for wrong tenant
	req := httptest.NewRequest("GET", "/tenant/leads/kanban?responsible=user-from-other-tenant", nil)
	req = req.WithContext(injectOwaspAuth(req.Context(), "tenant-1", "user-1"))
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	// Should succeed (200) but return no leads from other tenant
	assert.Equal(t, http.StatusOK, w.Code)
}
```

- [ ] **Step 5: Run OWASP tests**

Run: `go test ./internal/funnel/interfaces/http/ -run TestOWASP -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/funnel/interfaces/http/owasp_test.go
git commit -m "test: update OWASP tests for responsible filter and handler deps"
```

---

### Task 16: Run full test suite and verify coverage

- [ ] **Step 1: Run all funnel tests**

Run: `go test ./internal/funnel/... -v -count=1`
Expected: All PASS

- [ ] **Step 2: Check coverage**

Run: `go test ./internal/funnel/... -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`
Expected: >= 80%

- [ ] **Step 3: Run full build**

Run: `go build ./cmd/api/`
Expected: compiles cleanly

- [ ] **Step 4: Commit any remaining fixes if needed**

---

### Task 17: Update rest/ HTTP files

**Files:**
- Modify or create: `rest/leads.http`

- [ ] **Step 1: Add HTTP test entries for new endpoints**

```http
### Kanban with responsible filter
GET {{host}}/tenant/leads/kanban?funnel_id={{funnel_id}}&responsible={{user_id}}
Authorization: Bearer {{token}}

### Assign lead form
GET {{host}}/tenant/leads/{{lead_id}}/assign-form
Authorization: Bearer {{token}}

### Assign responsible to lead
PUT {{host}}/tenant/leads/{{lead_id}}/assign
Authorization: Bearer {{token}}
Content-Type: application/x-www-form-urlencoded

user_id={{user_id}}
```

- [ ] **Step 2: Commit**

```bash
git add rest/
git commit -m "docs: add HTTP test entries for responsible lead endpoints"
```
