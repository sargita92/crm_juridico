# Plan 1: EventBus Shared + Permission Module

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote EventBus to shared infrastructure and build the complete permission module (groups, permissions, view profiles, middleware) as foundation for F08+F09.

**Architecture:** The EventBus moves from `internal/whatsapp/` to `internal/shared/events/` so all modules can publish/subscribe. The permission module follows DDD + Clean Architecture with domain entities, GORM repositories, use cases, middleware, and HTTP handlers. Permission resolution uses union semantics (any grant from group or individual = allowed).

**Tech Stack:** Go, Gin, GORM, MySQL, golang-migrate, testify, TDD

**Spec:** `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/design-v1.md`

---

## File Structure

### EventBus Promotion

```
internal/shared/events/
  event.go              # EventBus interface, Event struct, EventType constants
  memory_bus.go         # In-memory implementation (moved from whatsapp)
  memory_bus_test.go    # Tests (moved from whatsapp)
```

### Permission Module

```
internal/permission/
  domain/
    permission_group.go   # PermissionGroup entity
    user_group.go         # UserGroup entity
    permission.go         # Permission entity (group + individual)
    view_profile.go       # ViewProfile entity
    group_funnel.go       # GroupFunnel entity
    repository.go         # Repository interfaces
    errors.go             # Sentinel errors
  application/
    resolve_permission.go       # HasPermission resolver
    resolve_permission_test.go  # Tests for resolver
    create_group.go             # Create group use case
    create_group_test.go
    update_group.go             # Update group use case
    list_groups.go              # List groups use case
    delete_group.go             # Delete group use case
    manage_members.go           # Add/remove group members
    manage_members_test.go
    manage_permissions.go       # Set group/user permissions
    manage_permissions_test.go
    manage_view_profiles.go     # Configure view profiles
    manage_group_funnels.go     # Configure group-funnel associations
    mocks_test.go               # Shared mock implementations
  infrastructure/
    models.go                          # GORM models + mappers
    gorm_permission_group_repository.go
    gorm_user_group_repository.go
    gorm_permission_repository.go
    gorm_view_profile_repository.go
    gorm_group_funnel_repository.go
  interfaces/http/
    handler.go   # HTTP handlers
    routes.go    # Route registration
  module.go      # Module wiring
```

### Migrations

```
migrations/
  000035_create_permission_groups.up.sql
  000035_create_permission_groups.down.sql
  000036_create_user_groups.up.sql
  000036_create_user_groups.down.sql
  000037_create_permissions.up.sql
  000037_create_permissions.down.sql
  000038_create_view_profiles.up.sql
  000038_create_view_profiles.down.sql
  000039_create_group_funnels.up.sql
  000039_create_group_funnels.down.sql
```

### Modified Files

```
internal/shared/module/module.go                      # Add RequirePermission to Middlewares
internal/whatsapp/domain/events.go                    # Delete (moved to shared)
internal/whatsapp/infrastructure/memory_event_bus.go  # Delete (moved to shared)
internal/whatsapp/infrastructure/memory_event_bus_test.go  # Delete (moved to shared)
internal/whatsapp/module.go                           # Accept EventBus as parameter
internal/whatsapp/interfaces/http/handler.go          # Import from shared/events
internal/whatsapp/interfaces/http/handler_test.go     # Import from shared/events
internal/whatsapp/application/receive_message.go      # Import from shared/events
internal/whatsapp/application/send_message.go         # Import from shared/events
cmd/api/main.go                                       # Wire EventBus + permission module
```

---

## Task 1: Promote EventBus Interface to Shared

**Files:**
- Create: `internal/shared/events/event.go`

- [ ] **Step 1: Create the shared events package with interface and types**

```go
// internal/shared/events/event.go
package events

type EventType string

const (
	EventNewMessage         EventType = "new-message"
	EventConversationUpdate EventType = "conversation-update"
	EventLeadCreated        EventType = "lead-created"
	EventLeadMoved          EventType = "lead-moved"
	EventNotification       EventType = "notification"
)

type Event struct {
	Type     EventType
	TenantID string
	Payload  interface{}
}

// EventBus distributes events to subscribers scoped by tenant.
type EventBus interface {
	Publish(event Event)
	Subscribe(tenantID string) (<-chan Event, func())
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./internal/shared/events/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/shared/events/event.go
git commit -m "refactor: promote EventBus interface to shared/events"
```

---

## Task 2: Move MemoryEventBus Implementation to Shared

**Files:**
- Create: `internal/shared/events/memory_bus.go`
- Create: `internal/shared/events/memory_bus_test.go`

- [ ] **Step 1: Create memory bus implementation**

```go
// internal/shared/events/memory_bus.go
package events

import "sync"

const eventBufferSize = 100

type MemoryEventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event
}

func NewMemoryEventBus() *MemoryEventBus {
	return &MemoryEventBus{
		subscribers: make(map[string][]chan Event),
	}
}

func (b *MemoryEventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	channels, ok := b.subscribers[event.TenantID]
	if !ok {
		return
	}

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
			// Channel full, discard event to avoid blocking
		}
	}
}

func (b *MemoryEventBus) Subscribe(tenantID string) (<-chan Event, func()) {
	ch := make(chan Event, eventBufferSize)

	b.mu.Lock()
	b.subscribers[tenantID] = append(b.subscribers[tenantID], ch)
	b.mu.Unlock()

	cleanup := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		channels := b.subscribers[tenantID]
		for i, c := range channels {
			if c == ch {
				b.subscribers[tenantID] = append(channels[:i], channels[i+1:]...)
				break
			}
		}
		if len(b.subscribers[tenantID]) == 0 {
			delete(b.subscribers, tenantID)
		}
		close(ch)
	}

	return ch, cleanup
}
```

- [ ] **Step 2: Create tests (moved from whatsapp, updated imports)**

```go
// internal/shared/events/memory_bus_test.go
package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryEventBus_PublishSubscribe(t *testing.T) {
	bus := NewMemoryEventBus()
	ch, cleanup := bus.Subscribe("tenant-1")
	defer cleanup()

	event := Event{
		Type:     EventNewMessage,
		TenantID: "tenant-1",
		Payload:  map[string]string{"msg": "hello"},
	}
	bus.Publish(event)

	select {
	case received := <-ch:
		assert.Equal(t, EventNewMessage, received.Type)
		assert.Equal(t, "tenant-1", received.TenantID)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestMemoryEventBus_TenantIsolation(t *testing.T) {
	bus := NewMemoryEventBus()
	ch1, cleanup1 := bus.Subscribe("tenant-1")
	defer cleanup1()
	ch2, cleanup2 := bus.Subscribe("tenant-2")
	defer cleanup2()

	bus.Publish(Event{
		Type:     EventNewMessage,
		TenantID: "tenant-1",
		Payload:  nil,
	})

	select {
	case <-ch1:
		// expected
	case <-time.After(time.Second):
		t.Fatal("tenant-1 should receive event")
	}

	select {
	case <-ch2:
		t.Fatal("tenant-2 should NOT receive tenant-1 event")
	case <-time.After(100 * time.Millisecond):
		// expected
	}
}

func TestMemoryEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewMemoryEventBus()
	ch1, cleanup1 := bus.Subscribe("tenant-1")
	defer cleanup1()
	ch2, cleanup2 := bus.Subscribe("tenant-1")
	defer cleanup2()

	bus.Publish(Event{
		Type:     EventNewMessage,
		TenantID: "tenant-1",
		Payload:  nil,
	})

	for _, ch := range []<-chan Event{ch1, ch2} {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatal("both subscribers should receive event")
		}
	}
}

func TestMemoryEventBus_Cleanup(t *testing.T) {
	bus := NewMemoryEventBus()
	ch, cleanup := bus.Subscribe("tenant-1")
	cleanup()

	bus.Publish(Event{
		Type:     EventNewMessage,
		TenantID: "tenant-1",
		Payload:  nil,
	})

	select {
	case _, ok := <-ch:
		require.False(t, ok, "channel should be closed")
	case <-time.After(100 * time.Millisecond):
		// expected — channel closed, no event
	}
}

func TestMemoryEventBus_FullBuffer_NoBlock(t *testing.T) {
	bus := NewMemoryEventBus()
	_, cleanup := bus.Subscribe("tenant-1")
	defer cleanup()

	done := make(chan struct{})
	go func() {
		for i := 0; i < eventBufferSize+10; i++ {
			bus.Publish(Event{
				Type:     EventNewMessage,
				TenantID: "tenant-1",
				Payload:  nil,
			})
		}
		close(done)
	}()

	select {
	case <-done:
		// Publish should not block even when buffer is full
	case <-time.After(2 * time.Second):
		t.Fatal("Publish should not block on full buffer")
	}
}

func TestMemoryEventBus_ConcurrentAccess(t *testing.T) {
	bus := NewMemoryEventBus()
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, cleanup := bus.Subscribe("tenant-1")
			defer cleanup()
			<-ch
		}()
	}

	time.Sleep(50 * time.Millisecond)
	bus.Publish(Event{
		Type:     EventNewMessage,
		TenantID: "tenant-1",
		Payload:  nil,
	})

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent subscribers should all receive events")
	}
}
```

- [ ] **Step 3: Run tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/shared/events/ -v`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add internal/shared/events/memory_bus.go internal/shared/events/memory_bus_test.go
git commit -m "refactor: move MemoryEventBus implementation to shared/events"
```

---

## Task 3: Update WhatsApp Module to Use Shared EventBus

**Files:**
- Modify: `internal/whatsapp/module.go`
- Modify: `internal/whatsapp/application/receive_message.go`
- Modify: `internal/whatsapp/application/send_message.go`
- Modify: `internal/whatsapp/interfaces/http/handler.go`
- Modify: `internal/whatsapp/interfaces/http/handler_test.go`
- Delete: `internal/whatsapp/domain/events.go`
- Delete: `internal/whatsapp/infrastructure/memory_event_bus.go`
- Delete: `internal/whatsapp/infrastructure/memory_event_bus_test.go`

This task is a mechanical refactor: replace all `whatsapp/domain.Event*` and `whatsapp/domain.EventBus` imports with `shared/events.Event*` and `shared/events.EventBus`.

- [ ] **Step 1: Update whatsapp module to accept EventBus as parameter**

In `internal/whatsapp/module.go`, change `NewModule` to accept an `events.EventBus` instead of creating it internally:

```go
// Replace the import:
//   "github.com/sasrgita/crm-juridico/internal/whatsapp/infrastructure"
// Add:
//   "github.com/sasrgita/crm-juridico/internal/shared/events"

// Change NewModule signature:
func NewModule(db *gorm.DB, provider domain.WhatsAppProvider, eventBus events.EventBus, log *zap.Logger) *Module {
	// Remove: eventBus := infrastructure.NewMemoryEventBus()
	// Use the passed eventBus parameter everywhere
```

- [ ] **Step 2: Update whatsapp domain — replace EventBus interface with shared import**

In `internal/whatsapp/domain/events.go`, delete the file entirely. The interface and types now live in `internal/shared/events/event.go`.

Update all files in `internal/whatsapp/` that reference `domain.EventBus`, `domain.Event`, `domain.EventNewMessage`, or `domain.EventConversationUpdate` to import from `shared/events`:

- `application/receive_message.go`: change `domain.EventBus` → `events.EventBus`, `domain.Event{Type: domain.EventNewMessage, ...}` → `events.Event{Type: events.EventNewMessage, ...}`
- `application/send_message.go`: same changes
- `interfaces/http/handler.go`: change `domain.EventBus` → `events.EventBus`, `domain.Event` → `events.Event`
- `interfaces/http/handler_test.go`: change mock to use `events.Event` and `events.EventBus`

- [ ] **Step 3: Delete old files**

```bash
rm internal/whatsapp/domain/events.go
rm internal/whatsapp/infrastructure/memory_event_bus.go
rm internal/whatsapp/infrastructure/memory_event_bus_test.go
```

- [ ] **Step 4: Verify build and tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./... && go test ./internal/whatsapp/... -v`
Expected: build succeeds, all whatsapp tests pass

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: update whatsapp module to use shared EventBus"
```

---

## Task 4: Permission Domain — Errors and Entities

**Files:**
- Create: `internal/permission/domain/errors.go`
- Create: `internal/permission/domain/permission_group.go`
- Create: `internal/permission/domain/user_group.go`
- Create: `internal/permission/domain/permission.go`
- Create: `internal/permission/domain/view_profile.go`
- Create: `internal/permission/domain/group_funnel.go`

- [ ] **Step 1: Create domain errors**

```go
// internal/permission/domain/errors.go
package domain

import "errors"

var (
	// PermissionGroup
	ErrGroupNotFound       = errors.New("permission group not found")
	ErrGroupNameRequired   = errors.New("group name is required")
	ErrGroupNameTooLong    = errors.New("group name exceeds 100 characters")
	ErrGroupDescTooLong    = errors.New("group description exceeds 500 characters")
	ErrTenantIDRequired    = errors.New("tenant ID is required")

	// UserGroup
	ErrUserIDRequired      = errors.New("user ID is required")
	ErrGroupIDRequired     = errors.New("group ID is required")
	ErrUserAlreadyInGroup  = errors.New("user is already in this group")

	// Permission
	ErrResourceRequired    = errors.New("resource is required")
	ErrActionRequired      = errors.New("action is required")
	ErrInvalidResource     = errors.New("invalid resource")
	ErrInvalidAction       = errors.New("invalid action for resource")
	ErrPermissionXOR       = errors.New("permission must have either group_id or user_id, not both")
	ErrPermissionNotFound  = errors.New("permission not found")

	// ViewProfile
	ErrFunnelIDRequired    = errors.New("funnel ID is required")
	ErrViewProfileNotFound = errors.New("view profile not found")

	// GroupFunnel
	ErrGroupFunnelNotFound = errors.New("group-funnel association not found")
)
```

- [ ] **Step 2: Create PermissionGroup entity**

```go
// internal/permission/domain/permission_group.go
package domain

import "time"

const (
	MaxGroupNameLength = 100
	MaxGroupDescLength = 500
)

type PermissionGroup struct {
	ID          string
	TenantID    string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewPermissionGroup(id, tenantID, name, description string) (*PermissionGroup, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if name == "" {
		return nil, ErrGroupNameRequired
	}
	if len(name) > MaxGroupNameLength {
		return nil, ErrGroupNameTooLong
	}
	if len(description) > MaxGroupDescLength {
		return nil, ErrGroupDescTooLong
	}
	now := time.Now()
	return &PermissionGroup{
		ID: id, TenantID: tenantID, Name: name, Description: description,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (g *PermissionGroup) Update(name, description string) error {
	if name == "" {
		return ErrGroupNameRequired
	}
	if len(name) > MaxGroupNameLength {
		return ErrGroupNameTooLong
	}
	if len(description) > MaxGroupDescLength {
		return ErrGroupDescTooLong
	}
	g.Name = name
	g.Description = description
	g.UpdatedAt = time.Now()
	return nil
}
```

- [ ] **Step 3: Create UserGroup entity**

```go
// internal/permission/domain/user_group.go
package domain

import "time"

type UserGroup struct {
	ID        string
	UserID    string
	GroupID   string
	TenantID  string
	CreatedAt time.Time
}

func NewUserGroup(id, userID, groupID, tenantID string) (*UserGroup, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}
	if groupID == "" {
		return nil, ErrGroupIDRequired
	}
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	return &UserGroup{
		ID: id, UserID: userID, GroupID: groupID, TenantID: tenantID,
		CreatedAt: time.Now(),
	}, nil
}
```

- [ ] **Step 4: Create Permission entity with resource/action validation**

```go
// internal/permission/domain/permission.go
package domain

import "time"

// Valid resource:action pairs
var ValidPermissions = map[string][]string{
	"funnels":     {"manage", "customize"},
	"leads":       {"view", "manage"},
	"automations": {"manage"},
	"users":       {"manage"},
	"groups":      {"manage"},
	"products":    {"manage"},
	"specialists": {"manage"},
	"whatsapp":    {"view", "send"},
	"settings":    {"manage"},
}

type Permission struct {
	ID        string
	TenantID  string
	GroupID   string // empty = individual permission
	UserID    string // empty = group permission
	Resource  string
	Action    string
	CreatedAt time.Time
}

func NewPermission(id, tenantID, groupID, userID, resource, action string) (*Permission, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if resource == "" {
		return nil, ErrResourceRequired
	}
	if action == "" {
		return nil, ErrActionRequired
	}

	// XOR: exactly one of groupID or userID must be set
	if (groupID == "" && userID == "") || (groupID != "" && userID != "") {
		return nil, ErrPermissionXOR
	}

	// Validate resource
	actions, ok := ValidPermissions[resource]
	if !ok {
		return nil, ErrInvalidResource
	}

	// Validate action for resource
	valid := false
	for _, a := range actions {
		if a == action {
			valid = true
			break
		}
	}
	if !valid {
		return nil, ErrInvalidAction
	}

	return &Permission{
		ID: id, TenantID: tenantID, GroupID: groupID, UserID: userID,
		Resource: resource, Action: action, CreatedAt: time.Now(),
	}, nil
}

// IsGroupPermission returns true if this permission belongs to a group.
func (p *Permission) IsGroupPermission() bool {
	return p.GroupID != ""
}
```

- [ ] **Step 5: Create ViewProfile entity**

```go
// internal/permission/domain/view_profile.go
package domain

import "time"

type ViewProfile struct {
	ID             string
	GroupID        string
	FunnelID       string
	VisibleColumns []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewViewProfile(id, groupID, funnelID string, visibleColumns []string) (*ViewProfile, error) {
	if groupID == "" {
		return nil, ErrGroupIDRequired
	}
	if funnelID == "" {
		return nil, ErrFunnelIDRequired
	}
	now := time.Now()
	return &ViewProfile{
		ID: id, GroupID: groupID, FunnelID: funnelID,
		VisibleColumns: visibleColumns, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (vp *ViewProfile) UpdateColumns(columns []string) {
	vp.VisibleColumns = columns
	vp.UpdatedAt = time.Now()
}
```

- [ ] **Step 6: Create GroupFunnel entity**

```go
// internal/permission/domain/group_funnel.go
package domain

import "time"

type GroupFunnel struct {
	ID        string
	GroupID   string
	FunnelID  string
	ColumnIDs []string // empty = entire funnel
	CreatedAt time.Time
}

func NewGroupFunnel(id, groupID, funnelID string, columnIDs []string) (*GroupFunnel, error) {
	if groupID == "" {
		return nil, ErrGroupIDRequired
	}
	if funnelID == "" {
		return nil, ErrFunnelIDRequired
	}
	return &GroupFunnel{
		ID: id, GroupID: groupID, FunnelID: funnelID,
		ColumnIDs: columnIDs, CreatedAt: time.Now(),
	}, nil
}

// CoversColumn returns true if this association covers the given column.
func (gf *GroupFunnel) CoversColumn(columnID string) bool {
	if len(gf.ColumnIDs) == 0 {
		return true // empty = entire funnel
	}
	for _, id := range gf.ColumnIDs {
		if id == columnID {
			return true
		}
	}
	return false
}
```

- [ ] **Step 7: Verify build**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./internal/permission/...`
Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add internal/permission/domain/
git commit -m "feat(permission): add domain entities and errors"
```

---

## Task 5: Permission Domain — Repository Interfaces

**Files:**
- Create: `internal/permission/domain/repository.go`

- [ ] **Step 1: Define repository interfaces**

```go
// internal/permission/domain/repository.go
package domain

import "context"

type PermissionGroupRepository interface {
	Create(ctx context.Context, group *PermissionGroup) error
	FindByID(ctx context.Context, id string) (*PermissionGroup, error)
	Update(ctx context.Context, group *PermissionGroup) error
	Delete(ctx context.Context, id string) error
	FindByTenantID(ctx context.Context, tenantID string) ([]PermissionGroup, error)
}

type UserGroupRepository interface {
	Create(ctx context.Context, ug *UserGroup) error
	Delete(ctx context.Context, userID, groupID string) error
	FindByGroupID(ctx context.Context, groupID string) ([]UserGroup, error)
	FindByUserAndTenant(ctx context.Context, userID, tenantID string) ([]UserGroup, error)
	Exists(ctx context.Context, userID, groupID string) (bool, error)
}

type PermissionRepository interface {
	Create(ctx context.Context, perm *Permission) error
	Delete(ctx context.Context, id string) error
	FindByGroupID(ctx context.Context, groupID string) ([]Permission, error)
	FindByUserID(ctx context.Context, tenantID, userID string) ([]Permission, error)
	FindByGroupIDs(ctx context.Context, groupIDs []string) ([]Permission, error)
	DeleteByGroupAndResource(ctx context.Context, groupID, resource, action string) error
	DeleteByUserAndResource(ctx context.Context, tenantID, userID, resource, action string) error
}

type ViewProfileRepository interface {
	CreateOrUpdate(ctx context.Context, vp *ViewProfile) error
	FindByGroupID(ctx context.Context, groupID string) ([]ViewProfile, error)
	FindByGroupAndFunnel(ctx context.Context, groupID, funnelID string) (*ViewProfile, error)
	FindByGroupIDs(ctx context.Context, groupIDs []string, funnelID string) ([]ViewProfile, error)
	Delete(ctx context.Context, groupID, funnelID string) error
}

type GroupFunnelRepository interface {
	CreateOrUpdate(ctx context.Context, gf *GroupFunnel) error
	FindByGroupID(ctx context.Context, groupID string) ([]GroupFunnel, error)
	FindByFunnelID(ctx context.Context, funnelID string) ([]GroupFunnel, error)
	FindByFunnelAndColumn(ctx context.Context, funnelID, columnID string) ([]GroupFunnel, error)
	Delete(ctx context.Context, groupID, funnelID string) error
}

// OwnerChecker checks if a user is the owner of a tenant.
// Implemented by the auth module via adapter.
type OwnerChecker interface {
	IsOwner(ctx context.Context, userID, tenantID string) (bool, error)
}

// AdminChecker checks if a user has the admin role.
// Implemented by the auth module via adapter.
type AdminChecker interface {
	IsAdmin(ctx context.Context, userID string) (bool, error)
}
```

- [ ] **Step 2: Verify build**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./internal/permission/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/permission/domain/repository.go
git commit -m "feat(permission): add repository interfaces"
```

---

## Task 6: Permission Migrations

**Files:**
- Create: `migrations/000035_create_permission_groups.up.sql`
- Create: `migrations/000035_create_permission_groups.down.sql`
- Create: `migrations/000036_create_user_groups.up.sql`
- Create: `migrations/000036_create_user_groups.down.sql`
- Create: `migrations/000037_create_permissions.up.sql`
- Create: `migrations/000037_create_permissions.down.sql`
- Create: `migrations/000038_create_view_profiles.up.sql`
- Create: `migrations/000038_create_view_profiles.down.sql`
- Create: `migrations/000039_create_group_funnels.up.sql`
- Create: `migrations/000039_create_group_funnels.down.sql`

- [ ] **Step 1: Create permission_groups migration**

```sql
-- migrations/000035_create_permission_groups.up.sql
CREATE TABLE permission_groups (
    id CHAR(36) NOT NULL PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_permission_groups_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
-- migrations/000035_create_permission_groups.down.sql
DROP TABLE IF EXISTS permission_groups;
```

- [ ] **Step 2: Create user_groups migration**

```sql
-- migrations/000036_create_user_groups.up.sql
CREATE TABLE user_groups (
    id CHAR(36) NOT NULL PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    group_id CHAR(36) NOT NULL,
    tenant_id CHAR(36) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES permission_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE KEY uk_user_group (user_id, group_id),
    INDEX idx_user_groups_group (group_id),
    INDEX idx_user_groups_tenant_user (tenant_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
-- migrations/000036_create_user_groups.down.sql
DROP TABLE IF EXISTS user_groups;
```

- [ ] **Step 3: Create permissions migration**

```sql
-- migrations/000037_create_permissions.up.sql
CREATE TABLE permissions (
    id CHAR(36) NOT NULL PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    group_id CHAR(36) NULL,
    user_id CHAR(36) NULL,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES permission_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT chk_permission_xor CHECK (
        (group_id IS NOT NULL AND user_id IS NULL) OR
        (group_id IS NULL AND user_id IS NOT NULL)
    ),
    UNIQUE KEY uk_perm_group (tenant_id, group_id, resource, action),
    UNIQUE KEY uk_perm_user (tenant_id, user_id, resource, action),
    INDEX idx_permissions_tenant_user (tenant_id, user_id),
    INDEX idx_permissions_group (group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
-- migrations/000037_create_permissions.down.sql
DROP TABLE IF EXISTS permissions;
```

- [ ] **Step 4: Create view_profiles migration**

```sql
-- migrations/000038_create_view_profiles.up.sql
CREATE TABLE view_profiles (
    id CHAR(36) NOT NULL PRIMARY KEY,
    group_id CHAR(36) NOT NULL,
    funnel_id CHAR(36) NOT NULL,
    visible_columns JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES permission_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (funnel_id) REFERENCES funnels(id) ON DELETE CASCADE,
    UNIQUE KEY uk_view_profile (group_id, funnel_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
-- migrations/000038_create_view_profiles.down.sql
DROP TABLE IF EXISTS view_profiles;
```

- [ ] **Step 5: Create group_funnels migration**

```sql
-- migrations/000039_create_group_funnels.up.sql
CREATE TABLE group_funnels (
    id CHAR(36) NOT NULL PRIMARY KEY,
    group_id CHAR(36) NOT NULL,
    funnel_id CHAR(36) NOT NULL,
    column_ids JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES permission_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (funnel_id) REFERENCES funnels(id) ON DELETE CASCADE,
    UNIQUE KEY uk_group_funnel (group_id, funnel_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

```sql
-- migrations/000039_create_group_funnels.down.sql
DROP TABLE IF EXISTS group_funnels;
```

- [ ] **Step 6: Commit**

```bash
git add migrations/000035_* migrations/000036_* migrations/000037_* migrations/000038_* migrations/000039_*
git commit -m "feat(permission): add database migrations"
```

---

## Task 7: GORM Models and Repositories

**Files:**
- Create: `internal/permission/infrastructure/models.go`
- Create: `internal/permission/infrastructure/gorm_permission_group_repository.go`
- Create: `internal/permission/infrastructure/gorm_user_group_repository.go`
- Create: `internal/permission/infrastructure/gorm_permission_repository.go`
- Create: `internal/permission/infrastructure/gorm_view_profile_repository.go`
- Create: `internal/permission/infrastructure/gorm_group_funnel_repository.go`

- [ ] **Step 1: Create GORM models with mappers**

```go
// internal/permission/infrastructure/models.go
package infrastructure

import (
	"encoding/json"
	"time"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// --- PermissionGroup ---

type permissionGroupModel struct {
	ID          string    `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID    string    `gorm:"column:tenant_id;type:char(36);not null"`
	Name        string    `gorm:"column:name;type:varchar(100);not null"`
	Description string    `gorm:"column:description;type:varchar(500);not null;default:''"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (permissionGroupModel) TableName() string { return "permission_groups" }

func groupToModel(g *domain.PermissionGroup) *permissionGroupModel {
	return &permissionGroupModel{
		ID: g.ID, TenantID: g.TenantID, Name: g.Name,
		Description: g.Description, CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt,
	}
}

func groupToDomain(m *permissionGroupModel) *domain.PermissionGroup {
	return &domain.PermissionGroup{
		ID: m.ID, TenantID: m.TenantID, Name: m.Name,
		Description: m.Description, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// --- UserGroup ---

type userGroupModel struct {
	ID        string    `gorm:"primaryKey;column:id;type:char(36)"`
	UserID    string    `gorm:"column:user_id;type:char(36);not null"`
	GroupID   string    `gorm:"column:group_id;type:char(36);not null"`
	TenantID  string    `gorm:"column:tenant_id;type:char(36);not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (userGroupModel) TableName() string { return "user_groups" }

func userGroupToModel(ug *domain.UserGroup) *userGroupModel {
	return &userGroupModel{
		ID: ug.ID, UserID: ug.UserID, GroupID: ug.GroupID,
		TenantID: ug.TenantID, CreatedAt: ug.CreatedAt,
	}
}

func userGroupToDomain(m *userGroupModel) *domain.UserGroup {
	return &domain.UserGroup{
		ID: m.ID, UserID: m.UserID, GroupID: m.GroupID,
		TenantID: m.TenantID, CreatedAt: m.CreatedAt,
	}
}

// --- Permission ---

type permissionModel struct {
	ID        string    `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID  string    `gorm:"column:tenant_id;type:char(36);not null"`
	GroupID   *string   `gorm:"column:group_id;type:char(36)"`
	UserID    *string   `gorm:"column:user_id;type:char(36)"`
	Resource  string    `gorm:"column:resource;type:varchar(50);not null"`
	Action    string    `gorm:"column:action;type:varchar(50);not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (permissionModel) TableName() string { return "permissions" }

func permToModel(p *domain.Permission) *permissionModel {
	m := &permissionModel{
		ID: p.ID, TenantID: p.TenantID, Resource: p.Resource,
		Action: p.Action, CreatedAt: p.CreatedAt,
	}
	if p.GroupID != "" {
		m.GroupID = &p.GroupID
	}
	if p.UserID != "" {
		m.UserID = &p.UserID
	}
	return m
}

func permToDomain(m *permissionModel) *domain.Permission {
	p := &domain.Permission{
		ID: m.ID, TenantID: m.TenantID, Resource: m.Resource,
		Action: m.Action, CreatedAt: m.CreatedAt,
	}
	if m.GroupID != nil {
		p.GroupID = *m.GroupID
	}
	if m.UserID != nil {
		p.UserID = *m.UserID
	}
	return p
}

// --- ViewProfile ---

type viewProfileModel struct {
	ID             string    `gorm:"primaryKey;column:id;type:char(36)"`
	GroupID        string    `gorm:"column:group_id;type:char(36);not null"`
	FunnelID       string    `gorm:"column:funnel_id;type:char(36);not null"`
	VisibleColumns string    `gorm:"column:visible_columns;type:json;not null"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (viewProfileModel) TableName() string { return "view_profiles" }

func vpToModel(vp *domain.ViewProfile) *viewProfileModel {
	cols, _ := json.Marshal(vp.VisibleColumns)
	return &viewProfileModel{
		ID: vp.ID, GroupID: vp.GroupID, FunnelID: vp.FunnelID,
		VisibleColumns: string(cols), CreatedAt: vp.CreatedAt, UpdatedAt: vp.UpdatedAt,
	}
}

func vpToDomain(m *viewProfileModel) *domain.ViewProfile {
	var cols []string
	_ = json.Unmarshal([]byte(m.VisibleColumns), &cols)
	return &domain.ViewProfile{
		ID: m.ID, GroupID: m.GroupID, FunnelID: m.FunnelID,
		VisibleColumns: cols, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// --- GroupFunnel ---

type groupFunnelModel struct {
	ID        string    `gorm:"primaryKey;column:id;type:char(36)"`
	GroupID   string    `gorm:"column:group_id;type:char(36);not null"`
	FunnelID  string    `gorm:"column:funnel_id;type:char(36);not null"`
	ColumnIDs string    `gorm:"column:column_ids;type:json;not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (groupFunnelModel) TableName() string { return "group_funnels" }

func gfToModel(gf *domain.GroupFunnel) *groupFunnelModel {
	cols, _ := json.Marshal(gf.ColumnIDs)
	return &groupFunnelModel{
		ID: gf.ID, GroupID: gf.GroupID, FunnelID: gf.FunnelID,
		ColumnIDs: string(cols), CreatedAt: gf.CreatedAt,
	}
}

func gfToDomain(m *groupFunnelModel) *domain.GroupFunnel {
	var cols []string
	_ = json.Unmarshal([]byte(m.ColumnIDs), &cols)
	return &domain.GroupFunnel{
		ID: m.ID, GroupID: m.GroupID, FunnelID: m.FunnelID,
		ColumnIDs: cols, CreatedAt: m.CreatedAt,
	}
}
```

- [ ] **Step 2: Create PermissionGroupRepository**

```go
// internal/permission/infrastructure/gorm_permission_group_repository.go
package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type GormPermissionGroupRepository struct{ db *gorm.DB }

func NewGormPermissionGroupRepository(db *gorm.DB) *GormPermissionGroupRepository {
	return &GormPermissionGroupRepository{db: db}
}

func (r *GormPermissionGroupRepository) Create(ctx context.Context, group *domain.PermissionGroup) error {
	return r.db.WithContext(ctx).Create(groupToModel(group)).Error
}

func (r *GormPermissionGroupRepository) FindByID(ctx context.Context, id string) (*domain.PermissionGroup, error) {
	var m permissionGroupModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrGroupNotFound
		}
		return nil, err
	}
	return groupToDomain(&m), nil
}

func (r *GormPermissionGroupRepository) Update(ctx context.Context, group *domain.PermissionGroup) error {
	result := r.db.WithContext(ctx).Save(groupToModel(group))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrGroupNotFound
	}
	return nil
}

func (r *GormPermissionGroupRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&permissionGroupModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrGroupNotFound
	}
	return nil
}

func (r *GormPermissionGroupRepository) FindByTenantID(ctx context.Context, tenantID string) ([]domain.PermissionGroup, error) {
	var models []permissionGroupModel
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("name ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	groups := make([]domain.PermissionGroup, len(models))
	for i := range models {
		groups[i] = *groupToDomain(&models[i])
	}
	return groups, nil
}
```

- [ ] **Step 3: Create UserGroupRepository**

```go
// internal/permission/infrastructure/gorm_user_group_repository.go
package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type GormUserGroupRepository struct{ db *gorm.DB }

func NewGormUserGroupRepository(db *gorm.DB) *GormUserGroupRepository {
	return &GormUserGroupRepository{db: db}
}

func (r *GormUserGroupRepository) Create(ctx context.Context, ug *domain.UserGroup) error {
	return r.db.WithContext(ctx).Create(userGroupToModel(ug)).Error
}

func (r *GormUserGroupRepository) Delete(ctx context.Context, userID, groupID string) error {
	result := r.db.WithContext(ctx).Where("user_id = ? AND group_id = ?", userID, groupID).Delete(&userGroupModel{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *GormUserGroupRepository) FindByGroupID(ctx context.Context, groupID string) ([]domain.UserGroup, error) {
	var models []userGroupModel
	if err := r.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&models).Error; err != nil {
		return nil, err
	}
	ugs := make([]domain.UserGroup, len(models))
	for i := range models {
		ugs[i] = *userGroupToDomain(&models[i])
	}
	return ugs, nil
}

func (r *GormUserGroupRepository) FindByUserAndTenant(ctx context.Context, userID, tenantID string) ([]domain.UserGroup, error) {
	var models []userGroupModel
	if err := r.db.WithContext(ctx).Where("user_id = ? AND tenant_id = ?", userID, tenantID).Find(&models).Error; err != nil {
		return nil, err
	}
	ugs := make([]domain.UserGroup, len(models))
	for i := range models {
		ugs[i] = *userGroupToDomain(&models[i])
	}
	return ugs, nil
}

func (r *GormUserGroupRepository) Exists(ctx context.Context, userID, groupID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&userGroupModel{}).Where("user_id = ? AND group_id = ?", userID, groupID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
```

- [ ] **Step 4: Create PermissionRepository**

```go
// internal/permission/infrastructure/gorm_permission_repository.go
package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type GormPermissionRepository struct{ db *gorm.DB }

func NewGormPermissionRepository(db *gorm.DB) *GormPermissionRepository {
	return &GormPermissionRepository{db: db}
}

func (r *GormPermissionRepository) Create(ctx context.Context, perm *domain.Permission) error {
	return r.db.WithContext(ctx).Create(permToModel(perm)).Error
}

func (r *GormPermissionRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&permissionModel{}).Error
}

func (r *GormPermissionRepository) FindByGroupID(ctx context.Context, groupID string) ([]domain.Permission, error) {
	var models []permissionModel
	if err := r.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&models).Error; err != nil {
		return nil, err
	}
	perms := make([]domain.Permission, len(models))
	for i := range models {
		perms[i] = *permToDomain(&models[i])
	}
	return perms, nil
}

func (r *GormPermissionRepository) FindByUserID(ctx context.Context, tenantID, userID string) ([]domain.Permission, error) {
	var models []permissionModel
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND user_id = ?", tenantID, userID).Find(&models).Error; err != nil {
		return nil, err
	}
	perms := make([]domain.Permission, len(models))
	for i := range models {
		perms[i] = *permToDomain(&models[i])
	}
	return perms, nil
}

func (r *GormPermissionRepository) FindByGroupIDs(ctx context.Context, groupIDs []string) ([]domain.Permission, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	var models []permissionModel
	if err := r.db.WithContext(ctx).Where("group_id IN ?", groupIDs).Find(&models).Error; err != nil {
		return nil, err
	}
	perms := make([]domain.Permission, len(models))
	for i := range models {
		perms[i] = *permToDomain(&models[i])
	}
	return perms, nil
}

func (r *GormPermissionRepository) DeleteByGroupAndResource(ctx context.Context, groupID, resource, action string) error {
	return r.db.WithContext(ctx).Where("group_id = ? AND resource = ? AND action = ?", groupID, resource, action).Delete(&permissionModel{}).Error
}

func (r *GormPermissionRepository) DeleteByUserAndResource(ctx context.Context, tenantID, userID, resource, action string) error {
	return r.db.WithContext(ctx).Where("tenant_id = ? AND user_id = ? AND resource = ? AND action = ?", tenantID, userID, resource, action).Delete(&permissionModel{}).Error
}
```

- [ ] **Step 5: Create ViewProfileRepository**

```go
// internal/permission/infrastructure/gorm_view_profile_repository.go
package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type GormViewProfileRepository struct{ db *gorm.DB }

func NewGormViewProfileRepository(db *gorm.DB) *GormViewProfileRepository {
	return &GormViewProfileRepository{db: db}
}

func (r *GormViewProfileRepository) CreateOrUpdate(ctx context.Context, vp *domain.ViewProfile) error {
	m := vpToModel(vp)
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *GormViewProfileRepository) FindByGroupID(ctx context.Context, groupID string) ([]domain.ViewProfile, error) {
	var models []viewProfileModel
	if err := r.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&models).Error; err != nil {
		return nil, err
	}
	vps := make([]domain.ViewProfile, len(models))
	for i := range models {
		vps[i] = *vpToDomain(&models[i])
	}
	return vps, nil
}

func (r *GormViewProfileRepository) FindByGroupAndFunnel(ctx context.Context, groupID, funnelID string) (*domain.ViewProfile, error) {
	var m viewProfileModel
	if err := r.db.WithContext(ctx).Where("group_id = ? AND funnel_id = ?", groupID, funnelID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrViewProfileNotFound
		}
		return nil, err
	}
	return vpToDomain(&m), nil
}

func (r *GormViewProfileRepository) FindByGroupIDs(ctx context.Context, groupIDs []string, funnelID string) ([]domain.ViewProfile, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	var models []viewProfileModel
	if err := r.db.WithContext(ctx).Where("group_id IN ? AND funnel_id = ?", groupIDs, funnelID).Find(&models).Error; err != nil {
		return nil, err
	}
	vps := make([]domain.ViewProfile, len(models))
	for i := range models {
		vps[i] = *vpToDomain(&models[i])
	}
	return vps, nil
}

func (r *GormViewProfileRepository) Delete(ctx context.Context, groupID, funnelID string) error {
	return r.db.WithContext(ctx).Where("group_id = ? AND funnel_id = ?", groupID, funnelID).Delete(&viewProfileModel{}).Error
}
```

- [ ] **Step 6: Create GroupFunnelRepository**

```go
// internal/permission/infrastructure/gorm_group_funnel_repository.go
package infrastructure

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type GormGroupFunnelRepository struct{ db *gorm.DB }

func NewGormGroupFunnelRepository(db *gorm.DB) *GormGroupFunnelRepository {
	return &GormGroupFunnelRepository{db: db}
}

func (r *GormGroupFunnelRepository) CreateOrUpdate(ctx context.Context, gf *domain.GroupFunnel) error {
	return r.db.WithContext(ctx).Save(gfToModel(gf)).Error
}

func (r *GormGroupFunnelRepository) FindByGroupID(ctx context.Context, groupID string) ([]domain.GroupFunnel, error) {
	var models []groupFunnelModel
	if err := r.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&models).Error; err != nil {
		return nil, err
	}
	gfs := make([]domain.GroupFunnel, len(models))
	for i := range models {
		gfs[i] = *gfToDomain(&models[i])
	}
	return gfs, nil
}

func (r *GormGroupFunnelRepository) FindByFunnelID(ctx context.Context, funnelID string) ([]domain.GroupFunnel, error) {
	var models []groupFunnelModel
	if err := r.db.WithContext(ctx).Where("funnel_id = ?", funnelID).Find(&models).Error; err != nil {
		return nil, err
	}
	gfs := make([]domain.GroupFunnel, len(models))
	for i := range models {
		gfs[i] = *gfToDomain(&models[i])
	}
	return gfs, nil
}

func (r *GormGroupFunnelRepository) FindByFunnelAndColumn(ctx context.Context, funnelID, columnID string) ([]domain.GroupFunnel, error) {
	var models []groupFunnelModel
	if err := r.db.WithContext(ctx).Where("funnel_id = ?", funnelID).Find(&models).Error; err != nil {
		return nil, err
	}
	// Filter in application: check if columnIDs is empty (whole funnel) or contains columnID
	var result []domain.GroupFunnel
	for i := range models {
		gf := gfToDomain(&models[i])
		if gf.CoversColumn(columnID) {
			result = append(result, *gf)
		}
	}
	return result, nil
}

func (r *GormGroupFunnelRepository) Delete(ctx context.Context, groupID, funnelID string) error {
	return r.db.WithContext(ctx).Where("group_id = ? AND funnel_id = ?", groupID, funnelID).Delete(&groupFunnelModel{}).Error
}
```

- [ ] **Step 7: Verify build**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./internal/permission/...`
Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add internal/permission/infrastructure/
git commit -m "feat(permission): add GORM models and repositories"
```

---

## Task 8: HasPermission Resolver

**Files:**
- Create: `internal/permission/application/mocks_test.go`
- Create: `internal/permission/application/resolve_permission.go`
- Create: `internal/permission/application/resolve_permission_test.go`

- [ ] **Step 1: Create shared mocks for tests**

```go
// internal/permission/application/mocks_test.go
package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// --- Mock PermissionGroupRepository ---

type mockGroupRepo struct {
	groups map[string]*domain.PermissionGroup
}

func newMockGroupRepo() *mockGroupRepo {
	return &mockGroupRepo{groups: make(map[string]*domain.PermissionGroup)}
}

func (m *mockGroupRepo) Create(_ context.Context, g *domain.PermissionGroup) error {
	m.groups[g.ID] = g
	return nil
}

func (m *mockGroupRepo) FindByID(_ context.Context, id string) (*domain.PermissionGroup, error) {
	if g, ok := m.groups[id]; ok {
		return g, nil
	}
	return nil, domain.ErrGroupNotFound
}

func (m *mockGroupRepo) Update(_ context.Context, g *domain.PermissionGroup) error {
	m.groups[g.ID] = g
	return nil
}

func (m *mockGroupRepo) Delete(_ context.Context, id string) error {
	delete(m.groups, id)
	return nil
}

func (m *mockGroupRepo) FindByTenantID(_ context.Context, tenantID string) ([]domain.PermissionGroup, error) {
	var result []domain.PermissionGroup
	for _, g := range m.groups {
		if g.TenantID == tenantID {
			result = append(result, *g)
		}
	}
	return result, nil
}

// --- Mock UserGroupRepository ---

type mockUserGroupRepo struct {
	items []domain.UserGroup
}

func newMockUserGroupRepo() *mockUserGroupRepo {
	return &mockUserGroupRepo{}
}

func (m *mockUserGroupRepo) Create(_ context.Context, ug *domain.UserGroup) error {
	m.items = append(m.items, *ug)
	return nil
}

func (m *mockUserGroupRepo) Delete(_ context.Context, userID, groupID string) error {
	for i, ug := range m.items {
		if ug.UserID == userID && ug.GroupID == groupID {
			m.items = append(m.items[:i], m.items[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockUserGroupRepo) FindByGroupID(_ context.Context, groupID string) ([]domain.UserGroup, error) {
	var result []domain.UserGroup
	for _, ug := range m.items {
		if ug.GroupID == groupID {
			result = append(result, ug)
		}
	}
	return result, nil
}

func (m *mockUserGroupRepo) FindByUserAndTenant(_ context.Context, userID, tenantID string) ([]domain.UserGroup, error) {
	var result []domain.UserGroup
	for _, ug := range m.items {
		if ug.UserID == userID && ug.TenantID == tenantID {
			result = append(result, ug)
		}
	}
	return result, nil
}

func (m *mockUserGroupRepo) Exists(_ context.Context, userID, groupID string) (bool, error) {
	for _, ug := range m.items {
		if ug.UserID == userID && ug.GroupID == groupID {
			return true, nil
		}
	}
	return false, nil
}

// --- Mock PermissionRepository ---

type mockPermRepo struct {
	items []domain.Permission
}

func newMockPermRepo() *mockPermRepo {
	return &mockPermRepo{}
}

func (m *mockPermRepo) Create(_ context.Context, p *domain.Permission) error {
	m.items = append(m.items, *p)
	return nil
}

func (m *mockPermRepo) Delete(_ context.Context, id string) error {
	for i, p := range m.items {
		if p.ID == id {
			m.items = append(m.items[:i], m.items[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockPermRepo) FindByGroupID(_ context.Context, groupID string) ([]domain.Permission, error) {
	var result []domain.Permission
	for _, p := range m.items {
		if p.GroupID == groupID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPermRepo) FindByUserID(_ context.Context, tenantID, userID string) ([]domain.Permission, error) {
	var result []domain.Permission
	for _, p := range m.items {
		if p.TenantID == tenantID && p.UserID == userID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPermRepo) FindByGroupIDs(_ context.Context, groupIDs []string) ([]domain.Permission, error) {
	idSet := make(map[string]bool)
	for _, id := range groupIDs {
		idSet[id] = true
	}
	var result []domain.Permission
	for _, p := range m.items {
		if idSet[p.GroupID] {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPermRepo) DeleteByGroupAndResource(_ context.Context, groupID, resource, action string) error {
	var filtered []domain.Permission
	for _, p := range m.items {
		if !(p.GroupID == groupID && p.Resource == resource && p.Action == action) {
			filtered = append(filtered, p)
		}
	}
	m.items = filtered
	return nil
}

func (m *mockPermRepo) DeleteByUserAndResource(_ context.Context, tenantID, userID, resource, action string) error {
	var filtered []domain.Permission
	for _, p := range m.items {
		if !(p.TenantID == tenantID && p.UserID == userID && p.Resource == resource && p.Action == action) {
			filtered = append(filtered, p)
		}
	}
	m.items = filtered
	return nil
}

// --- Mock OwnerChecker ---

type mockOwnerChecker struct {
	owners map[string]bool // key: "userID:tenantID"
}

func newMockOwnerChecker() *mockOwnerChecker {
	return &mockOwnerChecker{owners: make(map[string]bool)}
}

func (m *mockOwnerChecker) IsOwner(_ context.Context, userID, tenantID string) (bool, error) {
	return m.owners[userID+":"+tenantID], nil
}

func (m *mockOwnerChecker) setOwner(userID, tenantID string) {
	m.owners[userID+":"+tenantID] = true
}

// --- Mock AdminChecker ---

type mockAdminChecker struct {
	admins map[string]bool
}

func newMockAdminChecker() *mockAdminChecker {
	return &mockAdminChecker{admins: make(map[string]bool)}
}

func (m *mockAdminChecker) IsAdmin(_ context.Context, userID string) (bool, error) {
	return m.admins[userID], nil
}

func (m *mockAdminChecker) setAdmin(userID string) {
	m.admins[userID] = true
}
```

- [ ] **Step 2: Write failing tests for HasPermission resolver**

```go
// internal/permission/application/resolve_permission_test.go
package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePermission_OwnerBypass(t *testing.T) {
	ownerChecker := newMockOwnerChecker()
	ownerChecker.setOwner("user-1", "tenant-1")
	uc := NewResolvePermissionUseCase(newMockPermRepo(), newMockUserGroupRepo(), ownerChecker, newMockAdminChecker())

	has, err := uc.HasPermission(context.Background(), "user-1", "tenant-1", "automations", "manage")
	require.NoError(t, err)
	assert.True(t, has)
}

func TestResolvePermission_AdminBypass(t *testing.T) {
	adminChecker := newMockAdminChecker()
	adminChecker.setAdmin("user-1")
	uc := NewResolvePermissionUseCase(newMockPermRepo(), newMockUserGroupRepo(), newMockOwnerChecker(), adminChecker)

	has, err := uc.HasPermission(context.Background(), "user-1", "tenant-1", "automations", "manage")
	require.NoError(t, err)
	assert.True(t, has)
}

func TestResolvePermission_IndividualPermission(t *testing.T) {
	permRepo := newMockPermRepo()
	permRepo.items = append(permRepo.items, newTestPerm("tenant-1", "", "user-1", "automations", "manage"))
	uc := NewResolvePermissionUseCase(permRepo, newMockUserGroupRepo(), newMockOwnerChecker(), newMockAdminChecker())

	has, err := uc.HasPermission(context.Background(), "user-1", "tenant-1", "automations", "manage")
	require.NoError(t, err)
	assert.True(t, has)
}

func TestResolvePermission_GroupPermission(t *testing.T) {
	permRepo := newMockPermRepo()
	permRepo.items = append(permRepo.items, newTestPerm("tenant-1", "group-1", "", "leads", "view"))

	ugRepo := newMockUserGroupRepo()
	ugRepo.items = append(ugRepo.items, newTestUserGroup("user-1", "group-1", "tenant-1"))

	uc := NewResolvePermissionUseCase(permRepo, ugRepo, newMockOwnerChecker(), newMockAdminChecker())

	has, err := uc.HasPermission(context.Background(), "user-1", "tenant-1", "leads", "view")
	require.NoError(t, err)
	assert.True(t, has)
}

func TestResolvePermission_UnionAcrossGroups(t *testing.T) {
	permRepo := newMockPermRepo()
	permRepo.items = append(permRepo.items, newTestPerm("tenant-1", "group-1", "", "leads", "view"))
	permRepo.items = append(permRepo.items, newTestPerm("tenant-1", "group-2", "", "automations", "manage"))

	ugRepo := newMockUserGroupRepo()
	ugRepo.items = append(ugRepo.items, newTestUserGroup("user-1", "group-1", "tenant-1"))
	ugRepo.items = append(ugRepo.items, newTestUserGroup("user-1", "group-2", "tenant-1"))

	uc := NewResolvePermissionUseCase(permRepo, ugRepo, newMockOwnerChecker(), newMockAdminChecker())

	has1, _ := uc.HasPermission(context.Background(), "user-1", "tenant-1", "leads", "view")
	has2, _ := uc.HasPermission(context.Background(), "user-1", "tenant-1", "automations", "manage")
	assert.True(t, has1, "should have leads:view from group-1")
	assert.True(t, has2, "should have automations:manage from group-2")
}

func TestResolvePermission_NoPermission(t *testing.T) {
	uc := NewResolvePermissionUseCase(newMockPermRepo(), newMockUserGroupRepo(), newMockOwnerChecker(), newMockAdminChecker())

	has, err := uc.HasPermission(context.Background(), "user-1", "tenant-1", "automations", "manage")
	require.NoError(t, err)
	assert.False(t, has)
}

func TestResolvePermission_IndividualOverridesEmptyGroup(t *testing.T) {
	// User has no group permission but has individual permission
	permRepo := newMockPermRepo()
	permRepo.items = append(permRepo.items, newTestPerm("tenant-1", "", "user-1", "settings", "manage"))

	uc := NewResolvePermissionUseCase(permRepo, newMockUserGroupRepo(), newMockOwnerChecker(), newMockAdminChecker())

	has, err := uc.HasPermission(context.Background(), "user-1", "tenant-1", "settings", "manage")
	require.NoError(t, err)
	assert.True(t, has)
}

// --- helpers ---

func newTestPerm(tenantID, groupID, userID, resource, action string) domain.Permission {
	return domain.Permission{
		ID: tenantID + groupID + userID + resource + action,
		TenantID: tenantID, GroupID: groupID, UserID: userID,
		Resource: resource, Action: action,
	}
}

func newTestUserGroup(userID, groupID, tenantID string) domain.UserGroup {
	return domain.UserGroup{
		ID: userID + groupID, UserID: userID, GroupID: groupID, TenantID: tenantID,
	}
}
```

- [ ] **Step 3: Run tests — verify they fail**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/permission/application/ -v -run TestResolve`
Expected: FAIL — `NewResolvePermissionUseCase` not defined

- [ ] **Step 4: Implement the resolver**

```go
// internal/permission/application/resolve_permission.go
package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type ResolvePermissionUseCase struct {
	permRepo     domain.PermissionRepository
	ugRepo       domain.UserGroupRepository
	ownerChecker domain.OwnerChecker
	adminChecker domain.AdminChecker
}

func NewResolvePermissionUseCase(
	permRepo domain.PermissionRepository,
	ugRepo domain.UserGroupRepository,
	ownerChecker domain.OwnerChecker,
	adminChecker domain.AdminChecker,
) *ResolvePermissionUseCase {
	return &ResolvePermissionUseCase{
		permRepo: permRepo, ugRepo: ugRepo,
		ownerChecker: ownerChecker, adminChecker: adminChecker,
	}
}

// HasPermission checks if a user has a specific permission via union resolution.
func (uc *ResolvePermissionUseCase) HasPermission(ctx context.Context, userID, tenantID, resource, action string) (bool, error) {
	// 1. Owner bypass
	isOwner, err := uc.ownerChecker.IsOwner(ctx, userID, tenantID)
	if err != nil {
		return false, err
	}
	if isOwner {
		return true, nil
	}

	// 2. Admin bypass
	isAdmin, err := uc.adminChecker.IsAdmin(ctx, userID)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}

	// 3. Individual permissions
	userPerms, err := uc.permRepo.FindByUserID(ctx, tenantID, userID)
	if err != nil {
		return false, err
	}
	for _, p := range userPerms {
		if p.Resource == resource && p.Action == action {
			return true, nil
		}
	}

	// 4. Group permissions (union)
	userGroups, err := uc.ugRepo.FindByUserAndTenant(ctx, userID, tenantID)
	if err != nil {
		return false, err
	}
	if len(userGroups) == 0 {
		return false, nil
	}

	groupIDs := make([]string, len(userGroups))
	for i, ug := range userGroups {
		groupIDs[i] = ug.GroupID
	}

	groupPerms, err := uc.permRepo.FindByGroupIDs(ctx, groupIDs)
	if err != nil {
		return false, err
	}
	for _, p := range groupPerms {
		if p.Resource == resource && p.Action == action {
			return true, nil
		}
	}

	return false, nil
}
```

- [ ] **Step 5: Run tests — verify they pass**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/permission/application/ -v -run TestResolve`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/permission/application/
git commit -m "feat(permission): add HasPermission resolver with union semantics"
```

---

## Task 9: RequirePermission Middleware

**Files:**
- Create: `internal/shared/middleware/permission.go`
- Modify: `internal/shared/module/module.go`

- [ ] **Step 1: Add RequirePermission to Middlewares struct**

In `internal/shared/module/module.go`, add the permission middleware factory:

```go
package module

import "github.com/gin-gonic/gin"

type Middlewares struct {
	Auth              gin.HandlerFunc
	Tenant            gin.HandlerFunc
	Admin             gin.HandlerFunc
	RequirePermission func(resource, action string) gin.HandlerFunc
}

type Module interface {
	Name() string
	RegisterRoutes(router *gin.Engine, mw Middlewares)
}
```

- [ ] **Step 2: Create the permission middleware**

```go
// internal/shared/middleware/permission.go
package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PermissionChecker is the interface that the permission resolver must implement.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, tenantID, resource, action string) (bool, error)
}

// RequirePermission returns a middleware factory that checks permissions.
func RequirePermission(checker PermissionChecker) func(resource, action string) gin.HandlerFunc {
	return func(resource, action string) gin.HandlerFunc {
		return func(c *gin.Context) {
			claims := GetClaims(c.Request.Context())
			if claims == nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
				return
			}

			tenantID := GetTenantID(c.Request.Context())
			if tenantID == "" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "tenant selection required"})
				return
			}

			has, err := checker.HasPermission(c.Request.Context(), claims.UserID, tenantID, resource, action)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to check permissions"})
				return
			}

			if !has {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
				return
			}

			c.Next()
		}
	}
}
```

- [ ] **Step 3: Verify build**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/shared/middleware/permission.go internal/shared/module/module.go
git commit -m "feat(permission): add RequirePermission middleware"
```

---

## Task 10: CRUD Groups Use Cases

**Files:**
- Create: `internal/permission/application/create_group.go`
- Create: `internal/permission/application/create_group_test.go`
- Create: `internal/permission/application/update_group.go`
- Create: `internal/permission/application/list_groups.go`
- Create: `internal/permission/application/delete_group.go`

- [ ] **Step 1: Write failing tests for CreateGroup**

```go
// internal/permission/application/create_group_test.go
package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

func TestCreateGroup_Success(t *testing.T) {
	repo := newMockGroupRepo()
	uc := NewCreateGroupUseCase(repo)

	output, err := uc.Execute(context.Background(), CreateGroupInput{
		TenantID:    "tenant-1",
		Name:        "Vendas",
		Description: "Equipe de vendas",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.ID)
	assert.Equal(t, "Vendas", output.Name)
	assert.Equal(t, "Equipe de vendas", output.Description)
}

func TestCreateGroup_EmptyName(t *testing.T) {
	uc := NewCreateGroupUseCase(newMockGroupRepo())

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		TenantID: "tenant-1", Name: "",
	})

	assert.ErrorIs(t, err, domain.ErrGroupNameRequired)
}

func TestCreateGroup_EmptyTenantID(t *testing.T) {
	uc := NewCreateGroupUseCase(newMockGroupRepo())

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		TenantID: "", Name: "Grupo",
	})

	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}
```

- [ ] **Step 2: Run tests — verify fail**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/permission/application/ -v -run TestCreateGroup`
Expected: FAIL

- [ ] **Step 3: Implement CreateGroup, UpdateGroup, ListGroups, DeleteGroup**

```go
// internal/permission/application/create_group.go
package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type CreateGroupInput struct {
	TenantID    string
	Name        string
	Description string
}

type GroupOutput struct {
	ID          string
	Name        string
	Description string
}

type CreateGroupUseCase struct {
	groupRepo domain.PermissionGroupRepository
}

func NewCreateGroupUseCase(groupRepo domain.PermissionGroupRepository) *CreateGroupUseCase {
	return &CreateGroupUseCase{groupRepo: groupRepo}
}

func (uc *CreateGroupUseCase) Execute(ctx context.Context, input CreateGroupInput) (*GroupOutput, error) {
	group, err := domain.NewPermissionGroup(uuid.New().String(), input.TenantID, input.Name, input.Description)
	if err != nil {
		return nil, err
	}
	if err := uc.groupRepo.Create(ctx, group); err != nil {
		return nil, err
	}
	return &GroupOutput{ID: group.ID, Name: group.Name, Description: group.Description}, nil
}
```

```go
// internal/permission/application/update_group.go
package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type UpdateGroupInput struct {
	ID          string
	Name        string
	Description string
}

type UpdateGroupUseCase struct {
	groupRepo domain.PermissionGroupRepository
}

func NewUpdateGroupUseCase(groupRepo domain.PermissionGroupRepository) *UpdateGroupUseCase {
	return &UpdateGroupUseCase{groupRepo: groupRepo}
}

func (uc *UpdateGroupUseCase) Execute(ctx context.Context, input UpdateGroupInput) (*GroupOutput, error) {
	group, err := uc.groupRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if err := group.Update(input.Name, input.Description); err != nil {
		return nil, err
	}
	if err := uc.groupRepo.Update(ctx, group); err != nil {
		return nil, err
	}
	return &GroupOutput{ID: group.ID, Name: group.Name, Description: group.Description}, nil
}
```

```go
// internal/permission/application/list_groups.go
package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type ListGroupsUseCase struct {
	groupRepo domain.PermissionGroupRepository
}

func NewListGroupsUseCase(groupRepo domain.PermissionGroupRepository) *ListGroupsUseCase {
	return &ListGroupsUseCase{groupRepo: groupRepo}
}

func (uc *ListGroupsUseCase) Execute(ctx context.Context, tenantID string) ([]GroupOutput, error) {
	groups, err := uc.groupRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	output := make([]GroupOutput, len(groups))
	for i, g := range groups {
		output[i] = GroupOutput{ID: g.ID, Name: g.Name, Description: g.Description}
	}
	return output, nil
}
```

```go
// internal/permission/application/delete_group.go
package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type DeleteGroupUseCase struct {
	groupRepo domain.PermissionGroupRepository
}

func NewDeleteGroupUseCase(groupRepo domain.PermissionGroupRepository) *DeleteGroupUseCase {
	return &DeleteGroupUseCase{groupRepo: groupRepo}
}

func (uc *DeleteGroupUseCase) Execute(ctx context.Context, id string) error {
	return uc.groupRepo.Delete(ctx, id)
}
```

- [ ] **Step 4: Run tests — verify pass**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/permission/application/ -v -run TestCreateGroup`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/permission/application/
git commit -m "feat(permission): add CRUD group use cases"
```

---

## Task 11: Manage Members and Permissions Use Cases

**Files:**
- Create: `internal/permission/application/manage_members.go`
- Create: `internal/permission/application/manage_members_test.go`
- Create: `internal/permission/application/manage_permissions.go`
- Create: `internal/permission/application/manage_permissions_test.go`

- [ ] **Step 1: Write failing tests for member management**

```go
// internal/permission/application/manage_members_test.go
package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

func TestAddMember_Success(t *testing.T) {
	ugRepo := newMockUserGroupRepo()
	groupRepo := newMockGroupRepo()
	groupRepo.groups["group-1"] = &domain.PermissionGroup{ID: "group-1", TenantID: "tenant-1"}
	uc := NewManageMembersUseCase(ugRepo, groupRepo)

	err := uc.AddMember(context.Background(), "user-1", "group-1", "tenant-1")
	require.NoError(t, err)
	assert.Len(t, ugRepo.items, 1)
}

func TestAddMember_AlreadyInGroup(t *testing.T) {
	ugRepo := newMockUserGroupRepo()
	ugRepo.items = append(ugRepo.items, newTestUserGroup("user-1", "group-1", "tenant-1"))
	groupRepo := newMockGroupRepo()
	groupRepo.groups["group-1"] = &domain.PermissionGroup{ID: "group-1", TenantID: "tenant-1"}
	uc := NewManageMembersUseCase(ugRepo, groupRepo)

	err := uc.AddMember(context.Background(), "user-1", "group-1", "tenant-1")
	assert.ErrorIs(t, err, domain.ErrUserAlreadyInGroup)
}

func TestRemoveMember_Success(t *testing.T) {
	ugRepo := newMockUserGroupRepo()
	ugRepo.items = append(ugRepo.items, newTestUserGroup("user-1", "group-1", "tenant-1"))
	uc := NewManageMembersUseCase(ugRepo, newMockGroupRepo())

	err := uc.RemoveMember(context.Background(), "user-1", "group-1")
	require.NoError(t, err)
	assert.Empty(t, ugRepo.items)
}

func TestListMembers_Success(t *testing.T) {
	ugRepo := newMockUserGroupRepo()
	ugRepo.items = append(ugRepo.items, newTestUserGroup("user-1", "group-1", "tenant-1"))
	ugRepo.items = append(ugRepo.items, newTestUserGroup("user-2", "group-1", "tenant-1"))
	uc := NewManageMembersUseCase(ugRepo, newMockGroupRepo())

	members, err := uc.ListMembers(context.Background(), "group-1")
	require.NoError(t, err)
	assert.Len(t, members, 2)
}
```

- [ ] **Step 2: Run tests — verify fail**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/permission/application/ -v -run TestAddMember`
Expected: FAIL

- [ ] **Step 3: Implement ManageMembers use case**

```go
// internal/permission/application/manage_members.go
package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type ManageMembersUseCase struct {
	ugRepo    domain.UserGroupRepository
	groupRepo domain.PermissionGroupRepository
}

func NewManageMembersUseCase(ugRepo domain.UserGroupRepository, groupRepo domain.PermissionGroupRepository) *ManageMembersUseCase {
	return &ManageMembersUseCase{ugRepo: ugRepo, groupRepo: groupRepo}
}

func (uc *ManageMembersUseCase) AddMember(ctx context.Context, userID, groupID, tenantID string) error {
	// Verify group exists
	if _, err := uc.groupRepo.FindByID(ctx, groupID); err != nil {
		return err
	}

	exists, err := uc.ugRepo.Exists(ctx, userID, groupID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrUserAlreadyInGroup
	}

	ug, err := domain.NewUserGroup(uuid.New().String(), userID, groupID, tenantID)
	if err != nil {
		return err
	}
	return uc.ugRepo.Create(ctx, ug)
}

func (uc *ManageMembersUseCase) RemoveMember(ctx context.Context, userID, groupID string) error {
	return uc.ugRepo.Delete(ctx, userID, groupID)
}

func (uc *ManageMembersUseCase) ListMembers(ctx context.Context, groupID string) ([]MemberOutput, error) {
	ugs, err := uc.ugRepo.FindByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	output := make([]MemberOutput, len(ugs))
	for i, ug := range ugs {
		output[i] = MemberOutput{UserID: ug.UserID, GroupID: ug.GroupID}
	}
	return output, nil
}

type MemberOutput struct {
	UserID  string
	GroupID string
}
```

- [ ] **Step 4: Write failing tests for permission management**

```go
// internal/permission/application/manage_permissions_test.go
package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

func TestSetGroupPermissions_Success(t *testing.T) {
	permRepo := newMockPermRepo()
	uc := NewManagePermissionsUseCase(permRepo)

	err := uc.SetGroupPermissions(context.Background(), "tenant-1", "group-1", []PermissionInput{
		{Resource: "leads", Action: "view"},
		{Resource: "leads", Action: "manage"},
	})

	require.NoError(t, err)
	assert.Len(t, permRepo.items, 2)
}

func TestSetGroupPermissions_InvalidResource(t *testing.T) {
	uc := NewManagePermissionsUseCase(newMockPermRepo())

	err := uc.SetGroupPermissions(context.Background(), "tenant-1", "group-1", []PermissionInput{
		{Resource: "invalid", Action: "manage"},
	})

	assert.ErrorIs(t, err, domain.ErrInvalidResource)
}

func TestSetUserPermissions_Success(t *testing.T) {
	permRepo := newMockPermRepo()
	uc := NewManagePermissionsUseCase(permRepo)

	err := uc.SetUserPermissions(context.Background(), "tenant-1", "user-1", []PermissionInput{
		{Resource: "automations", Action: "manage"},
	})

	require.NoError(t, err)
	assert.Len(t, permRepo.items, 1)
	assert.Equal(t, "user-1", permRepo.items[0].UserID)
	assert.Empty(t, permRepo.items[0].GroupID)
}

func TestGetGroupPermissions_Success(t *testing.T) {
	permRepo := newMockPermRepo()
	permRepo.items = append(permRepo.items, newTestPerm("tenant-1", "group-1", "", "leads", "view"))
	uc := NewManagePermissionsUseCase(permRepo)

	perms, err := uc.GetGroupPermissions(context.Background(), "group-1")
	require.NoError(t, err)
	assert.Len(t, perms, 1)
	assert.Equal(t, "leads", perms[0].Resource)
}
```

- [ ] **Step 5: Implement ManagePermissions use case**

```go
// internal/permission/application/manage_permissions.go
package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type PermissionInput struct {
	Resource string
	Action   string
}

type PermissionOutput struct {
	ID       string
	Resource string
	Action   string
}

type ManagePermissionsUseCase struct {
	permRepo domain.PermissionRepository
}

func NewManagePermissionsUseCase(permRepo domain.PermissionRepository) *ManagePermissionsUseCase {
	return &ManagePermissionsUseCase{permRepo: permRepo}
}

// SetGroupPermissions replaces all permissions for a group.
func (uc *ManagePermissionsUseCase) SetGroupPermissions(ctx context.Context, tenantID, groupID string, perms []PermissionInput) error {
	// Validate all first
	for _, p := range perms {
		if _, err := domain.NewPermission("tmp", tenantID, groupID, "", p.Resource, p.Action); err != nil {
			return err
		}
	}

	// Delete existing group permissions
	existing, err := uc.permRepo.FindByGroupID(ctx, groupID)
	if err != nil {
		return err
	}
	for _, e := range existing {
		if err := uc.permRepo.Delete(ctx, e.ID); err != nil {
			return err
		}
	}

	// Create new permissions
	for _, p := range perms {
		perm, _ := domain.NewPermission(uuid.New().String(), tenantID, groupID, "", p.Resource, p.Action)
		if err := uc.permRepo.Create(ctx, perm); err != nil {
			return err
		}
	}
	return nil
}

// SetUserPermissions replaces all individual permissions for a user.
func (uc *ManagePermissionsUseCase) SetUserPermissions(ctx context.Context, tenantID, userID string, perms []PermissionInput) error {
	for _, p := range perms {
		if _, err := domain.NewPermission("tmp", tenantID, "", userID, p.Resource, p.Action); err != nil {
			return err
		}
	}

	existing, err := uc.permRepo.FindByUserID(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	for _, e := range existing {
		if err := uc.permRepo.Delete(ctx, e.ID); err != nil {
			return err
		}
	}

	for _, p := range perms {
		perm, _ := domain.NewPermission(uuid.New().String(), tenantID, "", userID, p.Resource, p.Action)
		if err := uc.permRepo.Create(ctx, perm); err != nil {
			return err
		}
	}
	return nil
}

func (uc *ManagePermissionsUseCase) GetGroupPermissions(ctx context.Context, groupID string) ([]PermissionOutput, error) {
	perms, err := uc.permRepo.FindByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	output := make([]PermissionOutput, len(perms))
	for i, p := range perms {
		output[i] = PermissionOutput{ID: p.ID, Resource: p.Resource, Action: p.Action}
	}
	return output, nil
}

func (uc *ManagePermissionsUseCase) GetUserPermissions(ctx context.Context, tenantID, userID string) ([]PermissionOutput, error) {
	perms, err := uc.permRepo.FindByUserID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	output := make([]PermissionOutput, len(perms))
	for i, p := range perms {
		output[i] = PermissionOutput{ID: p.ID, Resource: p.Resource, Action: p.Action}
	}
	return output, nil
}
```

- [ ] **Step 6: Run all tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/permission/... -v`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add internal/permission/application/
git commit -m "feat(permission): add member and permission management use cases"
```

---

## Task 12: ViewProfile and GroupFunnel Use Cases

**Files:**
- Create: `internal/permission/application/manage_view_profiles.go`
- Create: `internal/permission/application/manage_group_funnels.go`

- [ ] **Step 1: Implement ViewProfile use case**

```go
// internal/permission/application/manage_view_profiles.go
package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type ViewProfileInput struct {
	GroupID        string
	FunnelID       string
	VisibleColumns []string
}

type ViewProfileOutput struct {
	ID             string
	GroupID        string
	FunnelID       string
	VisibleColumns []string
}

type ManageViewProfilesUseCase struct {
	vpRepo domain.ViewProfileRepository
}

func NewManageViewProfilesUseCase(vpRepo domain.ViewProfileRepository) *ManageViewProfilesUseCase {
	return &ManageViewProfilesUseCase{vpRepo: vpRepo}
}

func (uc *ManageViewProfilesUseCase) SetViewProfile(ctx context.Context, input ViewProfileInput) error {
	existing, err := uc.vpRepo.FindByGroupAndFunnel(ctx, input.GroupID, input.FunnelID)
	if err != nil && err != domain.ErrViewProfileNotFound {
		return err
	}

	if existing != nil {
		existing.UpdateColumns(input.VisibleColumns)
		return uc.vpRepo.CreateOrUpdate(ctx, existing)
	}

	vp, err := domain.NewViewProfile(uuid.New().String(), input.GroupID, input.FunnelID, input.VisibleColumns)
	if err != nil {
		return err
	}
	return uc.vpRepo.CreateOrUpdate(ctx, vp)
}

func (uc *ManageViewProfilesUseCase) ListByGroup(ctx context.Context, groupID string) ([]ViewProfileOutput, error) {
	vps, err := uc.vpRepo.FindByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	output := make([]ViewProfileOutput, len(vps))
	for i, vp := range vps {
		output[i] = ViewProfileOutput{ID: vp.ID, GroupID: vp.GroupID, FunnelID: vp.FunnelID, VisibleColumns: vp.VisibleColumns}
	}
	return output, nil
}

// ResolveVisibleColumns returns the union of visible columns from all groups a user belongs to.
func (uc *ManageViewProfilesUseCase) ResolveVisibleColumns(ctx context.Context, groupIDs []string, funnelID string) ([]string, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	vps, err := uc.vpRepo.FindByGroupIDs(ctx, groupIDs, funnelID)
	if err != nil {
		return nil, err
	}
	if len(vps) == 0 {
		return nil, nil // no profiles = show all columns
	}

	// Union of all visible columns
	seen := make(map[string]bool)
	var result []string
	for _, vp := range vps {
		for _, col := range vp.VisibleColumns {
			if !seen[col] {
				seen[col] = true
				result = append(result, col)
			}
		}
	}
	return result, nil
}
```

- [ ] **Step 2: Implement GroupFunnel use case**

```go
// internal/permission/application/manage_group_funnels.go
package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

type GroupFunnelInput struct {
	GroupID   string
	FunnelID  string
	ColumnIDs []string
}

type GroupFunnelOutput struct {
	ID        string
	GroupID   string
	FunnelID  string
	ColumnIDs []string
}

type ManageGroupFunnelsUseCase struct {
	gfRepo domain.GroupFunnelRepository
}

func NewManageGroupFunnelsUseCase(gfRepo domain.GroupFunnelRepository) *ManageGroupFunnelsUseCase {
	return &ManageGroupFunnelsUseCase{gfRepo: gfRepo}
}

func (uc *ManageGroupFunnelsUseCase) SetGroupFunnel(ctx context.Context, input GroupFunnelInput) error {
	gf, err := domain.NewGroupFunnel(uuid.New().String(), input.GroupID, input.FunnelID, input.ColumnIDs)
	if err != nil {
		return err
	}
	return uc.gfRepo.CreateOrUpdate(ctx, gf)
}

func (uc *ManageGroupFunnelsUseCase) ListByGroup(ctx context.Context, groupID string) ([]GroupFunnelOutput, error) {
	gfs, err := uc.gfRepo.FindByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	output := make([]GroupFunnelOutput, len(gfs))
	for i, gf := range gfs {
		output[i] = GroupFunnelOutput{ID: gf.ID, GroupID: gf.GroupID, FunnelID: gf.FunnelID, ColumnIDs: gf.ColumnIDs}
	}
	return output, nil
}

func (uc *ManageGroupFunnelsUseCase) RemoveGroupFunnel(ctx context.Context, groupID, funnelID string) error {
	return uc.gfRepo.Delete(ctx, groupID, funnelID)
}
```

- [ ] **Step 3: Verify build and run all tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./internal/permission/... && go test ./internal/permission/... -v`
Expected: build ok, all tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/permission/application/manage_view_profiles.go internal/permission/application/manage_group_funnels.go
git commit -m "feat(permission): add view profile and group-funnel use cases"
```

---

## Task 13: HTTP Handlers

**Files:**
- Create: `internal/permission/interfaces/http/handler.go`
- Create: `internal/permission/interfaces/http/routes.go`

- [ ] **Step 1: Create handler with all endpoints**

```go
// internal/permission/interfaces/http/handler.go
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/permission/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

type Handler struct {
	createGroupUC   *application.CreateGroupUseCase
	updateGroupUC   *application.UpdateGroupUseCase
	listGroupsUC    *application.ListGroupsUseCase
	deleteGroupUC   *application.DeleteGroupUseCase
	manageMembersUC *application.ManageMembersUseCase
	managePermsUC   *application.ManagePermissionsUseCase
	manageVPUC      *application.ManageViewProfilesUseCase
	manageGFUC      *application.ManageGroupFunnelsUseCase
	log             *zap.Logger
}

func NewHandler(
	createGroupUC *application.CreateGroupUseCase,
	updateGroupUC *application.UpdateGroupUseCase,
	listGroupsUC *application.ListGroupsUseCase,
	deleteGroupUC *application.DeleteGroupUseCase,
	manageMembersUC *application.ManageMembersUseCase,
	managePermsUC *application.ManagePermissionsUseCase,
	manageVPUC *application.ManageViewProfilesUseCase,
	manageGFUC *application.ManageGroupFunnelsUseCase,
	log *zap.Logger,
) *Handler {
	return &Handler{
		createGroupUC: createGroupUC, updateGroupUC: updateGroupUC,
		listGroupsUC: listGroupsUC, deleteGroupUC: deleteGroupUC,
		manageMembersUC: manageMembersUC, managePermsUC: managePermsUC,
		manageVPUC: manageVPUC, manageGFUC: manageGFUC, log: log,
	}
}

// --- Groups ---

func (h *Handler) ListGroups(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groups, err := h.listGroupsUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list groups", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list groups"})
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (h *Handler) CreateGroup(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	var req struct {
		Name        string `json:"name" form:"name"`
		Description string `json:"description" form:"description"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	output, err := h.createGroupUC.Execute(c.Request.Context(), application.CreateGroupInput{
		TenantID: tenantID, Name: req.Name, Description: req.Description,
	})
	if err != nil {
		h.log.Error("failed to create group", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, output)
}

func (h *Handler) UpdateGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" form:"name"`
		Description string `json:"description" form:"description"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	output, err := h.updateGroupUC.Execute(c.Request.Context(), application.UpdateGroupInput{
		ID: c.Param("id"), Name: req.Name, Description: req.Description,
	})
	if err != nil {
		h.log.Error("failed to update group", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, output)
}

func (h *Handler) DeleteGroup(c *gin.Context) {
	if err := h.deleteGroupUC.Execute(c.Request.Context(), c.Param("id")); err != nil {
		h.log.Error("failed to delete group", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Members ---

func (h *Handler) AddMember(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	var req struct {
		UserID string `json:"user_id" form:"user_id"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.manageMembersUC.AddMember(c.Request.Context(), req.UserID, c.Param("id"), tenantID); err != nil {
		h.log.Error("failed to add member", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusCreated)
}

func (h *Handler) RemoveMember(c *gin.Context) {
	if err := h.manageMembersUC.RemoveMember(c.Request.Context(), c.Param("uid"), c.Param("id")); err != nil {
		h.log.Error("failed to remove member", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListMembers(c *gin.Context) {
	members, err := h.manageMembersUC.ListMembers(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.log.Error("failed to list members", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list members"})
		return
	}
	c.JSON(http.StatusOK, members)
}

// --- Permissions ---

func (h *Handler) GetGroupPermissions(c *gin.Context) {
	perms, err := h.managePermsUC.GetGroupPermissions(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.log.Error("failed to get group permissions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get permissions"})
		return
	}
	c.JSON(http.StatusOK, perms)
}

func (h *Handler) SetGroupPermissions(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	var req struct {
		Permissions []application.PermissionInput `json:"permissions" form:"permissions"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.managePermsUC.SetGroupPermissions(c.Request.Context(), tenantID, c.Param("id"), req.Permissions); err != nil {
		h.log.Error("failed to set group permissions", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) GetUserPermissions(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	perms, err := h.managePermsUC.GetUserPermissions(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		h.log.Error("failed to get user permissions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get permissions"})
		return
	}
	c.JSON(http.StatusOK, perms)
}

func (h *Handler) SetUserPermissions(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	var req struct {
		Permissions []application.PermissionInput `json:"permissions" form:"permissions"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.managePermsUC.SetUserPermissions(c.Request.Context(), tenantID, c.Param("id"), req.Permissions); err != nil {
		h.log.Error("failed to set user permissions", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// --- View Profiles ---

func (h *Handler) ListViewProfiles(c *gin.Context) {
	vps, err := h.manageVPUC.ListByGroup(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.log.Error("failed to list view profiles", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list view profiles"})
		return
	}
	c.JSON(http.StatusOK, vps)
}

func (h *Handler) SetViewProfile(c *gin.Context) {
	var req struct {
		VisibleColumns []string `json:"visible_columns" form:"visible_columns"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.manageVPUC.SetViewProfile(c.Request.Context(), application.ViewProfileInput{
		GroupID: c.Param("id"), FunnelID: c.Param("fid"), VisibleColumns: req.VisibleColumns,
	}); err != nil {
		h.log.Error("failed to set view profile", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// --- Group-Funnel Associations ---

func (h *Handler) ListGroupFunnels(c *gin.Context) {
	gfs, err := h.manageGFUC.ListByGroup(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.log.Error("failed to list group funnels", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list group funnels"})
		return
	}
	c.JSON(http.StatusOK, gfs)
}

func (h *Handler) SetGroupFunnels(c *gin.Context) {
	var req struct {
		Funnels []application.GroupFunnelInput `json:"funnels" form:"funnels"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	for _, gf := range req.Funnels {
		gf.GroupID = c.Param("id")
		if err := h.manageGFUC.SetGroupFunnel(c.Request.Context(), gf); err != nil {
			h.log.Error("failed to set group funnel", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.Status(http.StatusOK)
}
```

- [ ] **Step 2: Create route registration**

```go
// internal/permission/interfaces/http/routes.go
package http

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, tenantMw gin.HandlerFunc, requirePerm func(resource, action string) gin.HandlerFunc) {
	groups := router.Group("/tenant/groups")
	groups.Use(authMw, tenantMw)
	{
		groups.GET("", requirePerm("groups", "manage"), h.ListGroups)
		groups.POST("", requirePerm("groups", "manage"), h.CreateGroup)
		groups.GET("/:id", requirePerm("groups", "manage"), h.ListMembers)
		groups.PUT("/:id", requirePerm("groups", "manage"), h.UpdateGroup)
		groups.DELETE("/:id", requirePerm("groups", "manage"), h.DeleteGroup)

		// Members
		groups.POST("/:id/members", requirePerm("groups", "manage"), h.AddMember)
		groups.DELETE("/:id/members/:uid", requirePerm("groups", "manage"), h.RemoveMember)

		// Permissions
		groups.GET("/:id/permissions", requirePerm("groups", "manage"), h.GetGroupPermissions)
		groups.PUT("/:id/permissions", requirePerm("groups", "manage"), h.SetGroupPermissions)

		// View profiles
		groups.GET("/:id/view-profiles", requirePerm("funnels", "customize"), h.ListViewProfiles)
		groups.PUT("/:id/view-profiles/:fid", requirePerm("funnels", "customize"), h.SetViewProfile)

		// Group-funnel associations
		groups.GET("/:id/funnels", requirePerm("groups", "manage"), h.ListGroupFunnels)
		groups.PUT("/:id/funnels", requirePerm("groups", "manage"), h.SetGroupFunnels)
	}

	// User-level permissions
	users := router.Group("/tenant/users")
	users.Use(authMw, tenantMw)
	{
		users.GET("/:id/permissions", requirePerm("users", "manage"), h.GetUserPermissions)
		users.PUT("/:id/permissions", requirePerm("users", "manage"), h.SetUserPermissions)
	}
}
```

- [ ] **Step 3: Verify build**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./internal/permission/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/permission/interfaces/
git commit -m "feat(permission): add HTTP handlers and routes"
```

---

## Task 14: Module Wiring and Composition Root

**Files:**
- Create: `internal/permission/module.go`
- Create: `internal/permission/infrastructure/adapters.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Create auth adapters for OwnerChecker and AdminChecker**

```go
// internal/permission/infrastructure/adapters.go
package infrastructure

import (
	"context"

	"gorm.io/gorm"
)

// OwnerCheckerAdapter checks UserTenant table for ownership.
// For now, checks User.Role == "admin" as proxy until IsOwner field is added in Plan 2.
type OwnerCheckerAdapter struct {
	db *gorm.DB
}

func NewOwnerCheckerAdapter(db *gorm.DB) *OwnerCheckerAdapter {
	return &OwnerCheckerAdapter{db: db}
}

func (a *OwnerCheckerAdapter) IsOwner(ctx context.Context, userID, tenantID string) (bool, error) {
	// Plan 2 will add is_owner to user_tenants. For now, return false.
	// Owner bypass is handled by admin check below.
	return false, nil
}

// AdminCheckerAdapter checks the users table for admin role.
type AdminCheckerAdapter struct {
	db *gorm.DB
}

func NewAdminCheckerAdapter(db *gorm.DB) *AdminCheckerAdapter {
	return &AdminCheckerAdapter{db: db}
}

func (a *AdminCheckerAdapter) IsAdmin(ctx context.Context, userID string) (bool, error) {
	var count int64
	err := a.db.WithContext(ctx).Table("users").Where("id = ? AND role = ?", userID, "admin").Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
```

- [ ] **Step 2: Create permission module**

```go
// internal/permission/module.go
package permission

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/permission/application"
	"github.com/sasrgita/crm-juridico/internal/permission/infrastructure"
	permhttp "github.com/sasrgita/crm-juridico/internal/permission/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

type Module struct {
	handler    *permhttp.Handler
	resolverUC *application.ResolvePermissionUseCase
	vpUC       *application.ManageViewProfilesUseCase
	ugRepo     *infrastructure.GormUserGroupRepository
	gfRepo     *infrastructure.GormGroupFunnelRepository
}

func NewModule(db *gorm.DB, log *zap.Logger) *Module {
	groupRepo := infrastructure.NewGormPermissionGroupRepository(db)
	ugRepo := infrastructure.NewGormUserGroupRepository(db)
	permRepo := infrastructure.NewGormPermissionRepository(db)
	vpRepo := infrastructure.NewGormViewProfileRepository(db)
	gfRepo := infrastructure.NewGormGroupFunnelRepository(db)

	ownerChecker := infrastructure.NewOwnerCheckerAdapter(db)
	adminChecker := infrastructure.NewAdminCheckerAdapter(db)

	// Use cases
	resolverUC := application.NewResolvePermissionUseCase(permRepo, ugRepo, ownerChecker, adminChecker)
	createGroupUC := application.NewCreateGroupUseCase(groupRepo)
	updateGroupUC := application.NewUpdateGroupUseCase(groupRepo)
	listGroupsUC := application.NewListGroupsUseCase(groupRepo)
	deleteGroupUC := application.NewDeleteGroupUseCase(groupRepo)
	manageMembersUC := application.NewManageMembersUseCase(ugRepo, groupRepo)
	managePermsUC := application.NewManagePermissionsUseCase(permRepo)
	manageVPUC := application.NewManageViewProfilesUseCase(vpRepo)
	manageGFUC := application.NewManageGroupFunnelsUseCase(gfRepo)

	handler := permhttp.NewHandler(
		createGroupUC, updateGroupUC, listGroupsUC, deleteGroupUC,
		manageMembersUC, managePermsUC, manageVPUC, manageGFUC, log,
	)

	return &Module{
		handler: handler, resolverUC: resolverUC,
		vpUC: manageVPUC, ugRepo: ugRepo, gfRepo: gfRepo,
	}
}

func (m *Module) Name() string { return "permission" }

func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant, mw.RequirePermission)
}

// Resolver returns the permission resolver for use by the middleware.
func (m *Module) Resolver() *application.ResolvePermissionUseCase {
	return m.resolverUC
}

// ViewProfileUC returns the view profile use case for use by funnel module.
func (m *Module) ViewProfileUC() *application.ManageViewProfilesUseCase {
	return m.vpUC
}
```

- [ ] **Step 3: Update main.go — add permission module and EventBus wiring**

Add to `cmd/api/main.go` after existing module instantiation:

```go
// After whatsapp module creation, pass shared EventBus:
// Change: whatsappMod := whatsapp.NewModule(db, whatsmeowProvider, log)
// To:     whatsappMod := whatsapp.NewModule(db, whatsmeowProvider, sharedEventBus, log)

// Add before module creation:
sharedEventBus := events.NewMemoryEventBus()

// Add permission module:
permissionMod := permission.NewModule(db, log)

// Create the RequirePermission middleware:
requirePermMw := middleware.RequirePermission(permissionMod.Resolver())

// Update the Middlewares struct:
mw := module.Middlewares{
    Auth:              authMw,
    Tenant:            tenantMw,
    Admin:             adminMw,
    RequirePermission: requirePermMw,
}

// Add permissionMod to modules slice:
modules := []module.Module{tenantMod, specialistMod, documentMod, mcpMod, whatsappMod, funnelMod, productMod, aiMod, permissionMod}
```

Add imports:
```go
"github.com/sasrgita/crm-juridico/internal/permission"
"github.com/sasrgita/crm-juridico/internal/shared/events"
```

- [ ] **Step 4: Verify full build**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./...`
Expected: no errors

- [ ] **Step 5: Run all tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./... -count=1`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/permission/module.go internal/permission/infrastructure/adapters.go cmd/api/main.go
git commit -m "feat(permission): wire module into composition root with RequirePermission middleware"
```

---

## Task 15: Verify End-to-End with Docker

- [ ] **Step 1: Build and run containers**

Run: `cd /home/sasrgita/projects/crm_juridico && docker compose up --build -d`
Expected: containers start, migrations run (including new permission tables)

- [ ] **Step 2: Verify migrations applied**

Run: `docker compose exec api curl -s http://localhost:8080/health`
Expected: healthy response

- [ ] **Step 3: Run full test suite**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./... -count=1 -v`
Expected: all tests pass, including shared/events and permission module

- [ ] **Step 4: Stop containers**

Run: `docker compose down`

- [ ] **Step 5: Final commit if any cleanup needed**

```bash
git status
# If clean, no commit needed
```
