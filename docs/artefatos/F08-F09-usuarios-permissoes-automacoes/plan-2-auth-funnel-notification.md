# Plan 2: Auth Extensions + Funnel Extensions + Notification Module

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend auth (invites, load balance, owner), extend funnel (responsible_user_id, events), and build notification module (SSE + polling + WhatsApp optional).

**Architecture:** Auth module gets InviteToken, LoadBalanceConfig, and IsOwner on UserTenant. Funnel module gains ResponsibleUserID on Lead and publishes lead-created/lead-moved events via shared EventBus. Notification module is a new DDD module that persists notifications, delivers via SSE with polling fallback, and optionally sends via WhatsApp.

**Tech Stack:** Go, Gin, GORM, MySQL, golang-migrate, testify, SSE, shared EventBus

**Spec:** `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/design-v1.md`

**Depends on:** Plan 1 (EventBus shared + Permission module)

---

## File Structure

### Auth Extensions
```
internal/auth/
  domain/
    user.go              # MODIFY: add UserTenant fields (IsOwner, WhatsAppID)
    repository.go        # MODIFY: add new repo methods + InviteTokenRepository
    invite_token.go      # NEW: InviteToken entity
    load_balance.go      # NEW: LoadBalanceConfig entity
  application/
    invite_user.go       # NEW: generate + accept invite
    manage_users.go      # NEW: list/remove tenant users
    load_balance.go      # NEW: configure + execute load balance
  infrastructure/
    gorm_user_tenant_repo.go  # MODIFY: add IsOwner, WhatsAppID, new methods
    gorm_invite_token_repo.go # NEW
    gorm_load_balance_repo.go # NEW
    models.go                 # NEW: consolidated auth models
  interfaces/http/
    handler.go           # MODIFY: add invite + user management handlers
    routes.go            # NEW: separate route registration
```

### Funnel Extensions
```
internal/funnel/
  domain/
    lead.go              # MODIFY: add ResponsibleUserID field
  infrastructure/
    models.go            # MODIFY: add ResponsibleUserID to leadModel
  application/
    create_lead.go       # MODIFY: publish lead-created event
    move_lead.go         # MODIFY: publish lead-moved event
    assign_lead.go       # NEW: manual assignment
  interfaces/http/
    handler.go           # MODIFY: add AssignLead handler
    routes.go            # MODIFY: add PUT /leads/:id/assign route
  module.go              # MODIFY: accept EventBus, expose new UCs
```

### Notification Module
```
internal/notification/
  domain/
    notification.go      # Notification entity
    preference.go        # NotificationPreference entity
    repository.go        # Repository interfaces
    errors.go            # Sentinel errors
  application/
    notify.go            # Core notify service
    list_notifications.go
    mark_read.go
    manage_preferences.go
    mocks_test.go
    notify_test.go
  infrastructure/
    models.go
    gorm_notification_repo.go
    gorm_preference_repo.go
  interfaces/http/
    handler.go           # SSE + REST handlers
    routes.go
  module.go
```

### Migrations
```
migrations/
  000040_add_is_owner_whatsappid_to_user_tenants.up.sql
  000040_add_is_owner_whatsappid_to_user_tenants.down.sql
  000041_create_invite_tokens.up.sql
  000041_create_invite_tokens.down.sql
  000042_create_load_balance_configs.up.sql
  000042_create_load_balance_configs.down.sql
  000043_add_responsible_user_id_to_leads.up.sql
  000043_add_responsible_user_id_to_leads.down.sql
  000044_create_notifications.up.sql
  000044_create_notifications.down.sql
  000045_create_notification_preferences.up.sql
  000045_create_notification_preferences.down.sql
```

---

## Task 1: Migrations

**Files:** 6 up + 6 down migration files in `migrations/`

- [ ] **Step 1: Create all migration files**

```sql
-- 000040_add_is_owner_whatsappid_to_user_tenants.up.sql
ALTER TABLE user_tenants ADD COLUMN is_owner TINYINT(1) NOT NULL DEFAULT 0;
ALTER TABLE user_tenants ADD COLUMN whatsapp_id VARCHAR(100) NULL;

-- 000040_add_is_owner_whatsappid_to_user_tenants.down.sql
ALTER TABLE user_tenants DROP COLUMN whatsapp_id;
ALTER TABLE user_tenants DROP COLUMN is_owner;

-- 000041_create_invite_tokens.up.sql
CREATE TABLE invite_tokens (
    id CHAR(36) NOT NULL PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    token VARCHAR(64) NOT NULL,
    created_by CHAR(36) NOT NULL,
    group_ids JSON NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at DATETIME NULL,
    used_by CHAR(36) NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id),
    UNIQUE KEY uk_invite_token (token),
    INDEX idx_invite_tokens_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 000041_create_invite_tokens.down.sql
DROP TABLE IF EXISTS invite_tokens;

-- 000042_create_load_balance_configs.up.sql
CREATE TABLE load_balance_configs (
    id CHAR(36) NOT NULL PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    group_id CHAR(36) NOT NULL,
    algorithm VARCHAR(20) NOT NULL DEFAULT 'round_robin',
    last_index INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES permission_groups(id) ON DELETE CASCADE,
    UNIQUE KEY uk_lb_tenant_group (tenant_id, group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 000042_create_load_balance_configs.down.sql
DROP TABLE IF EXISTS load_balance_configs;

-- 000043_add_responsible_user_id_to_leads.up.sql
ALTER TABLE leads ADD COLUMN responsible_user_id CHAR(36) NULL;
ALTER TABLE leads ADD INDEX idx_leads_responsible (responsible_user_id);

-- 000043_add_responsible_user_id_to_leads.down.sql
ALTER TABLE leads DROP INDEX idx_leads_responsible;
ALTER TABLE leads DROP COLUMN responsible_user_id;

-- 000044_create_notifications.up.sql
CREATE TABLE notifications (
    id CHAR(36) NOT NULL PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    body VARCHAR(1000) NOT NULL DEFAULT '',
    metadata JSON NOT NULL,
    is_read TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_notifications_user_unread (user_id, is_read, created_at DESC),
    INDEX idx_notifications_tenant_user (tenant_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 000044_create_notifications.down.sql
DROP TABLE IF EXISTS notifications;

-- 000045_create_notification_preferences.up.sql
CREATE TABLE notification_preferences (
    id CHAR(36) NOT NULL PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    tenant_id CHAR(36) NOT NULL,
    channel VARCHAR(20) NOT NULL,
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE KEY uk_notif_pref (user_id, tenant_id, channel)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 000045_create_notification_preferences.down.sql
DROP TABLE IF EXISTS notification_preferences;
```

- [ ] **Step 2: Commit**

```bash
git add migrations/000040_* migrations/000041_* migrations/000042_* migrations/000043_* migrations/000044_* migrations/000045_*
git commit -m "feat(F08-F09): add migrations for auth extensions, lead responsible, notifications"
```

---

## Task 2: Auth Domain Extensions

**Files:**
- Modify: `internal/auth/domain/user.go`
- Create: `internal/auth/domain/invite_token.go`
- Create: `internal/auth/domain/load_balance.go`
- Modify: `internal/auth/domain/repository.go`

- [ ] **Step 1: Extend UserTenant with IsOwner and WhatsAppID**

In `internal/auth/domain/user.go`, update `UserTenant`:
```go
type UserTenant struct {
	UserID     string
	TenantID   string
	IsOwner    bool
	WhatsAppID string
}
```

Add errors:
```go
var (
	// ... existing errors ...
	ErrInviteTokenNotFound = errors.New("invite token not found")
	ErrInviteTokenExpired  = errors.New("invite token has expired")
	ErrInviteTokenUsed     = errors.New("invite token has already been used")
)
```

- [ ] **Step 2: Create InviteToken entity**

```go
// internal/auth/domain/invite_token.go
package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type InviteToken struct {
	ID        string
	TenantID  string
	Token     string
	CreatedBy string
	GroupIDs  []string
	ExpiresAt time.Time
	UsedAt    *time.Time
	UsedBy    string
	CreatedAt time.Time
}

func NewInviteToken(id, tenantID, createdBy string, groupIDs []string, expiresAt time.Time) (*InviteToken, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if createdBy == "" {
		return nil, ErrUserIDRequired
	}
	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	return &InviteToken{
		ID: id, TenantID: tenantID, Token: token, CreatedBy: createdBy,
		GroupIDs: groupIDs, ExpiresAt: expiresAt, CreatedAt: time.Now(),
	}, nil
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (t *InviteToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

func (t *InviteToken) IsUsed() bool {
	return t.UsedAt != nil
}

func (t *InviteToken) MarkUsed(userID string) {
	now := time.Now()
	t.UsedAt = &now
	t.UsedBy = userID
}

// Validate checks if the token can still be used.
func (t *InviteToken) Validate() error {
	if t.IsUsed() {
		return ErrInviteTokenUsed
	}
	if t.IsExpired() {
		return ErrInviteTokenExpired
	}
	return nil
}
```

- [ ] **Step 3: Create LoadBalanceConfig entity**

```go
// internal/auth/domain/load_balance.go
package domain

import (
	"errors"
	"time"
)

type LoadBalanceAlgorithm string

const (
	AlgorithmRoundRobin LoadBalanceAlgorithm = "round_robin"
	AlgorithmLeastLoad  LoadBalanceAlgorithm = "least_load"
	AlgorithmRandom     LoadBalanceAlgorithm = "random"
)

var ErrInvalidAlgorithm = errors.New("invalid load balance algorithm")
var ErrLoadBalanceNotFound = errors.New("load balance config not found")

type LoadBalanceConfig struct {
	ID        string
	TenantID  string
	GroupID   string
	Algorithm LoadBalanceAlgorithm
	LastIndex int
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewLoadBalanceConfig(id, tenantID, groupID string, algorithm LoadBalanceAlgorithm) (*LoadBalanceConfig, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if groupID == "" {
		return nil, ErrGroupIDRequired
	}
	if !isValidAlgorithm(algorithm) {
		return nil, ErrInvalidAlgorithm
	}
	now := time.Now()
	return &LoadBalanceConfig{
		ID: id, TenantID: tenantID, GroupID: groupID,
		Algorithm: algorithm, LastIndex: 0,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func isValidAlgorithm(a LoadBalanceAlgorithm) bool {
	return a == AlgorithmRoundRobin || a == AlgorithmLeastLoad || a == AlgorithmRandom
}

func (c *LoadBalanceConfig) IncrementIndex() {
	c.LastIndex++
	c.UpdatedAt = time.Now()
}
```

Add `ErrTenantIDRequired`, `ErrUserIDRequired`, `ErrGroupIDRequired` if they don't already exist in user.go.

- [ ] **Step 4: Extend repository interfaces**

In `internal/auth/domain/repository.go`, add:
```go
type UserTenantRepository interface {
	Associate(ctx context.Context, userID, tenantID string) error
	FindTenantIDsByUserID(ctx context.Context, userID string) ([]string, error)
	// New methods:
	FindByTenantID(ctx context.Context, tenantID string) ([]UserTenant, error)
	FindByUserAndTenant(ctx context.Context, userID, tenantID string) (*UserTenant, error)
	UpdateIsOwner(ctx context.Context, userID, tenantID string, isOwner bool) error
	UpdateWhatsAppID(ctx context.Context, userID, tenantID string, whatsAppID string) error
	RemoveFromTenant(ctx context.Context, userID, tenantID string) error
	IsOwner(ctx context.Context, userID, tenantID string) (bool, error)
}

type InviteTokenRepository interface {
	Create(ctx context.Context, token *InviteToken) error
	FindByToken(ctx context.Context, token string) (*InviteToken, error)
	FindByTenantID(ctx context.Context, tenantID string) ([]InviteToken, error)
	Update(ctx context.Context, token *InviteToken) error
	Delete(ctx context.Context, id string) error
}

type LoadBalanceConfigRepository interface {
	CreateOrUpdate(ctx context.Context, config *LoadBalanceConfig) error
	FindByGroupID(ctx context.Context, tenantID, groupID string) (*LoadBalanceConfig, error)
	FindByTenantID(ctx context.Context, tenantID string) ([]LoadBalanceConfig, error)
	Update(ctx context.Context, config *LoadBalanceConfig) error
}
```

- [ ] **Step 5: Verify build**

Run: `go build ./internal/auth/...`

- [ ] **Step 6: Commit**

```bash
git add internal/auth/domain/
git commit -m "feat(auth): extend domain with InviteToken, LoadBalanceConfig, UserTenant fields"
```

---

## Task 3: Auth Infrastructure — Repositories

**Files:**
- Modify: `internal/auth/infrastructure/gorm_user_tenant_repo.go`
- Create: `internal/auth/infrastructure/gorm_invite_token_repo.go`
- Create: `internal/auth/infrastructure/gorm_load_balance_repo.go`

Implement all repository interfaces from Task 2. Follow existing GORM patterns. Use `*string` for nullable fields, `encoding/json` for JSON columns (GroupIDs).

- [ ] **Step 1: Extend userTenantModel and repository with new fields/methods**
- [ ] **Step 2: Create InviteToken GORM model + repository**
- [ ] **Step 3: Create LoadBalanceConfig GORM model + repository**
- [ ] **Step 4: Verify build**: `go build ./internal/auth/...`
- [ ] **Step 5: Commit**

---

## Task 4: Auth Use Cases — Invites + User Management

**Files:**
- Create: `internal/auth/application/invite_user.go`
- Create: `internal/auth/application/manage_users.go`

- [ ] **Step 1: Create InviteUserUseCase**

```go
// GenerateInvite(ctx, tenantID, createdBy, groupIDs, expiresAt) → InviteOutput{ID, Token, Link}
// AcceptInvite(ctx, token, name, email, password) → user created/associated + groups assigned
// ListInvites(ctx, tenantID) → []InviteOutput
// RevokeInvite(ctx, id) → delete
```

AcceptInvite flow:
1. Find token by value, validate (not expired, not used)
2. Check if user exists by email
3. If new: create user with hashed password + associate to tenant
4. If existing: just associate to tenant
5. For each groupID in token.GroupIDs, add user to group (via permission UserGroupRepository)
6. Mark token as used

- [ ] **Step 2: Create ManageUsersUseCase**

```go
// ListTenantUsers(ctx, tenantID) → []UserOutput{ID, Name, Email, IsOwner}
// RemoveFromTenant(ctx, userID, tenantID) → remove association
// SetWhatsAppID(ctx, userID, tenantID, whatsAppID) → update
```

- [ ] **Step 3: Write tests for invite flow (generate, accept, validate expired/used)**
- [ ] **Step 4: Verify**: `go test ./internal/auth/application/ -v`
- [ ] **Step 5: Commit**

---

## Task 5: Funnel Extensions — ResponsibleUserID + Events

**Files:**
- Modify: `internal/funnel/domain/lead.go` — add `ResponsibleUserID string`
- Modify: `internal/funnel/infrastructure/models.go` — add field to leadModel
- Modify: `internal/funnel/application/create_lead.go` — publish `lead-created` event
- Modify: `internal/funnel/application/move_lead.go` — publish `lead-moved` event
- Create: `internal/funnel/application/assign_lead.go` — manual assignment
- Modify: `internal/funnel/module.go` — accept EventBus parameter

- [ ] **Step 1: Add ResponsibleUserID to Lead entity and model**

In `lead.go`:
```go
type Lead struct {
	// ... existing fields ...
	ResponsibleUserID string
}

func (l *Lead) AssignResponsible(userID string) {
	l.ResponsibleUserID = userID
	l.UpdatedAt = time.Now()
}
```

In `models.go`, add to leadModel:
```go
ResponsibleUserID string `gorm:"column:responsible_user_id;type:char(36)"`
```
Update mappers.

- [ ] **Step 2: Publish events in create_lead.go and move_lead.go**

Add EventBus as dependency to both use cases. After creating/moving lead, publish:
```go
uc.eventBus.Publish(events.Event{
	Type: events.EventLeadCreated,
	TenantID: lead.TenantID,
	Payload: map[string]string{
		"lead_id": lead.ID, "funnel_id": lead.FunnelID,
		"column_id": lead.ColumnID, "contact_id": lead.ContactID,
	},
})
```

- [ ] **Step 3: Create AssignLeadUseCase**

```go
func (uc *AssignLeadUseCase) Execute(ctx context.Context, input AssignLeadInput) error {
	lead, err := uc.leadRepo.FindByID(ctx, input.LeadID)
	// verify tenant
	lead.AssignResponsible(input.UserID)
	return uc.leadRepo.Update(ctx, lead)
}
```

- [ ] **Step 4: Add handler + route for PUT /leads/:id/assign**
- [ ] **Step 5: Update funnel module.go to accept/expose EventBus**
- [ ] **Step 6: Update main.go to pass EventBus to funnel module**
- [ ] **Step 7: Run tests**: `go test ./internal/funnel/... -v`
- [ ] **Step 8: Commit**

---

## Task 6: Notification Domain

**Files:**
- Create: `internal/notification/domain/notification.go`
- Create: `internal/notification/domain/preference.go`
- Create: `internal/notification/domain/errors.go`
- Create: `internal/notification/domain/repository.go`

- [ ] **Step 1: Create Notification entity**

```go
type NotificationType string

const (
	TypeLeadAssigned     NotificationType = "lead_assigned"
	TypeLeadMoved        NotificationType = "lead_moved"
	TypeLeadHandoff      NotificationType = "lead_handoff"
	TypeLeadQualified    NotificationType = "lead_qualified"
	TypeRateLimitReached NotificationType = "rate_limit_reached"
	TypeAutomationError  NotificationType = "automation_error"
)

type Notification struct {
	ID        string
	TenantID  string
	UserID    string
	Type      NotificationType
	Title     string
	Body      string
	Metadata  map[string]string
	Read      bool
	CreatedAt time.Time
}
```

- [ ] **Step 2: Create NotificationPreference entity**

```go
type Channel string
const (
	ChannelInApp    Channel = "in_app"
	ChannelWhatsApp Channel = "whatsapp"
)

type NotificationPreference struct {
	ID        string
	UserID    string
	TenantID  string
	Channel   Channel
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

- [ ] **Step 3: Create errors and repository interfaces**
- [ ] **Step 4: Verify build**: `go build ./internal/notification/...`
- [ ] **Step 5: Commit**

---

## Task 7: Notification Infrastructure — Repositories

**Files:**
- Create: `internal/notification/infrastructure/models.go`
- Create: `internal/notification/infrastructure/gorm_notification_repo.go`
- Create: `internal/notification/infrastructure/gorm_preference_repo.go`

- [ ] **Step 1: Create GORM models and repos**

NotificationRepository methods: Create, FindByUserID(tenantID, userID, onlyUnread, limit, offset), CountUnread(tenantID, userID), MarkRead(id), MarkAllRead(tenantID, userID)

PreferenceRepository methods: CreateOrUpdate, FindByUser(userID, tenantID), FindByUserAndChannel(userID, tenantID, channel)

- [ ] **Step 2: Verify build**
- [ ] **Step 3: Commit**

---

## Task 8: Notification Use Cases

**Files:**
- Create: `internal/notification/application/notify.go`
- Create: `internal/notification/application/notify_test.go`
- Create: `internal/notification/application/list_notifications.go`
- Create: `internal/notification/application/mark_read.go`
- Create: `internal/notification/application/manage_preferences.go`
- Create: `internal/notification/application/mocks_test.go`

- [ ] **Step 1: Create NotifyService**

```go
type NotifyService struct {
	notifRepo domain.NotificationRepository
	prefRepo  domain.PreferenceRepository
	eventBus  events.EventBus
}

func (s *NotifyService) Notify(ctx context.Context, userID, tenantID string, notifType domain.NotificationType, title, body string, metadata map[string]string) error {
	notif := domain.NewNotification(uuid.New().String(), tenantID, userID, notifType, title, body, metadata)
	if err := s.notifRepo.Create(ctx, notif); err != nil {
		return err
	}
	// Check in_app preference (default on)
	s.eventBus.Publish(events.Event{
		Type: events.EventNotification, TenantID: tenantID,
		Payload: notif,
	})
	return nil
}
```

- [ ] **Step 2: Create list/mark-read/preferences use cases**
- [ ] **Step 3: Write tests for NotifyService**
- [ ] **Step 4: Verify**: `go test ./internal/notification/application/ -v`
- [ ] **Step 5: Commit**

---

## Task 9: Notification HTTP Handlers + SSE

**Files:**
- Create: `internal/notification/interfaces/http/handler.go`
- Create: `internal/notification/interfaces/http/routes.go`

- [ ] **Step 1: Create handler with SSE stream**

```go
func (h *Handler) StreamNotifications(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ch, cleanup := h.eventBus.Subscribe(tenantID)
	defer cleanup()

	c.Stream(func(w io.Writer) bool {
		select {
		case event := <-ch:
			if event.Type == events.EventNotification {
				notif, ok := event.Payload.(*domain.Notification)
				if ok && notif.UserID == claims.UserID {
					data, _ := json.Marshal(notif)
					c.SSEvent("notification", string(data))
				}
			}
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
```

- [ ] **Step 2: Create REST handlers**

List, UnreadCount, MarkRead, MarkAllRead, GetPreferences, UpdatePreferences.

- [ ] **Step 3: Create routes**

```go
notif := router.Group("/notifications")
notif.Use(authMw, tenantMw)
{
	notif.GET("/stream", h.StreamNotifications)
	notif.GET("", h.ListNotifications)
	notif.GET("/unread-count", h.UnreadCount)
	notif.POST("/:id/read", h.MarkRead)
	notif.POST("/read-all", h.MarkAllRead)
	notif.GET("/preferences", h.GetPreferences)
	notif.PUT("/preferences", h.UpdatePreferences)
}
```

- [ ] **Step 4: Verify build**
- [ ] **Step 5: Commit**

---

## Task 10: Module Wiring + OwnerChecker Update

**Files:**
- Create: `internal/notification/module.go`
- Modify: `internal/permission/infrastructure/adapters.go` — update OwnerCheckerAdapter
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Create notification module**

Wire all repos, use cases, handler. Expose NotifyService for cross-module use.

- [ ] **Step 2: Update OwnerCheckerAdapter**

Now that `user_tenants.is_owner` exists, query it:
```go
func (a *OwnerCheckerAdapter) IsOwner(ctx context.Context, userID, tenantID string) (bool, error) {
	var count int64
	err := a.db.WithContext(ctx).Table("user_tenants").
		Where("user_id = ? AND tenant_id = ? AND is_owner = ?", userID, tenantID, true).
		Count(&count).Error
	return count > 0, err
}
```

- [ ] **Step 3: Update main.go**

- Create notification module
- Pass EventBus to funnel module
- Wire auth extensions (invite, user management, load balance handlers)
- Add notification module to modules slice

- [ ] **Step 4: Verify full build**: `go build ./...`
- [ ] **Step 5: Run all tests**: `go test ./... -count=1`
- [ ] **Step 6: Commit**

---

## Task 11: Load Balance Integration

**Files:**
- Modify: `internal/funnel/application/create_lead.go` — call load balance after creation

- [ ] **Step 1: Add LoadBalancer interface to funnel domain**

```go
// internal/funnel/domain/load_balancer.go
type LoadBalancer interface {
	AssignResponsible(ctx context.Context, tenantID, funnelID, columnID string) (string, error)
}
```

- [ ] **Step 2: Integrate in CreateLeadUseCase**

After creating lead, call LoadBalancer.AssignResponsible. If a responsible user is returned, set it on the lead and notify via NotificationService.

- [ ] **Step 3: Implement LoadBalancer in auth module**

Uses GroupFunnelRepository to find group for column, LoadBalanceConfigRepository to get algorithm, UserGroupRepository for members, then applies algorithm (round_robin/least_load/random).

- [ ] **Step 4: Wire via adapter in funnel module**
- [ ] **Step 5: Write test for load balance assignment**
- [ ] **Step 6: Commit**

---

## Task 12: Integration Tests + Verification

- [ ] **Step 1: Run full test suite**: `go test ./... -count=1`
- [ ] **Step 2: Check coverage**: `go test ./internal/notification/application/ -cover` (>= 80%)
- [ ] **Step 3: Check coverage**: `go test ./internal/auth/application/ -cover` (>= 80%)
- [ ] **Step 4: Verify build**: `go build ./...`
- [ ] **Step 5: Final commit if any cleanup**
