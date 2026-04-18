# F08 Step 6 — Telas de Equipe (HTMX) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the 4 remaining HTMX screens of F08 (users, groups with detail sections for members, permissions, funnels, view profiles, and load balance) under a single "Equipe" sidebar item, and complete the load-balance backend (use case + endpoints + `active` flag) that the UI needs.

**Architecture:** Adds PageHandler HTML renderers in the `auth` module (Users tab) and `permission` module (Groups tab + group detail). Each tab and each detail section is served by a dedicated Gin handler that returns HTML fragments for HTMX swaps. Load balance gets a new use case (`ManageLoadBalanceUseCase`) in `auth`, new endpoints `/tenant/groups/:id/load-balance` on the `permission` module (cross-module import of the use case), an `Active` column on `load_balance_configs`, and a small migration.

**Tech Stack:** Go, Gin, Gorm, HTMX 2.0.4, Go html/template, zap, existing CSS from `main.css`.

**Spec:** `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/design-step6-team-screens.md`

---

## File Structure

```
# Backend — Load Balance (Phase 1)
Modify: internal/auth/domain/load_balance.go                                 — add Active bool
Modify: internal/auth/infrastructure/gorm_load_balance_repo.go               — persist Active
Create: internal/auth/application/manage_load_balance.go                     — ManageLoadBalanceUseCase
Create: internal/auth/application/manage_load_balance_test.go
Create: migrations/000051_add_active_to_load_balance_configs.up.sql
Create: migrations/000051_add_active_to_load_balance_configs.down.sql
Modify: internal/auth/module.go                                              — wire LB repo + UC, expose accessor
Modify: internal/permission/interfaces/http/handler.go                       — load-balance handlers
Modify: internal/permission/interfaces/http/routes.go                        — GET/PUT /tenant/groups/:id/load-balance
Modify: internal/permission/module.go                                        — accept LB use case via constructor
Modify: cmd/api/main.go                                                      — pass LB use case into permission module

# Backend — Page Handlers (Phase 2+)
Create: internal/auth/interfaces/http/page_handler.go                        — Users tab HTML handler
Create: internal/auth/interfaces/http/page_handler_test.go
Modify: internal/auth/module.go                                              — wire PageHandler + HTML routes
Create: internal/permission/interfaces/http/page_handler.go                  — Groups tab + detail HTML handler
Create: internal/permission/interfaces/http/page_handler_test.go
Modify: internal/permission/interfaces/http/routes.go                        — register HTML routes
Modify: internal/permission/module.go                                        — accept cross-module deps (funnels, users)
Modify: cmd/api/main.go                                                      — pass cross-module deps

# Frontend — Templates
Modify: web/templates/partials/tenant_sidebar.html                           — add "Equipe" item
Create: web/templates/team/shell.html                                        — shared shell (sidebar + tabs)
Create: web/templates/team/users_page.html                                   — Users tab page
Create: web/templates/team/users_table.html                                  — Users + invites fragment
Create: web/templates/team/user_permissions_modal.html                       — modal: individual perms
Create: web/templates/team/user_whatsapp_modal.html                          — modal: WhatsApp ID
Create: web/templates/team/invite_new_modal.html                             — modal: new invite
Create: web/templates/team/invite_success.html                               — fragment after generating invite
Create: web/templates/team/groups_page.html                                  — Groups tab page
Create: web/templates/team/groups_table.html                                 — groups fragment
Create: web/templates/team/group_new_modal.html                              — modal: new group
Create: web/templates/team/group_detail.html                                 — group detail page
Create: web/templates/team/group_section_members.html                        — members section
Create: web/templates/team/group_section_permissions.html                    — permissions matrix section
Create: web/templates/team/group_section_funnels.html                        — group funnels section
Create: web/templates/team/group_section_view_profiles.html                  — view profiles section
Create: web/templates/team/group_section_load_balance.html                   — load balance section

# Testing & supporting
Create: rest/team.http                                                       — HTTP file for manual testing
```

---

## Phase 1 — Load Balance backend

### Task 1: Migration + domain/infra Active field

**Files:**
- Create: `migrations/000051_add_active_to_load_balance_configs.up.sql`
- Create: `migrations/000051_add_active_to_load_balance_configs.down.sql`
- Modify: `internal/auth/domain/load_balance.go`
- Modify: `internal/auth/infrastructure/gorm_load_balance_repo.go`

- [ ] **Step 1: Write the UP migration**

```sql
-- migrations/000051_add_active_to_load_balance_configs.up.sql
ALTER TABLE load_balance_configs
    ADD COLUMN active BOOLEAN NOT NULL DEFAULT TRUE;
```

- [ ] **Step 2: Write the DOWN migration**

```sql
-- migrations/000051_add_active_to_load_balance_configs.down.sql
ALTER TABLE load_balance_configs DROP COLUMN active;
```

- [ ] **Step 3: Extend the domain struct**

In `internal/auth/domain/load_balance.go`, add `Active bool` to `LoadBalanceConfig` and default to `true` in `NewLoadBalanceConfig`:

```go
type LoadBalanceConfig struct {
	ID        string
	TenantID  string
	GroupID   string
	Algorithm LoadBalanceAlgorithm
	LastIndex int
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

In `NewLoadBalanceConfig` set `Active: true`.

- [ ] **Step 4: Update the GORM model and mappers**

In `internal/auth/infrastructure/gorm_load_balance_repo.go`:
- Add `Active bool \`gorm:"column:active;not null;default:true"\`` to `loadBalanceConfigModel`.
- Copy the field in `loadBalanceToModel` and `loadBalanceToDomain`.
- Add `"active"` to the `DoUpdates` list inside `CreateOrUpdate` so the flag is persisted on upsert.

- [ ] **Step 5: Run migration + build**

```bash
make migrate-up
go build ./...
```

Expected: migration applies cleanly, build passes.

- [ ] **Step 6: Commit**

```bash
git add migrations/000051_add_active_to_load_balance_configs.*.sql \
        internal/auth/domain/load_balance.go \
        internal/auth/infrastructure/gorm_load_balance_repo.go
git commit -m "feat(F08): add active flag to load balance configs"
```

---

### Task 2: ManageLoadBalanceUseCase (tests first)

**Files:**
- Create: `internal/auth/application/manage_load_balance.go`
- Create: `internal/auth/application/manage_load_balance_test.go`

- [ ] **Step 1: Define the domain-level contract for cross-module group validation**

Add to `internal/auth/application/manage_load_balance.go`:

```go
package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

// GroupTenantChecker validates that a group belongs to the given tenant.
// This avoids a hard import of the permission domain from auth.
type GroupTenantChecker interface {
	BelongsToTenant(ctx context.Context, tenantID, groupID string) (bool, error)
}

var ErrGroupNotInTenant = errors.New("group does not belong to tenant")

type SetLoadBalanceInput struct {
	TenantID  string
	GroupID   string
	Algorithm domain.LoadBalanceAlgorithm
	Active    bool
}

type ManageLoadBalanceUseCase struct {
	repo         domain.LoadBalanceConfigRepository
	groupChecker GroupTenantChecker
}

func NewManageLoadBalanceUseCase(
	repo domain.LoadBalanceConfigRepository,
	groupChecker GroupTenantChecker,
) *ManageLoadBalanceUseCase {
	return &ManageLoadBalanceUseCase{repo: repo, groupChecker: groupChecker}
}

func (uc *ManageLoadBalanceUseCase) GetByGroup(ctx context.Context, tenantID, groupID string) (*domain.LoadBalanceConfig, error) {
	ok, err := uc.groupChecker.BelongsToTenant(ctx, tenantID, groupID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrGroupNotInTenant
	}
	return uc.repo.FindByGroupID(ctx, tenantID, groupID)
}

func (uc *ManageLoadBalanceUseCase) SetByGroup(ctx context.Context, in SetLoadBalanceInput) (*domain.LoadBalanceConfig, error) {
	ok, err := uc.groupChecker.BelongsToTenant(ctx, in.TenantID, in.GroupID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrGroupNotInTenant
	}

	existing, err := uc.repo.FindByGroupID(ctx, in.TenantID, in.GroupID)
	if err != nil && !errors.Is(err, domain.ErrLoadBalanceNotFound) {
		return nil, err
	}

	var cfg *domain.LoadBalanceConfig
	if existing == nil {
		cfg, err = domain.NewLoadBalanceConfig(uuid.NewString(), in.TenantID, in.GroupID, in.Algorithm)
		if err != nil {
			return nil, err
		}
	} else {
		cfg = existing
		cfg.Algorithm = in.Algorithm
	}
	cfg.Active = in.Active

	if err := uc.repo.CreateOrUpdate(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
```

- [ ] **Step 2: Write the unit tests (failing first)**

Create `internal/auth/application/manage_load_balance_test.go`. Use `testify/mock` (follow `internal/auth/application/mocks_test.go` style). Tests required:

```go
package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

type mockGroupChecker struct{ mock.Mock }

func (m *mockGroupChecker) BelongsToTenant(ctx context.Context, tenantID, groupID string) (bool, error) {
	args := m.Called(ctx, tenantID, groupID)
	return args.Bool(0), args.Error(1)
}

type mockLBRepo struct{ mock.Mock }

func (m *mockLBRepo) CreateOrUpdate(ctx context.Context, cfg *domain.LoadBalanceConfig) error {
	return m.Called(ctx, cfg).Error(0)
}
func (m *mockLBRepo) FindByGroupID(ctx context.Context, tenantID, groupID string) (*domain.LoadBalanceConfig, error) {
	args := m.Called(ctx, tenantID, groupID)
	if cfg, ok := args.Get(0).(*domain.LoadBalanceConfig); ok {
		return cfg, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockLBRepo) FindByTenantID(ctx context.Context, tenantID string) ([]*domain.LoadBalanceConfig, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*domain.LoadBalanceConfig), args.Error(1)
}
func (m *mockLBRepo) Update(ctx context.Context, cfg *domain.LoadBalanceConfig) error {
	return m.Called(ctx, cfg).Error(0)
}

func TestSetByGroup_CreatesWhenMissing(t *testing.T) {
	repo := new(mockLBRepo)
	chk := new(mockGroupChecker)
	uc := NewManageLoadBalanceUseCase(repo, chk)

	chk.On("BelongsToTenant", mock.Anything, "t1", "g1").Return(true, nil)
	repo.On("FindByGroupID", mock.Anything, "t1", "g1").Return(nil, domain.ErrLoadBalanceNotFound)
	repo.On("CreateOrUpdate", mock.Anything, mock.Anything).Return(nil)

	cfg, err := uc.SetByGroup(context.Background(), SetLoadBalanceInput{
		TenantID: "t1", GroupID: "g1", Algorithm: domain.AlgorithmRoundRobin, Active: true,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.AlgorithmRoundRobin, cfg.Algorithm)
	assert.True(t, cfg.Active)
}

func TestSetByGroup_UpdatesExisting(t *testing.T) {
	repo := new(mockLBRepo)
	chk := new(mockGroupChecker)
	uc := NewManageLoadBalanceUseCase(repo, chk)

	existing := &domain.LoadBalanceConfig{ID: "id", TenantID: "t1", GroupID: "g1", Algorithm: domain.AlgorithmRoundRobin, Active: true}
	chk.On("BelongsToTenant", mock.Anything, "t1", "g1").Return(true, nil)
	repo.On("FindByGroupID", mock.Anything, "t1", "g1").Return(existing, nil)
	repo.On("CreateOrUpdate", mock.Anything, mock.Anything).Return(nil)

	cfg, err := uc.SetByGroup(context.Background(), SetLoadBalanceInput{
		TenantID: "t1", GroupID: "g1", Algorithm: domain.AlgorithmRandom, Active: false,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.AlgorithmRandom, cfg.Algorithm)
	assert.False(t, cfg.Active)
}

func TestSetByGroup_InvalidAlgorithm(t *testing.T) {
	repo := new(mockLBRepo)
	chk := new(mockGroupChecker)
	uc := NewManageLoadBalanceUseCase(repo, chk)

	chk.On("BelongsToTenant", mock.Anything, "t1", "g1").Return(true, nil)
	repo.On("FindByGroupID", mock.Anything, "t1", "g1").Return(nil, domain.ErrLoadBalanceNotFound)

	_, err := uc.SetByGroup(context.Background(), SetLoadBalanceInput{
		TenantID: "t1", GroupID: "g1", Algorithm: "weird", Active: true,
	})
	assert.ErrorIs(t, err, domain.ErrInvalidAlgorithm)
}

func TestSetByGroup_GroupNotInTenant(t *testing.T) {
	repo := new(mockLBRepo)
	chk := new(mockGroupChecker)
	uc := NewManageLoadBalanceUseCase(repo, chk)

	chk.On("BelongsToTenant", mock.Anything, "t1", "g1").Return(false, nil)

	_, err := uc.SetByGroup(context.Background(), SetLoadBalanceInput{
		TenantID: "t1", GroupID: "g1", Algorithm: domain.AlgorithmRoundRobin, Active: true,
	})
	assert.ErrorIs(t, err, ErrGroupNotInTenant)
}

func TestGetByGroup_NotFound(t *testing.T) {
	repo := new(mockLBRepo)
	chk := new(mockGroupChecker)
	uc := NewManageLoadBalanceUseCase(repo, chk)

	chk.On("BelongsToTenant", mock.Anything, "t1", "g1").Return(true, nil)
	repo.On("FindByGroupID", mock.Anything, "t1", "g1").Return(nil, domain.ErrLoadBalanceNotFound)

	_, err := uc.GetByGroup(context.Background(), "t1", "g1")
	assert.ErrorIs(t, err, domain.ErrLoadBalanceNotFound)
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/auth/application/... -run TestSetByGroup -v
go test ./internal/auth/application/... -run TestGetByGroup -v
```

Expected: all pass, coverage > 80% for the new use case.

- [ ] **Step 4: Commit**

```bash
git add internal/auth/application/manage_load_balance.go internal/auth/application/manage_load_balance_test.go
git commit -m "feat(F08): add ManageLoadBalanceUseCase"
```

---

### Task 3: GroupTenantChecker adapter + wiring in auth.Module

**Files:**
- Create: `internal/auth/infrastructure/group_tenant_checker.go`
- Modify: `internal/auth/module.go`

- [ ] **Step 1: Create a lightweight adapter using the existing `permission_groups` table**

```go
// internal/auth/infrastructure/group_tenant_checker.go
package infrastructure

import (
	"context"

	"gorm.io/gorm"
)

type GroupTenantCheckerAdapter struct {
	db *gorm.DB
}

func NewGroupTenantCheckerAdapter(db *gorm.DB) *GroupTenantCheckerAdapter {
	return &GroupTenantCheckerAdapter{db: db}
}

func (a *GroupTenantCheckerAdapter) BelongsToTenant(ctx context.Context, tenantID, groupID string) (bool, error) {
	var count int64
	err := a.db.WithContext(ctx).
		Table("permission_groups").
		Where("id = ? AND tenant_id = ?", groupID, tenantID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
```

- [ ] **Step 2: Wire the use case in `internal/auth/module.go`**

Add fields to `Module`:

```go
type Module struct {
	handler       *authhttp.Handler
	inviteUC      *application.InviteUserUseCase
	manageUsersUC *application.ManageUsersUseCase
	loginUC       *application.LoginUseCase
	loadBalanceUC *application.ManageLoadBalanceUseCase
}
```

In `NewModule`:

```go
loadBalanceRepo := infrastructure.NewGormLoadBalanceConfigRepository(db)
groupChecker := infrastructure.NewGroupTenantCheckerAdapter(db)
loadBalanceUC := application.NewManageLoadBalanceUseCase(loadBalanceRepo, groupChecker)
```

Return it in the struct literal. Add an exported accessor:

```go
func (m *Module) LoadBalanceUseCase() *application.ManageLoadBalanceUseCase { return m.loadBalanceUC }
```

- [ ] **Step 3: Build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/auth/infrastructure/group_tenant_checker.go internal/auth/module.go
git commit -m "feat(F08): wire ManageLoadBalanceUseCase in auth module"
```

---

### Task 4: Load balance HTTP endpoints in permission module

**Files:**
- Modify: `internal/permission/interfaces/http/handler.go`
- Modify: `internal/permission/interfaces/http/routes.go`
- Modify: `internal/permission/module.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Extend Handler to receive the LB use case**

In `handler.go`, add the field and constructor arg:

```go
type Handler struct {
	// ... existing
	loadBalanceUC *authapp.ManageLoadBalanceUseCase
	// ...
}
```

Import path: `authapp "github.com/sasrgita/crm-juridico/internal/auth/application"`.

Update `NewHandler` signature to accept `loadBalanceUC *authapp.ManageLoadBalanceUseCase` and store it.

- [ ] **Step 2: Add the two handler methods**

Append to `handler.go`:

```go
// --- Load Balance ---

func (h *Handler) GetLoadBalance(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")

	cfg, err := h.loadBalanceUC.GetByGroup(c.Request.Context(), tenantID, groupID)
	if err != nil {
		if errors.Is(err, authapp.ErrGroupNotInTenant) {
			c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
			return
		}
		if errors.Is(err, authdomain.ErrLoadBalanceNotFound) {
			// Return 200 with empty body so the UI can render defaults.
			c.JSON(http.StatusOK, gin.H{"algorithm": "round_robin", "active": false, "exists": false})
			return
		}
		h.log.Error("failed to get load balance", zap.String("group_id", groupID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get load balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"algorithm": string(cfg.Algorithm),
		"active":    cfg.Active,
		"exists":    true,
	})
}

func (h *Handler) SetLoadBalance(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")

	var req struct {
		Algorithm string `json:"algorithm" form:"algorithm"`
		Active    bool   `json:"active" form:"active"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg, err := h.loadBalanceUC.SetByGroup(c.Request.Context(), authapp.SetLoadBalanceInput{
		TenantID:  tenantID,
		GroupID:   groupID,
		Algorithm: authdomain.LoadBalanceAlgorithm(req.Algorithm),
		Active:    req.Active,
	})
	if err != nil {
		if errors.Is(err, authapp.ErrGroupNotInTenant) {
			c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
			return
		}
		if errors.Is(err, authdomain.ErrInvalidAlgorithm) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid algorithm"})
			return
		}
		h.log.Error("failed to set load balance", zap.String("group_id", groupID), zap.Error(err))
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "failed to set load balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"algorithm": string(cfg.Algorithm),
		"active":    cfg.Active,
	})
}
```

Add imports: `"errors"`, `authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"`.

- [ ] **Step 3: Register the routes**

In `routes.go`, inside the `groups` route group, add:

```go
groups.GET("/:id/load-balance", requirePerm("groups", "manage"), h.GetLoadBalance)
groups.PUT("/:id/load-balance", requirePerm("groups", "manage"), h.SetLoadBalance)
```

- [ ] **Step 4: Propagate the use case through the module**

Update `internal/permission/module.go`:

```go
func NewModule(
	db *gorm.DB,
	log *zap.Logger,
	loadBalanceUC *authapp.ManageLoadBalanceUseCase,
) *Module {
	// existing wiring...
	handler := permhttp.NewHandler(
		createGroupUC, getGroupUC, updateGroupUC, listGroupsUC, deleteGroupUC,
		manageMembersUC, managePermsUC, manageVPUC, manageGFUC,
		loadBalanceUC,
		log,
	)
	// ...
}
```

Import `authapp "github.com/sasrgita/crm-juridico/internal/auth/application"`.

- [ ] **Step 5: Update wiring in `cmd/api/main.go`**

Pass the LB use case when building the permission module:

```go
permissionMod := permission.NewModule(db, log, authMod.LoadBalanceUseCase())
```

- [ ] **Step 6: Write handler test for 400/404/200**

Append to `internal/permission/interfaces/http/handler_test.go` (or create a new test file) with 3 tests using a stub `ManageLoadBalanceUseCase` wrapper. Cover:
- 200 OK when group exists (returns defaults if no config)
- 400 when algorithm is invalid
- 404 when group does not belong to tenant

Follow the existing test style in `internal/permission/interfaces/http/handler_test.go`.

- [ ] **Step 7: Run tests + build**

```bash
go test ./internal/permission/...
go build ./...
```

Expected: pass.

- [ ] **Step 8: Commit**

```bash
git add internal/permission/interfaces/http/handler.go \
        internal/permission/interfaces/http/routes.go \
        internal/permission/interfaces/http/handler_test.go \
        internal/permission/module.go \
        cmd/api/main.go
git commit -m "feat(F08): add load-balance endpoints (GET/PUT /tenant/groups/:id/load-balance)"
```

---

## Phase 2 — Sidebar + shared shell

### Task 5: Sidebar item + shell template

**Files:**
- Modify: `web/templates/partials/tenant_sidebar.html`
- Create: `web/templates/team/shell.html`

- [ ] **Step 1: Add "Equipe" item to the sidebar**

Insert **after** the Leads link and **before** Produtos in `tenant_sidebar.html`:

```html
<a href="/tenant/team" class="{{if .ActiveNav}}{{if eq .ActiveNav "team"}}active{{end}}{{end}}">
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M17 20h5v-2a4 4 0 00-3-3.87M9 20H4v-2a4 4 0 013-3.87m6-5.13a4 4 0 11-8 0 4 4 0 018 0zm6 0a4 4 0 11-8 0 4 4 0 018 0z"/></svg>
    Equipe
</a>
```

- [ ] **Step 2: Create the shared shell template**

```html
{{define "team/shell.html"}}
<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Equipe — CRM Juridico</title>
    <link rel="stylesheet" href="/static/css/main.css">
    <script src="https://unpkg.com/htmx.org@2.0.4" integrity="sha384-HGfztofotfshcF7+8n44JQL2oJmowVChPTg48S+jvZoztPfvwD79OC/LTtG6dMp+" crossorigin="anonymous"></script>
</head>
<body class="no-auto-close-modals">
<div class="admin-layout">
    {{template "partials/tenant_sidebar.html" (dict "ActiveNav" "team")}}
    <main class="admin-content">
        <div class="page-header">
            <div><h1>Equipe</h1></div>
        </div>
        <div class="tabs">
            <a href="/tenant/team/users"
               class="tab {{if eq .ActiveTab "users"}}active{{end}}">Usuários</a>
            <a href="/tenant/team/groups"
               class="tab {{if eq .ActiveTab "groups"}}active{{end}}">Grupos</a>
        </div>
        <div id="tab-content">
            {{template .ContentTemplate .}}
        </div>
    </main>
</div>

<div id="team-modal" class="modal-overlay" style="display:none"
     onclick="if(event.target===this)closeModal('team-modal')">
    <div class="modal-card modal-card-lg">
        <h2 id="team-modal-title">—</h2>
        <div id="team-modal-body"><p class="text-muted">Carregando...</p></div>
    </div>
</div>

<script src="/static/js/admin.js"></script>
<script>
document.body.addEventListener("refreshTeam", function() {
    closeModal('team-modal');
    htmx.trigger('#users-block', 'load');
    htmx.trigger('#groups-block', 'load');
});
</script>
</body>
</html>
{{end}}
```

The shell takes `ActiveTab` and `ContentTemplate` values; each page supplies the actual fragment template name.

- [ ] **Step 3: Commit**

```bash
git add web/templates/partials/tenant_sidebar.html web/templates/team/shell.html
git commit -m "feat(F08): add Equipe sidebar item and team shell template"
```

---

## Phase 3 — Users tab

### Task 6: Auth PageHandler scaffold + routes

**Files:**
- Create: `internal/auth/interfaces/http/page_handler.go`
- Modify: `internal/auth/module.go`

- [ ] **Step 1: Write page handler scaffold**

```go
// internal/auth/interfaces/http/page_handler.go
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/auth/application"
	permapp "github.com/sasrgita/crm-juridico/internal/permission/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// PageHandler renders HTML pages for the "Equipe > Usuários" tab.
type PageHandler struct {
	inviteUC     *application.InviteUserUseCase
	manageUsers  *application.ManageUsersUseCase
	listGroupsUC *permapp.ListGroupsUseCase
	manageUserPerms *permapp.ManagePermissionsUseCase
	resolverUC   *permapp.ResolvePermissionUseCase
	log          *zap.Logger
}

func NewPageHandler(
	inviteUC *application.InviteUserUseCase,
	manageUsers *application.ManageUsersUseCase,
	listGroupsUC *permapp.ListGroupsUseCase,
	manageUserPerms *permapp.ManagePermissionsUseCase,
	resolverUC *permapp.ResolvePermissionUseCase,
	log *zap.Logger,
) *PageHandler {
	return &PageHandler{
		inviteUC: inviteUC, manageUsers: manageUsers,
		listGroupsUC: listGroupsUC, manageUserPerms: manageUserPerms,
		resolverUC: resolverUC, log: log,
	}
}

func (h *PageHandler) RedirectToUsers(c *gin.Context) {
	c.Redirect(http.StatusFound, "/tenant/team/users")
}

func (h *PageHandler) UsersPage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	users, err := h.manageUsers.ListTenantUsers(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list users", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	invites, _ := h.inviteUC.ListInvites(c.Request.Context(), tenantID)

	c.HTML(http.StatusOK, "team/shell.html", gin.H{
		"ActiveTab":       "users",
		"ContentTemplate": "team/users_page.html",
		"Users":           users,
		"Invites":         invites,
	})
}
```

(Leave method stubs for the other endpoints — they return `c.Status(http.StatusNotImplemented)` for now. The following tasks implement them in place.)

- [ ] **Step 2: Register routes in `auth/module.go`**

Add `pageHandler` field + constructor wiring, then inside `registerTenantRoutes`:

```go
tenantGroup.GET("/team", m.pageHandler.RedirectToUsers)
tenantGroup.GET("/team/users", mw.RequirePermission("users", "read"), m.pageHandler.UsersPage)
tenantGroup.GET("/team/users/table", mw.RequirePermission("users", "read"), m.pageHandler.UsersTable)
tenantGroup.GET("/team/users/:id/permissions-modal", mw.RequirePermission("users", "update"), m.pageHandler.UserPermissionsModal)
tenantGroup.GET("/team/users/:id/whatsapp-modal", mw.RequirePermission("users", "update"), m.pageHandler.UserWhatsAppModal)
tenantGroup.GET("/team/invites/new-modal", mw.RequirePermission("invites", "create"), m.pageHandler.InviteNewModal)
tenantGroup.POST("/team/invites", mw.RequirePermission("invites", "create"), m.pageHandler.CreateInvite)
```

Add the new use cases as constructor parameters to `auth.NewModule`. `permission.Module` needs to expose them via accessors first; then `main.go` builds `permission` before `auth` and passes the accessors in. Signature:

```go
func NewModule(
	db *gorm.DB,
	tenantRepo tenantdomain.TenantRepository,
	listGroupsUC *permapp.ListGroupsUseCase,
	manageUserPerms *permapp.ManagePermissionsUseCase,
	resolverUC *permapp.ResolvePermissionUseCase,
	jwtSecret string,
	jwtExpiration time.Duration,
	secureCookie bool,
) *Module { ... }
```

Then build `permission` first in `main.go`, and pass its accessors into auth. Add accessors in `permission/module.go`:

```go
func (m *Module) ListGroupsUC() *application.ListGroupsUseCase { return m.listGroupsUC }
func (m *Module) ManagePermissionsUC() *application.ManagePermissionsUseCase { return m.managePermsUC }
```

(`resolverUC` already exposed via `Resolver()`.)

Requires also storing those use cases as fields on `permission.Module` (they are currently local variables in `NewModule`).

- [ ] **Step 3: Adjust `cmd/api/main.go`**

Order: build `auth` minimally (for login only), then `permission`, then call `authMod.AttachPermissionDeps(permissionMod.ListGroupsUC(), permissionMod.ManagePermissionsUC(), permissionMod.Resolver())`. Simpler: keep `NewModule` signature large, but order construction in `main.go` to build `permission` before `auth`. We refactor `main.go` accordingly.

- [ ] **Step 4: Build**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/auth/interfaces/http/page_handler.go \
        internal/auth/module.go \
        internal/permission/module.go \
        cmd/api/main.go
git commit -m "feat(F08): scaffold auth PageHandler and register Equipe HTML routes"
```

---

### Task 7: Users page template + fragment

**Files:**
- Create: `web/templates/team/users_page.html`
- Create: `web/templates/team/users_table.html`

- [ ] **Step 1: Write `users_page.html`**

```html
{{define "team/users_page.html"}}
<div class="admin-table-wrapper">
    <div class="filters-bar">
        <span class="muted">{{len .Users}} usuário(s)</span>
        <button class="btn btn-primary"
                hx-get="/tenant/team/invites/new-modal"
                hx-target="#team-modal-body"
                hx-swap="innerHTML"
                onclick="document.getElementById('team-modal-title').textContent='Convidar Usuário';openModal('team-modal')">
            + Convidar Usuário
        </button>
    </div>
    <div id="users-block"
         hx-get="/tenant/team/users/table"
         hx-trigger="load, refreshTeam from:body"
         hx-swap="innerHTML">
        {{template "team/users_table.html" .}}
    </div>
</div>
{{end}}
```

- [ ] **Step 2: Write `users_table.html`**

```html
{{define "team/users_table.html"}}
<table class="admin-table">
    <thead>
        <tr><th>Nome</th><th>Papel</th><th>Grupos</th><th>WhatsApp</th><th>Ações</th></tr>
    </thead>
    <tbody>
        {{range .Users}}
        <tr>
            <td>
                <div class="cell-primary">{{.Name}}</div>
                <div class="cell-secondary muted">{{.Email}}</div>
            </td>
            <td>
                {{if .IsOwner}}
                    <span class="badge badge-gold">Owner</span>
                {{else}}
                    <span class="badge">Membro</span>
                {{end}}
            </td>
            <td>—</td>
            <td>
                {{if .WhatsAppID}}{{.WhatsAppID}}{{else}}—{{end}}
                <button class="btn-icon"
                        hx-get="/tenant/team/users/{{.ID}}/whatsapp-modal"
                        hx-target="#team-modal-body"
                        onclick="document.getElementById('team-modal-title').textContent='Editar WhatsApp';openModal('team-modal')"
                        title="Editar WhatsApp">✏️</button>
            </td>
            <td>
                <button class="btn-icon"
                        hx-get="/tenant/team/users/{{.ID}}/permissions-modal"
                        hx-target="#team-modal-body"
                        onclick="document.getElementById('team-modal-title').textContent='Permissões';openModal('team-modal')"
                        title="Permissões">🔑</button>
                {{if not .IsOwner}}
                <button class="btn-icon"
                        hx-delete="/tenant/users/{{.ID}}"
                        hx-confirm="Remover este usuário do tenant?"
                        hx-target="#users-block"
                        hx-swap="innerHTML"
                        title="Remover">🗑️</button>
                {{end}}
            </td>
        </tr>
        {{else}}
        <tr><td colspan="5" class="empty-state">Convide seu primeiro membro.</td></tr>
        {{end}}
    </tbody>
</table>

<h3 style="margin-top:2rem">Convites pendentes</h3>
<table class="admin-table">
    <thead>
        <tr><th>Link</th><th>Expira em</th><th>Grupos</th><th>Ações</th></tr>
    </thead>
    <tbody>
        {{range .Invites}}
        <tr>
            <td><input type="text" readonly value="/invite/{{.Token}}" onclick="this.select();document.execCommand('copy')"/></td>
            <td>{{.ExpiresAt.Format "02/01/2006"}}</td>
            <td>{{len .GroupIDs}} grupo(s)</td>
            <td>
                <button class="btn-icon"
                        hx-delete="/tenant/invites/{{.ID}}"
                        hx-confirm="Revogar este convite?"
                        hx-target="#users-block"
                        hx-get="/tenant/team/users/table"
                        title="Revogar">🗑️</button>
            </td>
        </tr>
        {{else}}
        <tr><td colspan="4" class="empty-state">Nenhum convite pendente.</td></tr>
        {{end}}
    </tbody>
</table>
{{end}}
```

- [ ] **Step 3: Implement `UsersTable(c)` on PageHandler**

```go
func (h *PageHandler) UsersTable(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	users, _ := h.manageUsers.ListTenantUsers(c.Request.Context(), tenantID)
	invites, _ := h.inviteUC.ListInvites(c.Request.Context(), tenantID)
	c.HTML(http.StatusOK, "team/users_table.html", gin.H{"Users": users, "Invites": invites})
}
```

- [ ] **Step 4: Register templates in the template loader**

Verify the template loading configuration (usually in `cmd/api/main.go` or a `setupTemplates` helper) picks up `web/templates/team/*.html`. If templates are added via glob pattern, no change needed. Otherwise, add the `team/` directory.

- [ ] **Step 5: Smoke test**

```bash
go run ./cmd/api
```

Then in a browser: navigate to `/tenant/team/users`. Expect the page to render with users + an empty "Convites" section.

- [ ] **Step 6: Commit**

```bash
git add web/templates/team/users_page.html web/templates/team/users_table.html internal/auth/interfaces/http/page_handler.go
git commit -m "feat(F08): users tab page + table fragment"
```

---

### Task 8: Invite new modal + create

**Files:**
- Create: `web/templates/team/invite_new_modal.html`
- Create: `web/templates/team/invite_success.html`
- Modify: `internal/auth/interfaces/http/page_handler.go`

- [ ] **Step 1: Write `invite_new_modal.html`**

```html
{{define "team/invite_new_modal.html"}}
<form hx-post="/tenant/team/invites" hx-target="#team-modal-body" hx-swap="innerHTML">
    <label>Grupos de permissão</label>
    <div class="checkbox-list">
        {{range .Groups}}
        <label><input type="checkbox" name="group_ids" value="{{.ID}}"> {{.Name}}</label>
        {{else}}
        <p class="muted">Nenhum grupo cadastrado — o convite criará um usuário sem grupos.</p>
        {{end}}
    </div>
    <label>Expira em (dias)</label>
    <input type="number" name="expires_in_days" min="1" max="90" value="7" required>
    <div class="modal-actions">
        <button type="button" class="btn" onclick="closeModal('team-modal')">Cancelar</button>
        <button type="submit" class="btn btn-primary">Gerar convite</button>
    </div>
</form>
{{end}}
```

- [ ] **Step 2: Write `invite_success.html`**

```html
{{define "team/invite_success.html"}}
<div>
    <p>Convite gerado. Envie o link ao convidado:</p>
    <input type="text" readonly value="{{.InviteURL}}" onclick="this.select();document.execCommand('copy')" style="width:100%">
    <div class="modal-actions">
        <button class="btn" onclick="closeModal('team-modal')" hx-trigger="click"
                hx-on:click="htmx.trigger(document.body, 'refreshTeam')">Fechar</button>
    </div>
</div>
{{end}}
```

- [ ] **Step 3: Implement `InviteNewModal` and `CreateInvite`**

```go
func (h *PageHandler) InviteNewModal(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groups, _ := h.listGroupsUC.Execute(c.Request.Context(), tenantID)
	c.HTML(http.StatusOK, "team/invite_new_modal.html", gin.H{"Groups": groups})
}

func (h *PageHandler) CreateInvite(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())

	groupIDs := c.PostFormArray("group_ids")
	daysStr := c.PostForm("expires_in_days")
	days := 7
	if n, err := strconv.Atoi(daysStr); err == nil && n > 0 {
		days = n
	}

	out, err := h.inviteUC.GenerateInvite(c.Request.Context(), tenantID, claims.UserID, groupIDs, days)
	if err != nil {
		h.log.Error("failed to generate invite", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "team/invite_new_modal.html", gin.H{"Error": "Falha ao gerar convite"})
		return
	}

	scheme := "http"
	if c.Request.TLS != nil { scheme = "https" }
	inviteURL := scheme + "://" + c.Request.Host + "/invite/" + out.Token
	c.HTML(http.StatusOK, "team/invite_success.html", gin.H{"InviteURL": inviteURL})
}
```

Add import: `"strconv"`.

- [ ] **Step 4: Smoke test**

Click "+ Convidar Usuário", select a group, submit. Expect success view with copyable URL, and on close the users table refreshes.

- [ ] **Step 5: Commit**

```bash
git add web/templates/team/invite_new_modal.html web/templates/team/invite_success.html internal/auth/interfaces/http/page_handler.go
git commit -m "feat(F08): invite creation modal"
```

---

### Task 9: User permissions modal (individual override)

**Files:**
- Create: `web/templates/team/user_permissions_modal.html`
- Modify: `internal/auth/interfaces/http/page_handler.go`

- [ ] **Step 1: Write the modal template**

```html
{{define "team/user_permissions_modal.html"}}
<div>
    <h3>Permissões herdadas de grupos</h3>
    {{if .InheritedPerms}}
    <div class="chip-list">
        {{range .InheritedPerms}}
            <span class="chip chip-muted">{{.Resource}}:{{.Action}}</span>
        {{end}}
    </div>
    {{else}}
    <p class="muted">Este usuário não pertence a nenhum grupo.</p>
    {{end}}

    <h3 style="margin-top:1.5rem">Permissões individuais extras</h3>
    <form hx-put="/tenant/users/{{.UserID}}/permissions"
          hx-ext="json-enc"
          hx-target="#team-modal-body"
          hx-swap="innerHTML">
        <div class="checkbox-list">
            {{range .AllPerms}}
            <label>
                <input type="checkbox"
                       name="permissions"
                       value='{"resource":"{{.Resource}}","action":"{{.Action}}"}'
                       {{if .Granted}}checked{{end}}>
                {{.Resource}}:{{.Action}}
            </label>
            {{end}}
        </div>
        <div class="modal-actions">
            <button type="button" class="btn" onclick="closeModal('team-modal')">Cancelar</button>
            <button type="submit" class="btn btn-primary">Salvar</button>
        </div>
    </form>
</div>
{{end}}
```

**Note:** since the existing `SetUserPermissions` endpoint expects a JSON array of `{resource, action}`, the modal uses `hx-ext="json-enc"`. Include the extension in the shell:

```html
<script src="https://unpkg.com/htmx-ext-json-enc@2.0.2/json-enc.js"></script>
```

Added to `team/shell.html` right after the core HTMX script.

Alternative (simpler, no extension): replace the endpoint with an HTML-friendly one `POST /tenant/team/users/:id/permissions` that reads `resource[]` and `action[]` form fields. Recommended — add it in step 3.

- [ ] **Step 2: Add an HTML-friendly permissions endpoint**

In `internal/auth/interfaces/http/page_handler.go`:

```go
func (h *PageHandler) SetUserPermissionsHTML(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	userID := c.Param("id")

	items := c.PostFormArray("perms") // "resource:action" strings
	inputs := make([]permapp.PermissionInput, 0, len(items))
	for _, it := range items {
		parts := strings.SplitN(it, ":", 2)
		if len(parts) != 2 {
			continue
		}
		inputs = append(inputs, permapp.PermissionInput{Resource: parts[0], Action: parts[1]})
	}

	if err := h.manageUserPerms.SetUserPermissions(c.Request.Context(), tenantID, userID, inputs); err != nil {
		h.log.Error("failed to set user permissions", zap.Error(err))
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	c.Header("HX-Trigger", "refreshTeam")
	c.Status(http.StatusNoContent)
}
```

Register route:

```go
tenantGroup.POST("/team/users/:id/permissions", mw.RequirePermission("users", "update"), m.pageHandler.SetUserPermissionsHTML)
```

Imports: `"strings"`, `permapp "github.com/sasrgita/crm-juridico/internal/permission/application"`.

Update the modal template form fields to use a single `perms` input:

```html
<input type="checkbox" name="perms" value="{{.Resource}}:{{.Action}}" {{if .Granted}}checked{{end}}>
```

And the form submits to `/tenant/team/users/:id/permissions` (standard form POST, no json-enc needed).

- [ ] **Step 3: Implement `UserPermissionsModal(c)`**

```go
// Catalog of permissions available in the system (resource:action).
var permissionCatalog = []struct{ Resource, Action string }{
	{"leads", "read"}, {"leads", "create"}, {"leads", "update"}, {"leads", "delete"},
	{"users", "read"}, {"users", "update"}, {"users", "delete"},
	{"groups", "manage"},
	{"funnels", "read"}, {"funnels", "customize"},
	{"automations", "manage"},
	{"products", "manage"},
	{"specialists", "manage"},
	{"invites", "create"}, {"invites", "read"}, {"invites", "delete"},
	{"settings", "manage"},
}

type modalPerm struct {
	Resource string
	Action   string
	Granted  bool
}

func (h *PageHandler) UserPermissionsModal(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	userID := c.Param("id")

	// Inherited (from groups) — computed via resolver per action:
	granted, _ := h.manageUserPerms.GetUserPermissions(c.Request.Context(), tenantID, userID)
	grantedSet := make(map[string]bool)
	for _, p := range granted {
		grantedSet[p.Resource+":"+p.Action] = true
	}

	all := make([]modalPerm, len(permissionCatalog))
	for i, p := range permissionCatalog {
		all[i] = modalPerm{Resource: p.Resource, Action: p.Action, Granted: grantedSet[p.Resource+":"+p.Action]}
	}

	c.HTML(http.StatusOK, "team/user_permissions_modal.html", gin.H{
		"UserID":         userID,
		"AllPerms":       all,
		"InheritedPerms": nil, // Could be expanded later.
	})
}
```

- [ ] **Step 4: Smoke test**

Click 🔑 on a row. Expect the modal to load with checkboxes reflecting current individual permissions. Save and reopen — checkboxes persist.

- [ ] **Step 5: Commit**

```bash
git add web/templates/team/user_permissions_modal.html internal/auth/interfaces/http/page_handler.go internal/auth/module.go
git commit -m "feat(F08): user permissions modal"
```

---

### Task 10: WhatsApp ID modal

**Files:**
- Create: `web/templates/team/user_whatsapp_modal.html`
- Modify: `internal/auth/interfaces/http/page_handler.go`
- Modify: `internal/auth/module.go`

- [ ] **Step 1: Write modal template**

```html
{{define "team/user_whatsapp_modal.html"}}
<form hx-post="/tenant/team/users/{{.UserID}}/whatsapp" hx-target="#team-modal-body" hx-swap="innerHTML">
    <label>WhatsApp ID</label>
    <input type="text" name="whatsapp_id" value="{{.Current}}" placeholder="5511999999999" required>
    <p class="muted">Apenas dígitos (código do país + DDD + número).</p>
    <div class="modal-actions">
        <button type="button" class="btn" onclick="closeModal('team-modal')">Cancelar</button>
        <button type="submit" class="btn btn-primary">Salvar</button>
    </div>
</form>
{{end}}
```

- [ ] **Step 2: Handlers**

```go
func (h *PageHandler) UserWhatsAppModal(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	userID := c.Param("id")
	users, _ := h.manageUsers.ListTenantUsers(c.Request.Context(), tenantID)
	var current string
	for _, u := range users {
		if u.ID == userID { current = u.WhatsAppID; break }
	}
	c.HTML(http.StatusOK, "team/user_whatsapp_modal.html", gin.H{"UserID": userID, "Current": current})
}

func (h *PageHandler) SetUserWhatsApp(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	userID := c.Param("id")
	wid := strings.TrimSpace(c.PostForm("whatsapp_id"))
	if wid == "" {
		c.Status(http.StatusBadRequest); return
	}
	if err := h.manageUsers.SetWhatsAppID(c.Request.Context(), userID, tenantID, wid); err != nil {
		h.log.Error("failed to set whatsapp id", zap.Error(err))
		c.Status(http.StatusUnprocessableEntity); return
	}
	c.Header("HX-Trigger", "refreshTeam")
	c.Status(http.StatusNoContent)
}
```

- [ ] **Step 3: Register route**

```go
tenantGroup.POST("/team/users/:id/whatsapp", mw.RequirePermission("users", "update"), m.pageHandler.SetUserWhatsApp)
```

- [ ] **Step 4: Smoke test and commit**

```bash
git add web/templates/team/user_whatsapp_modal.html internal/auth/interfaces/http/page_handler.go internal/auth/module.go
git commit -m "feat(F08): WhatsApp ID modal for tenant users"
```

---

### Task 11: PageHandler unit tests (auth)

**Files:**
- Create: `internal/auth/interfaces/http/page_handler_test.go`

- [ ] **Step 1: Table-driven tests covering rendering & 401/403**

Test cases:
- `UsersPage` returns 200 and renders `team/shell.html` (check body contains "Equipe")
- `UsersTable` returns 200 and renders a table row for a seeded user (use mocked use cases)
- `InviteNewModal` returns 200 and includes the groups list
- `CreateInvite` returns 200 and the response body contains `/invite/` prefix
- 401 on missing auth (uses a router without the middleware)
- 403 on missing permission (uses a middleware that rejects)

Use the existing gin test pattern from `internal/permission/interfaces/http/handler_test.go`. Coverage target >= 80%.

- [ ] **Step 2: Run tests**

```bash
go test ./internal/auth/interfaces/http/... -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/auth/interfaces/http/page_handler_test.go
git commit -m "test(F08): auth PageHandler unit tests"
```

---

## Phase 4 — Groups tab + detail

### Task 12: Permission PageHandler scaffold + groups list

**Files:**
- Create: `internal/permission/interfaces/http/page_handler.go`
- Modify: `internal/permission/interfaces/http/routes.go`
- Modify: `internal/permission/module.go`
- Modify: `cmd/api/main.go`
- Create: `web/templates/team/groups_page.html`
- Create: `web/templates/team/groups_table.html`
- Create: `web/templates/team/group_new_modal.html`

- [ ] **Step 1: Scaffold `page_handler.go`**

```go
package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	authapp "github.com/sasrgita/crm-juridico/internal/auth/application"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	funnelapp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

type TenantUsersLister interface {
	ListTenantUsers(ctx gin.H, tenantID string) ([]authapp.UserOutput, error)
}

type PageHandler struct {
	createGroup     *application.CreateGroupUseCase
	listGroups      *application.ListGroupsUseCase
	getGroup        *application.GetGroupUseCase
	updateGroup     *application.UpdateGroupUseCase
	deleteGroup     *application.DeleteGroupUseCase
	manageMembers   *application.ManageMembersUseCase
	managePerms     *application.ManagePermissionsUseCase
	manageVP        *application.ManageViewProfilesUseCase
	manageGF        *application.ManageGroupFunnelsUseCase
	loadBalanceUC   *authapp.ManageLoadBalanceUseCase
	listFunnelsUC   *funnelapp.ListFunnelsUseCase
	columnRepo      funneldomain.ColumnRepository
	usersListUC     *authapp.ManageUsersUseCase
	log             *zap.Logger
}

// NewPageHandler constructs the permission-side HTML PageHandler.
func NewPageHandler(
	createGroup *application.CreateGroupUseCase,
	listGroups *application.ListGroupsUseCase,
	getGroup *application.GetGroupUseCase,
	updateGroup *application.UpdateGroupUseCase,
	deleteGroup *application.DeleteGroupUseCase,
	manageMembers *application.ManageMembersUseCase,
	managePerms *application.ManagePermissionsUseCase,
	manageVP *application.ManageViewProfilesUseCase,
	manageGF *application.ManageGroupFunnelsUseCase,
	loadBalanceUC *authapp.ManageLoadBalanceUseCase,
	listFunnelsUC *funnelapp.ListFunnelsUseCase,
	columnRepo funneldomain.ColumnRepository,
	usersListUC *authapp.ManageUsersUseCase,
	log *zap.Logger,
) *PageHandler {
	return &PageHandler{
		createGroup: createGroup, listGroups: listGroups, getGroup: getGroup,
		updateGroup: updateGroup, deleteGroup: deleteGroup,
		manageMembers: manageMembers, managePerms: managePerms,
		manageVP: manageVP, manageGF: manageGF,
		loadBalanceUC: loadBalanceUC, listFunnelsUC: listFunnelsUC,
		columnRepo: columnRepo, usersListUC: usersListUC, log: log,
	}
}

func (h *PageHandler) GroupsPage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groups, err := h.listGroups.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list groups", zap.Error(err))
		c.Status(http.StatusInternalServerError); return
	}
	c.HTML(http.StatusOK, "team/shell.html", gin.H{
		"ActiveTab":       "groups",
		"ContentTemplate": "team/groups_page.html",
		"Groups":          groups,
	})
}

func (h *PageHandler) GroupsTable(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groups, _ := h.listGroups.Execute(c.Request.Context(), tenantID)
	c.HTML(http.StatusOK, "team/groups_table.html", gin.H{"Groups": groups})
}

func (h *PageHandler) GroupNewModal(c *gin.Context) {
	c.HTML(http.StatusOK, "team/group_new_modal.html", nil)
}

func (h *PageHandler) CreateGroupHTML(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	out, err := h.createGroup.Execute(c.Request.Context(), application.CreateGroupInput{
		TenantID: tenantID, Name: c.PostForm("name"), Description: c.PostForm("description"),
	})
	if err != nil {
		c.Status(http.StatusUnprocessableEntity); return
	}
	c.Header("HX-Redirect", "/tenant/team/groups/"+out.ID)
	c.Status(http.StatusOK)
}
```

- [ ] **Step 2: Register routes in `routes.go`**

```go
team := router.Group("/tenant/team")
team.Use(authMw, tenantMw)
{
	team.GET("/groups", requirePerm("groups", "manage"), h.page.GroupsPage)
	team.GET("/groups/table", requirePerm("groups", "manage"), h.page.GroupsTable)
	team.GET("/groups/new-modal", requirePerm("groups", "manage"), h.page.GroupNewModal)
	team.POST("/groups", requirePerm("groups", "manage"), h.page.CreateGroupHTML)
	team.GET("/groups/:id", requirePerm("groups", "manage"), h.page.GroupDetail)
	team.GET("/groups/:id/section/:name", requirePerm("groups", "manage"), h.page.GroupSection)
}
```

Note `h.page` is a new `*PageHandler` field added to the existing `Handler` struct, or a separate var in `RegisterRoutes`. Simplest: add `page *PageHandler` as a second arg to `RegisterRoutes` and a `Handler.page` field (initialized in `NewHandler` via a `WithPage` builder or accepted directly).

**Recommended:** change `RegisterRoutes` signature to accept a `*PageHandler`:

```go
func (h *Handler) RegisterRoutes(
	router *gin.Engine,
	page *PageHandler,
	authMw, tenantMw gin.HandlerFunc,
	requirePerm func(resource, action string) gin.HandlerFunc,
)
```

Update all callers.

- [ ] **Step 3: Wire in `permission/module.go`**

Add field `pageHandler *permhttp.PageHandler`. In `NewModule` accept new params (funnels, users use cases) and build the page handler. Expose constructor args:

```go
func NewModule(
	db *gorm.DB,
	log *zap.Logger,
	loadBalanceUC *authapp.ManageLoadBalanceUseCase,
	listFunnelsUC *funnelapp.ListFunnelsUseCase,
	columnRepo funneldomain.ColumnRepository,
	usersListUC *authapp.ManageUsersUseCase,
) *Module { ... }
```

Adjust `cmd/api/main.go` to pass these new dependencies.

- [ ] **Step 4: Write `groups_page.html`**

```html
{{define "team/groups_page.html"}}
<div class="admin-table-wrapper">
    <div class="filters-bar">
        <span class="muted">{{len .Groups}} grupo(s)</span>
        <button class="btn btn-primary"
                hx-get="/tenant/team/groups/new-modal"
                hx-target="#team-modal-body"
                onclick="document.getElementById('team-modal-title').textContent='Novo Grupo';openModal('team-modal')">
            + Novo Grupo
        </button>
    </div>
    <div id="groups-block"
         hx-get="/tenant/team/groups/table"
         hx-trigger="load, refreshTeam from:body"
         hx-swap="innerHTML">
        {{template "team/groups_table.html" .}}
    </div>
</div>
{{end}}
```

- [ ] **Step 5: Write `groups_table.html`**

```html
{{define "team/groups_table.html"}}
<table class="admin-table">
    <thead><tr><th>Nome</th><th>Descrição</th><th>Ações</th></tr></thead>
    <tbody>
        {{range .Groups}}
        <tr>
            <td><a href="/tenant/team/groups/{{.ID}}">{{.Name}}</a></td>
            <td class="muted">{{.Description}}</td>
            <td>
                <a class="btn-icon" href="/tenant/team/groups/{{.ID}}" title="Abrir">👁️</a>
                <button class="btn-icon"
                        hx-delete="/tenant/groups/{{.ID}}"
                        hx-confirm="Excluir este grupo?"
                        hx-target="#groups-block"
                        hx-get="/tenant/team/groups/table"
                        title="Excluir">🗑️</button>
            </td>
        </tr>
        {{else}}
        <tr><td colspan="3" class="empty-state">Crie seu primeiro grupo.</td></tr>
        {{end}}
    </tbody>
</table>
{{end}}
```

- [ ] **Step 6: Write `group_new_modal.html`**

```html
{{define "team/group_new_modal.html"}}
<form hx-post="/tenant/team/groups">
    <label>Nome</label>
    <input type="text" name="name" required>
    <label>Descrição</label>
    <textarea name="description" rows="3"></textarea>
    <div class="modal-actions">
        <button type="button" class="btn" onclick="closeModal('team-modal')">Cancelar</button>
        <button type="submit" class="btn btn-primary">Criar</button>
    </div>
</form>
{{end}}
```

- [ ] **Step 7: Build + smoke test + commit**

```bash
go build ./...
# Manual: open /tenant/team/groups, click Novo Grupo, create one, expect redirect to /tenant/team/groups/:id (detail page shows 404 until Task 13).
git add web/templates/team/groups_page.html web/templates/team/groups_table.html web/templates/team/group_new_modal.html \
        internal/permission/interfaces/http/page_handler.go internal/permission/interfaces/http/routes.go \
        internal/permission/module.go cmd/api/main.go
git commit -m "feat(F08): groups tab page + create modal"
```

---

### Task 13: Group detail page scaffold

**Files:**
- Create: `web/templates/team/group_detail.html`
- Modify: `internal/permission/interfaces/http/page_handler.go`

- [ ] **Step 1: Write `group_detail.html`**

```html
{{define "team/group_detail.html"}}
<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <title>Grupo — {{.Group.Name}}</title>
    <link rel="stylesheet" href="/static/css/main.css">
    <script src="https://unpkg.com/htmx.org@2.0.4" crossorigin="anonymous"></script>
</head>
<body class="no-auto-close-modals">
<div class="admin-layout">
    {{template "partials/tenant_sidebar.html" (dict "ActiveNav" "team")}}
    <main class="admin-content">
        <div class="page-header">
            <nav class="breadcrumbs"><a href="/tenant/team">Equipe</a> › <a href="/tenant/team/groups">Grupos</a> › {{.Group.Name}}</nav>
            <h1>{{.Group.Name}}</h1>
            <p class="muted">{{.Group.Description}}</p>
            <form hx-delete="/tenant/groups/{{.Group.ID}}"
                  hx-confirm="Excluir este grupo?"
                  hx-on::after-request="if(event.detail.successful){window.location='/tenant/team/groups';}">
                <button type="submit" class="btn btn-danger">🗑️ Excluir grupo</button>
            </form>
        </div>

        {{range .Sections}}
        <section class="card"
                 hx-get="/tenant/team/groups/{{$.Group.ID}}/section/{{.Slug}}"
                 hx-trigger="load"
                 hx-swap="innerHTML">
            <h2>{{.Title}}</h2>
            <p class="muted">Carregando...</p>
        </section>
        {{end}}
    </main>
</div>
<script src="/static/js/admin.js"></script>
</body>
</html>
{{end}}
```

- [ ] **Step 2: Implement `GroupDetail`**

```go
func (h *PageHandler) GroupDetail(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")

	g, err := h.getGroup.Execute(c.Request.Context(), tenantID, groupID)
	if err != nil {
		c.Status(http.StatusNotFound); return
	}

	sections := []struct{ Slug, Title string }{
		{"members", "👥 Membros"},
		{"permissions", "🔐 Permissões"},
		{"funnels", "🎯 Funis atribuídos"},
		{"view-profiles", "👁️ Perfis de visualização"},
		{"load-balance", "⚖️ Load Balance"},
	}
	c.HTML(http.StatusOK, "team/group_detail.html", gin.H{"Group": g, "Sections": sections})
}

func (h *PageHandler) GroupSection(c *gin.Context) {
	name := c.Param("name")
	switch name {
	case "members":
		h.sectionMembers(c)
	case "permissions":
		h.sectionPermissions(c)
	case "funnels":
		h.sectionFunnels(c)
	case "view-profiles":
		h.sectionViewProfiles(c)
	case "load-balance":
		h.sectionLoadBalance(c)
	default:
		c.Status(http.StatusNotFound)
	}
}
```

Each `sectionXxx` helper is implemented in Tasks 14–18.

- [ ] **Step 3: Commit (stub-only state is fine; sections are 501/empty until implemented)**

```bash
git add web/templates/team/group_detail.html internal/permission/interfaces/http/page_handler.go
git commit -m "feat(F08): group detail page scaffold"
```

---

### Task 14: Section — Members

**Files:**
- Create: `web/templates/team/group_section_members.html`
- Modify: `internal/permission/interfaces/http/page_handler.go`

- [ ] **Step 1: Template**

```html
{{define "team/group_section_members.html"}}
<ul class="chip-list" id="members-list-{{.GroupID}}">
    {{range .Members}}
    <li class="chip">{{.Name}} <span class="muted">({{.Email}})</span>
        <button class="btn-icon"
                hx-delete="/tenant/groups/{{$.GroupID}}/members/{{.ID}}"
                hx-target="closest section"
                hx-get="/tenant/team/groups/{{$.GroupID}}/section/members"
                title="Remover">✕</button>
    </li>
    {{else}}<li class="muted">Sem membros.</li>{{end}}
</ul>
<form hx-post="/tenant/groups/{{.GroupID}}/members"
      hx-target="closest section"
      hx-get="/tenant/team/groups/{{.GroupID}}/section/members">
    <select name="user_id" required>
        <option value="">— Selecione um usuário —</option>
        {{range .Candidates}}<option value="{{.ID}}">{{.Name}} ({{.Email}})</option>{{end}}
    </select>
    <button class="btn btn-primary" type="submit">Adicionar</button>
</form>
{{end}}
```

- [ ] **Step 2: Handler**

```go
func (h *PageHandler) sectionMembers(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")

	members, _ := h.manageMembers.ListMembers(c.Request.Context(), groupID)
	allUsers, _ := h.usersListUC.ListTenantUsers(c.Request.Context(), tenantID)

	memberIDs := make(map[string]bool, len(members))
	for _, m := range members { memberIDs[m.ID] = true }

	candidates := make([]authapp.UserOutput, 0, len(allUsers))
	for _, u := range allUsers {
		if !memberIDs[u.ID] { candidates = append(candidates, u) }
	}

	c.HTML(http.StatusOK, "team/group_section_members.html", gin.H{
		"GroupID": groupID, "Members": members, "Candidates": candidates,
	})
}
```

- [ ] **Step 3: Smoke test + commit**

```bash
git add web/templates/team/group_section_members.html internal/permission/interfaces/http/page_handler.go
git commit -m "feat(F08): group detail — members section"
```

---

### Task 15: Section — Permissions matrix

**Files:**
- Create: `web/templates/team/group_section_permissions.html`
- Modify: `internal/permission/interfaces/http/page_handler.go`

- [ ] **Step 1: Template**

```html
{{define "team/group_section_permissions.html"}}
<form hx-post="/tenant/team/groups/{{.GroupID}}/permissions"
      hx-target="closest section"
      hx-get="/tenant/team/groups/{{.GroupID}}/section/permissions">
    <table class="admin-table">
        <thead>
            <tr><th>Recurso</th>{{range .Actions}}<th>{{.}}</th>{{end}}</tr>
        </thead>
        <tbody>
            {{range .Rows}}
            <tr>
                <td>{{.Resource}}</td>
                {{range .Cells}}
                    <td>
                        {{if .Available}}
                            <input type="checkbox" name="perms" value="{{.Resource}}:{{.Action}}" {{if .Granted}}checked{{end}}>
                        {{else}}—{{end}}
                    </td>
                {{end}}
            </tr>
            {{end}}
        </tbody>
    </table>
    <div class="section-actions">
        <button class="btn btn-primary" type="submit">Salvar permissões</button>
    </div>
</form>
{{end}}
```

- [ ] **Step 2: Handler + save endpoint**

```go
var permActions = []string{"read", "create", "update", "delete", "manage"}
var permResources = []string{"leads", "users", "groups", "funnels", "automations", "products", "specialists", "invites", "settings"}
var permAvailable = map[string]map[string]bool{
	"leads":       {"read": true, "create": true, "update": true, "delete": true},
	"users":       {"read": true, "update": true, "delete": true},
	"groups":      {"manage": true},
	"funnels":     {"read": true, "update": true, "customize": true},
	"automations": {"manage": true},
	"products":    {"manage": true},
	"specialists": {"manage": true},
	"invites":     {"create": true, "read": true, "delete": true},
	"settings":    {"manage": true},
}

type permCell struct { Resource, Action string; Granted, Available bool }
type permRow struct { Resource string; Cells []permCell }

func (h *PageHandler) sectionPermissions(c *gin.Context) {
	groupID := c.Param("id")
	granted, _ := h.managePerms.GetGroupPermissions(c.Request.Context(), groupID)
	gset := make(map[string]bool)
	for _, p := range granted { gset[p.Resource+":"+p.Action] = true }

	rows := make([]permRow, 0, len(permResources))
	for _, r := range permResources {
		cells := make([]permCell, len(permActions))
		for i, a := range permActions {
			available := permAvailable[r][a]
			cells[i] = permCell{Resource: r, Action: a, Available: available, Granted: gset[r+":"+a]}
		}
		rows = append(rows, permRow{Resource: r, Cells: cells})
	}

	c.HTML(http.StatusOK, "team/group_section_permissions.html", gin.H{
		"GroupID": groupID, "Actions": permActions, "Rows": rows,
	})
}

func (h *PageHandler) SetGroupPermissionsHTML(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")
	items := c.PostFormArray("perms")
	inputs := make([]application.PermissionInput, 0, len(items))
	for _, it := range items {
		parts := strings.SplitN(it, ":", 2)
		if len(parts) != 2 { continue }
		inputs = append(inputs, application.PermissionInput{Resource: parts[0], Action: parts[1]})
	}
	if err := h.managePerms.SetGroupPermissions(c.Request.Context(), tenantID, groupID, inputs); err != nil {
		c.Status(http.StatusUnprocessableEntity); return
	}
	c.Status(http.StatusOK)
}
```

Add imports: `"strings"`. Register the POST route:

```go
team.POST("/groups/:id/permissions", requirePerm("groups", "manage"), h.page.SetGroupPermissionsHTML)
```

- [ ] **Step 3: Commit**

```bash
git add web/templates/team/group_section_permissions.html internal/permission/interfaces/http/page_handler.go internal/permission/interfaces/http/routes.go
git commit -m "feat(F08): group detail — permissions matrix section"
```

---

### Task 16: Section — Funnels

**Files:**
- Create: `web/templates/team/group_section_funnels.html`
- Modify: `internal/permission/interfaces/http/page_handler.go`

- [ ] **Step 1: Template**

```html
{{define "team/group_section_funnels.html"}}
<form hx-post="/tenant/team/groups/{{.GroupID}}/funnels"
      hx-target="closest section"
      hx-get="/tenant/team/groups/{{.GroupID}}/section/funnels">
    <ul class="checkbox-list">
        {{range .Funnels}}
        <li>
            <label>
                <input type="checkbox" name="funnel_ids" value="{{.ID}}" {{if .Assigned}}checked{{end}}>
                {{.Name}}
            </label>
        </li>
        {{else}}<li class="muted">Nenhum funil cadastrado.</li>{{end}}
    </ul>
    <div class="section-actions">
        <button class="btn btn-primary" type="submit">Salvar funis</button>
    </div>
</form>
{{end}}
```

- [ ] **Step 2: Handler + save endpoint**

```go
type funnelOpt struct { ID, Name string; Assigned bool }

func (h *PageHandler) sectionFunnels(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")

	funnels, _ := h.listFunnelsUC.Execute(c.Request.Context(), tenantID)
	assigned, _ := h.manageGF.ListByGroup(c.Request.Context(), groupID)
	aset := make(map[string]bool)
	for _, a := range assigned { aset[a.FunnelID] = true }

	opts := make([]funnelOpt, len(funnels))
	for i, f := range funnels { opts[i] = funnelOpt{ID: f.ID, Name: f.Name, Assigned: aset[f.ID]} }

	c.HTML(http.StatusOK, "team/group_section_funnels.html", gin.H{
		"GroupID": groupID, "Funnels": opts,
	})
}

func (h *PageHandler) SetGroupFunnelsHTML(c *gin.Context) {
	groupID := c.Param("id")
	funnelIDs := c.PostFormArray("funnel_ids")
	// Save each funnel with empty column_ids (meaning "all columns") to keep the save simple.
	for _, fid := range funnelIDs {
		if err := h.manageGF.SetGroupFunnel(c.Request.Context(), application.GroupFunnelInput{
			GroupID: groupID, FunnelID: fid, ColumnIDs: nil,
		}); err != nil {
			c.Status(http.StatusUnprocessableEntity); return
		}
	}
	c.Status(http.StatusOK)
}
```

Register:

```go
team.POST("/groups/:id/funnels", requirePerm("groups", "manage"), h.page.SetGroupFunnelsHTML)
```

- [ ] **Step 3: Commit**

```bash
git add web/templates/team/group_section_funnels.html internal/permission/interfaces/http/page_handler.go internal/permission/interfaces/http/routes.go
git commit -m "feat(F08): group detail — funnels section"
```

---

### Task 17: Section — View profiles

**Files:**
- Create: `web/templates/team/group_section_view_profiles.html`
- Modify: `internal/permission/interfaces/http/page_handler.go`

- [ ] **Step 1: Template**

```html
{{define "team/group_section_view_profiles.html"}}
{{if not .Funnels}}
<p class="muted">Atribua funis ao grupo para configurar perfis de visualização.</p>
{{end}}
{{range .Funnels}}
<div class="card-inner">
    <h3>{{.Name}}</h3>
    <form hx-post="/tenant/team/groups/{{$.GroupID}}/view-profiles/{{.ID}}"
          hx-target="closest section"
          hx-get="/tenant/team/groups/{{$.GroupID}}/section/view-profiles">
        <ul class="checkbox-list">
            {{range .Columns}}
            <li>
                <label>
                    <input type="checkbox" name="visible_columns" value="{{.ID}}" {{if .Visible}}checked{{end}}>
                    {{.Name}}
                </label>
            </li>
            {{end}}
        </ul>
        <button class="btn btn-primary" type="submit">Salvar</button>
    </form>
</div>
{{end}}
{{end}}
```

- [ ] **Step 2: Handler + save endpoint**

```go
type columnOpt struct { ID, Name string; Visible bool }
type funnelWithCols struct { ID, Name string; Columns []columnOpt }

func (h *PageHandler) sectionViewProfiles(c *gin.Context) {
	ctx := c.Request.Context()
	groupID := c.Param("id")

	assignedFunnels, _ := h.manageGF.ListByGroup(ctx, groupID)
	profiles, _ := h.manageVP.ListByGroup(ctx, groupID)
	pset := make(map[string]map[string]bool) // funnelID -> colID -> visible
	for _, p := range profiles {
		m := make(map[string]bool)
		for _, cid := range p.VisibleColumns { m[cid] = true }
		pset[p.FunnelID] = m
	}

	tenantID := middleware.GetTenantID(ctx)
	allFunnels, _ := h.listFunnelsUC.Execute(ctx, tenantID)
	nameByID := make(map[string]string, len(allFunnels))
	for _, f := range allFunnels { nameByID[f.ID] = f.Name }

	result := make([]funnelWithCols, 0, len(assignedFunnels))
	for _, gf := range assignedFunnels {
		cols, _ := h.columnRepo.FindByFunnelID(ctx, gf.FunnelID)
		colOpts := make([]columnOpt, len(cols))
		for i, col := range cols {
			visible := true
			if m, ok := pset[gf.FunnelID]; ok { visible = m[col.ID] }
			colOpts[i] = columnOpt{ID: col.ID, Name: col.Name, Visible: visible}
		}
		result = append(result, funnelWithCols{ID: gf.FunnelID, Name: nameByID[gf.FunnelID], Columns: colOpts})
	}

	c.HTML(http.StatusOK, "team/group_section_view_profiles.html", gin.H{
		"GroupID": groupID, "Funnels": result,
	})
}

func (h *PageHandler) SetViewProfileHTML(c *gin.Context) {
	groupID := c.Param("id")
	funnelID := c.Param("fid")
	cols := c.PostFormArray("visible_columns")
	if err := h.manageVP.SetViewProfile(c.Request.Context(), application.ViewProfileInput{
		GroupID: groupID, FunnelID: funnelID, VisibleColumns: cols,
	}); err != nil {
		c.Status(http.StatusUnprocessableEntity); return
	}
	c.Status(http.StatusOK)
}
```

Register:

```go
team.POST("/groups/:id/view-profiles/:fid", requirePerm("funnels", "customize"), h.page.SetViewProfileHTML)
```

- [ ] **Step 3: Commit**

```bash
git add web/templates/team/group_section_view_profiles.html internal/permission/interfaces/http/page_handler.go internal/permission/interfaces/http/routes.go
git commit -m "feat(F08): group detail — view profiles section"
```

---

### Task 18: Section — Load balance

**Files:**
- Create: `web/templates/team/group_section_load_balance.html`
- Modify: `internal/permission/interfaces/http/page_handler.go`

- [ ] **Step 1: Template**

```html
{{define "team/group_section_load_balance.html"}}
<form hx-post="/tenant/team/groups/{{.GroupID}}/load-balance"
      hx-target="closest section"
      hx-get="/tenant/team/groups/{{.GroupID}}/section/load-balance">
    <fieldset>
        <legend>Algoritmo</legend>
        <label><input type="radio" name="algorithm" value="round_robin" {{if eq .Algorithm "round_robin"}}checked{{end}}> Round-robin — distribui em ordem circular</label><br>
        <label><input type="radio" name="algorithm" value="least_load" {{if eq .Algorithm "least_load"}}checked{{end}}> Menor carga — atribui para quem tem menos leads</label><br>
        <label><input type="radio" name="algorithm" value="random" {{if eq .Algorithm "random"}}checked{{end}}> Aleatório — sorteia</label>
    </fieldset>
    <label><input type="checkbox" name="active" value="true" {{if .Active}}checked{{end}}> Balanceamento ativo</label>
    <p class="muted">Quando ativo, novos leads serão atribuídos automaticamente aos membros deste grupo.</p>
    <div class="section-actions">
        <button class="btn btn-primary" type="submit">Salvar</button>
    </div>
</form>
{{end}}
```

- [ ] **Step 2: Handler + save endpoint**

```go
func (h *PageHandler) sectionLoadBalance(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")

	cfg, err := h.loadBalanceUC.GetByGroup(c.Request.Context(), tenantID, groupID)
	algo := string(authdomain.AlgorithmRoundRobin)
	active := false
	if err == nil && cfg != nil {
		algo = string(cfg.Algorithm)
		active = cfg.Active
	}
	c.HTML(http.StatusOK, "team/group_section_load_balance.html", gin.H{
		"GroupID": groupID, "Algorithm": algo, "Active": active,
	})
}

func (h *PageHandler) SetLoadBalanceHTML(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")
	algorithm := c.PostForm("algorithm")
	active := c.PostForm("active") == "true"

	if _, err := h.loadBalanceUC.SetByGroup(c.Request.Context(), authapp.SetLoadBalanceInput{
		TenantID: tenantID, GroupID: groupID,
		Algorithm: authdomain.LoadBalanceAlgorithm(algorithm),
		Active:    active,
	}); err != nil {
		c.Status(http.StatusUnprocessableEntity); return
	}
	c.Status(http.StatusOK)
}
```

Register:

```go
team.POST("/groups/:id/load-balance", requirePerm("groups", "manage"), h.page.SetLoadBalanceHTML)
```

- [ ] **Step 3: Commit**

```bash
git add web/templates/team/group_section_load_balance.html internal/permission/interfaces/http/page_handler.go internal/permission/interfaces/http/routes.go
git commit -m "feat(F08): group detail — load balance section"
```

---

### Task 19: Permission PageHandler tests

**Files:**
- Create: `internal/permission/interfaces/http/page_handler_test.go`

- [ ] **Step 1: Tests for rendering and 401/403**

Use the same pattern as `handler_test.go`. Cover:
- `GroupsPage` 200 (with seeded groups)
- `GroupsTable` 200
- `GroupDetail` 404 when group absent
- `GroupSection` 404 for unknown name
- `sectionLoadBalance` renders default algorithm "round_robin" when no config
- `SetLoadBalanceHTML` 422 when algorithm invalid (stub use case returning the error)

Coverage target >= 80%.

- [ ] **Step 2: Run tests**

```bash
go test ./internal/permission/interfaces/http/... -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/permission/interfaces/http/page_handler_test.go
git commit -m "test(F08): permission PageHandler unit tests"
```

---

## Phase 5 — OWASP & finishing

### Task 20: OWASP tests for new endpoints

**Files:**
- Modify: `internal/permission/interfaces/http/page_handler_test.go`
- Modify: `internal/auth/interfaces/http/page_handler_test.go`

- [ ] **Step 1: Add per-endpoint 401/403 and tenant-isolation tests**

For each new HTML + API route:
- No cookie → 401
- Cookie with missing permission → 403
- Cookie with a different tenant attempting to access group `:id` of this tenant → 404 (does not exist in its own tenant scope)

Follow the existing owasp_test style in `internal/product/interfaces/http/owasp_test.go`.

- [ ] **Step 2: Run**

```bash
go test ./... -run OWASP -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/permission/interfaces/http/page_handler_test.go internal/auth/interfaces/http/page_handler_test.go
git commit -m "test(F08): OWASP tests for Equipe endpoints"
```

---

### Task 21: rest/team.http + docs update

**Files:**
- Create: `rest/team.http`
- Modify: `docs/features/F08-usuarios-permissoes.md` — check Step 6 items
- Modify: `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/status.md`

- [ ] **Step 1: `rest/team.http` — sample requests**

```http
### Variables
@baseUrl = http://localhost:8080
@token = YOUR_JWT_HERE

### List groups
GET {{baseUrl}}/tenant/groups
Cookie: token={{token}}

### Create group
POST {{baseUrl}}/tenant/groups
Cookie: token={{token}}
Content-Type: application/json

{ "name": "Vendas", "description": "Time de vendas" }

### Get load balance
GET {{baseUrl}}/tenant/groups/GROUP_ID/load-balance
Cookie: token={{token}}

### Set load balance
PUT {{baseUrl}}/tenant/groups/GROUP_ID/load-balance
Cookie: token={{token}}
Content-Type: application/json

{ "algorithm": "round_robin", "active": true }

### Users tab (HTML)
GET {{baseUrl}}/tenant/team/users
Cookie: token={{token}}

### Create invite
POST {{baseUrl}}/tenant/invites
Cookie: token={{token}}
Content-Type: application/json

{ "group_ids": ["GROUP_ID"], "expires_in_days": 7 }
```

- [ ] **Step 2: Mark F08 Step 6 items as done in the feature doc**

Check off the 4 template items in `docs/features/F08-usuarios-permissoes.md` Step 6 and the interactions line.

- [ ] **Step 3: Update status.md**

Mark in `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/status.md`:
- [x] Load balance integration (backend complete)
- [x] Telas HTMX para gestão de grupos e permissões
- [x] Telas HTMX para convites e gestão de usuários

- [ ] **Step 4: Commit**

```bash
git add rest/team.http docs/features/F08-usuarios-permissoes.md docs/artefatos/F08-F09-usuarios-permissoes-automacoes/status.md
git commit -m "docs(F08): update backlog + status, add rest/team.http"
```

---

### Task 22: Final build + coverage check

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -cover
```

Expected: all pass, coverage >= 80% in touched packages.

- [ ] **Step 2: Run app in dev + smoke all 5 sections**

```bash
docker compose up -d
go run ./cmd/api
```

Manually:
1. `/tenant/team` → redirect to `/tenant/team/users`
2. Create an invite, copy link, open in new session, complete onboarding
3. `/tenant/team/groups` → create "Vendas"
4. In the group detail, exercise each of the 5 sections:
   - add a member
   - toggle permissions in the matrix
   - assign a funnel
   - toggle column visibility in the view profile
   - set algorithm + active = true and save

- [ ] **Step 3: Commit nothing (verification only)**

If something is off, open a follow-up task and fix; otherwise this plan is done.

---

## Spec coverage checklist

| Spec section | Where implemented |
|--------------|-------------------|
| Sidebar "Equipe" | Task 5 |
| Shell + 2 tabs | Task 5 |
| Users tab (list + perms modal + WhatsApp modal + remove) | Tasks 6–7, 9, 10 |
| Invites pending + new modal | Tasks 7, 8 |
| Groups tab (list + new modal) | Task 12 |
| Group detail header | Task 13 |
| Members section | Task 14 |
| Permissions matrix | Task 15 |
| Funnels section | Task 16 |
| View profiles section | Task 17 |
| Load balance section + backend | Tasks 1–4, 18 |
| OWASP tests | Task 20 |
| rest/.http | Task 21 |
| Observabilidade (log.Error com campos estruturados) | Tasks 1–18 handlers |
| Cobertura >= 80% | Tasks 2, 11, 19, 20, 22 |
