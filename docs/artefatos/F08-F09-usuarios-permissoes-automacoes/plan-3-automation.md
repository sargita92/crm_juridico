# Plan 3: Automation Module

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the automation module with 7 automation types, an event-driven engine, expiration ticker, and CRUD HTTP endpoints.

**Architecture:** New DDD module `internal/automation/` with domain entities (Automation, ExecutionLog, RateLimitCounter), executor pattern (one executor per automation type), an AutomationEngine that subscribes to EventBus lead events, and an ExpirationTicker goroutine. Sync executors run inline; async executors run in goroutines. Cross-module dependencies (MoveLeadUseCase, SendMessageUseCase, LeadNoteRepository, ConversationStateRepository, NotifyService, ProductDetector, FunnelProductRouter) are injected via interfaces defined in the automation domain.

**Tech Stack:** Go, Gin, GORM, MySQL, golang-migrate, testify, shared EventBus

**Spec:** `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/design-v1.md` (Módulo Automation section)

**Depends on:** Plan 1 (EventBus, Permission), Plan 2 (Notification, Funnel events)

---

## File Structure

```
internal/automation/
  domain/
    automation.go         # Automation entity + AutomationType enum
    execution_log.go      # ExecutionLog entity
    rate_limit.go         # RateLimitCounter entity
    repository.go         # Repository interfaces
    executor.go           # Executor interface + cross-module interfaces
    errors.go             # Sentinel errors
  application/
    engine.go             # AutomationEngine — subscribes to events, dispatches executors
    engine_test.go
    ticker.go             # ExpirationTicker — goroutine for time-based automations
    executors/
      move_funnel.go      # move_funnel executor
      auto_note.go        # auto_note executor
      switch_specialist.go # switch_specialist executor
      auto_message.go     # auto_message executor (async)
      detect_product.go   # detect_product executor
      expiration.go       # expiration executor (used by ticker)
    crud.go               # CRUD use cases (Create, Update, Delete, List, Toggle, GetLogs)
    crud_test.go
    mocks_test.go         # Shared test mocks
  infrastructure/
    models.go             # GORM models + mappers
    gorm_automation_repo.go
    gorm_execution_log_repo.go
    gorm_rate_limit_repo.go
  interfaces/http/
    handler.go            # HTTP handlers
    routes.go             # Route registration
  module.go               # Module wiring

migrations/
  000046_create_automations.up.sql
  000046_create_automations.down.sql
  000047_create_execution_logs.up.sql
  000047_create_execution_logs.down.sql
  000048_create_rate_limit_counters.up.sql
  000048_create_rate_limit_counters.down.sql
```

---

## Task 1: Migrations

**Files:** 6 migration files in `migrations/`

- [ ] **Step 1: Create migration files**

```sql
-- 000046_create_automations.up.sql
CREATE TABLE automations (
    id CHAR(36) NOT NULL PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    funnel_id CHAR(36) NOT NULL,
    column_id CHAR(36) NULL,
    type VARCHAR(30) NOT NULL,
    config JSON NOT NULL,
    active TINYINT(1) NOT NULL DEFAULT 1,
    priority INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (funnel_id) REFERENCES funnels(id) ON DELETE CASCADE,
    INDEX idx_automations_tenant (tenant_id),
    INDEX idx_automations_column (column_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 000046_create_automations.down.sql
DROP TABLE IF EXISTS automations;

-- 000047_create_execution_logs.up.sql
CREATE TABLE execution_logs (
    id CHAR(36) NOT NULL PRIMARY KEY,
    automation_id CHAR(36) NOT NULL,
    lead_id CHAR(36) NOT NULL,
    tenant_id CHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL,
    error_message TEXT NULL,
    executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (automation_id) REFERENCES automations(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    INDEX idx_exec_logs_automation (automation_id),
    INDEX idx_exec_logs_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 000047_create_execution_logs.down.sql
DROP TABLE IF EXISTS execution_logs;

-- 000048_create_rate_limit_counters.up.sql
CREATE TABLE rate_limit_counters (
    id CHAR(36) NOT NULL PRIMARY KEY,
    tenant_id CHAR(36) NOT NULL,
    specialist_id CHAR(36) NULL,
    period_start DATETIME NOT NULL,
    message_count INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE KEY uk_rate_limit (tenant_id, specialist_id, period_start)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 000048_create_rate_limit_counters.down.sql
DROP TABLE IF EXISTS rate_limit_counters;
```

- [ ] **Step 2: Commit**

```bash
git add migrations/000046_* migrations/000047_* migrations/000048_*
git commit -m "feat(automation): add database migrations 000046-000048"
```

---

## Task 2: Domain — Entities, Errors, Interfaces

**Files:**
- Create: `internal/automation/domain/errors.go`
- Create: `internal/automation/domain/automation.go`
- Create: `internal/automation/domain/execution_log.go`
- Create: `internal/automation/domain/rate_limit.go`
- Create: `internal/automation/domain/repository.go`
- Create: `internal/automation/domain/executor.go`

- [ ] **Step 1: Create errors**

```go
// internal/automation/domain/errors.go
package domain

import "errors"

var (
	ErrAutomationNotFound  = errors.New("automation not found")
	ErrTenantIDRequired    = errors.New("tenant ID is required")
	ErrFunnelIDRequired    = errors.New("funnel ID is required")
	ErrInvalidType         = errors.New("invalid automation type")
	ErrInvalidConfig       = errors.New("invalid automation config")
	ErrRateLimitNotFound   = errors.New("rate limit counter not found")
	ErrExecutionLogNotFound = errors.New("execution log not found")
)
```

- [ ] **Step 2: Create Automation entity**

```go
// internal/automation/domain/automation.go
package domain

import (
	"encoding/json"
	"time"
)

type AutomationType string

const (
	TypeExpiration       AutomationType = "expiration"
	TypeMoveFunnel       AutomationType = "move_funnel"
	TypeAutoMessage      AutomationType = "auto_message"
	TypeAutoNote         AutomationType = "auto_note"
	TypeSwitchSpecialist AutomationType = "switch_specialist"
	TypeRateLimit        AutomationType = "rate_limit"
	TypeDetectProduct    AutomationType = "detect_product"
)

var validTypes = map[AutomationType]bool{
	TypeExpiration: true, TypeMoveFunnel: true, TypeAutoMessage: true,
	TypeAutoNote: true, TypeSwitchSpecialist: true, TypeRateLimit: true,
	TypeDetectProduct: true,
}

type Automation struct {
	ID        string
	TenantID  string
	FunnelID  string
	ColumnID  string // empty for rate_limit
	Type      AutomationType
	Config    map[string]interface{}
	Active    bool
	Priority  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewAutomation(id, tenantID, funnelID, columnID string, automationType AutomationType, config map[string]interface{}, priority int) (*Automation, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if funnelID == "" {
		return nil, ErrFunnelIDRequired
	}
	if !validTypes[automationType] {
		return nil, ErrInvalidType
	}
	if config == nil {
		config = make(map[string]interface{})
	}
	now := time.Now()
	return &Automation{
		ID: id, TenantID: tenantID, FunnelID: funnelID, ColumnID: columnID,
		Type: automationType, Config: config, Active: true, Priority: priority,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (a *Automation) Update(columnID string, config map[string]interface{}, priority int) {
	a.ColumnID = columnID
	a.Config = config
	a.Priority = priority
	a.UpdatedAt = time.Now()
}

func (a *Automation) Activate()   { a.Active = true; a.UpdatedAt = time.Now() }
func (a *Automation) Deactivate() { a.Active = false; a.UpdatedAt = time.Now() }

func (a *Automation) ConfigJSON() string {
	data, _ := json.Marshal(a.Config)
	return string(data)
}

// ConfigString returns a config value as string, empty if not found.
func (a *Automation) ConfigString(key string) string {
	v, ok := a.Config[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// ConfigFloat returns a config value as float64, 0 if not found.
func (a *Automation) ConfigFloat(key string) float64 {
	v, ok := a.Config[key]
	if !ok {
		return 0
	}
	f, _ := v.(float64)
	return f
}

// ConfigBool returns a config value as bool, false if not found.
func (a *Automation) ConfigBool(key string) bool {
	v, ok := a.Config[key]
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}
```

- [ ] **Step 3: Create ExecutionLog entity**

```go
// internal/automation/domain/execution_log.go
package domain

import "time"

type ExecutionStatus string

const (
	StatusSuccess ExecutionStatus = "success"
	StatusError   ExecutionStatus = "error"
)

type ExecutionLog struct {
	ID           string
	AutomationID string
	LeadID       string
	TenantID     string
	Status       ExecutionStatus
	ErrorMessage string
	ExecutedAt   time.Time
}

func NewExecutionLog(id, automationID, leadID, tenantID string, status ExecutionStatus, errMsg string) *ExecutionLog {
	return &ExecutionLog{
		ID: id, AutomationID: automationID, LeadID: leadID, TenantID: tenantID,
		Status: status, ErrorMessage: errMsg, ExecutedAt: time.Now(),
	}
}
```

- [ ] **Step 4: Create RateLimitCounter entity**

```go
// internal/automation/domain/rate_limit.go
package domain

import "time"

type RateLimitCounter struct {
	ID           string
	TenantID     string
	SpecialistID string
	PeriodStart  time.Time
	MessageCount int
	CreatedAt    time.Time
}

func NewRateLimitCounter(id, tenantID, specialistID string) *RateLimitCounter {
	now := time.Now()
	return &RateLimitCounter{
		ID: id, TenantID: tenantID, SpecialistID: specialistID,
		PeriodStart: startOfDay(now), MessageCount: 0, CreatedAt: now,
	}
}

func (c *RateLimitCounter) Increment() { c.MessageCount++ }

func (c *RateLimitCounter) IsExceeded(maxMessages int) bool {
	return c.MessageCount >= maxMessages
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
```

- [ ] **Step 5: Create repository interfaces**

```go
// internal/automation/domain/repository.go
package domain

import "context"

type AutomationRepository interface {
	Create(ctx context.Context, a *Automation) error
	FindByID(ctx context.Context, id string) (*Automation, error)
	Update(ctx context.Context, a *Automation) error
	Delete(ctx context.Context, id string) error
	FindByTenantAndColumn(ctx context.Context, tenantID, columnID string) ([]Automation, error)
	FindByFunnelID(ctx context.Context, tenantID, funnelID string) ([]Automation, error)
	FindActiveByType(ctx context.Context, automationType AutomationType) ([]Automation, error)
}

type ExecutionLogRepository interface {
	Create(ctx context.Context, log *ExecutionLog) error
	FindByAutomationID(ctx context.Context, automationID string, limit, offset int) ([]ExecutionLog, error)
}

type RateLimitRepository interface {
	FindOrCreate(ctx context.Context, tenantID, specialistID string) (*RateLimitCounter, error)
	Increment(ctx context.Context, id string) error
}
```

- [ ] **Step 6: Create Executor interface and cross-module interfaces**

```go
// internal/automation/domain/executor.go
package domain

import "context"

// Executor executes a single automation for a given lead.
type Executor interface {
	Execute(ctx context.Context, automation *Automation, leadID, tenantID string) error
}

// --- Cross-module interfaces (implemented via adapters) ---

// LeadMover moves a lead to a different column/funnel.
type LeadMover interface {
	MoveLead(ctx context.Context, tenantID, leadID, columnID, funnelID string) error
}

// LeadFinder retrieves lead details needed by executors.
type LeadFinder interface {
	FindByID(ctx context.Context, leadID string) (*LeadInfo, error)
	FindExpiredInColumn(ctx context.Context, columnID string, maxAge time.Duration) ([]string, error)
}

type LeadInfo struct {
	ID                string
	TenantID          string
	FunnelID          string
	ColumnID          string
	ContactID         string
	ConversationID    string
	ProductID         string
	ResponsibleUserID string
	ColumnEnteredAt   time.Time
}

// NoteSaver creates a lead note.
type NoteSaver interface {
	SaveNote(ctx context.Context, leadID, tenantID, content, createdBy string) error
}

// MessageSender sends a WhatsApp message to a contact.
type MessageSender interface {
	SendToContact(ctx context.Context, tenantID, contactID, content string) error
}

// SpecialistSwitcher updates the conversation state specialist.
type SpecialistSwitcher interface {
	SwitchSpecialist(ctx context.Context, conversationID, specialistID string) error
}

// ProductRouter finds the funnel/column for a product.
type ProductRouter interface {
	FindFunnelForProduct(ctx context.Context, productID string) (funnelID, columnID string, err error)
}

// SpecialistForProduct finds the specialist linked to a product.
type SpecialistForProduct interface {
	FindByProductID(ctx context.Context, productID string) (specialistID string, err error)
}

// Notifier sends a notification to a user.
type Notifier interface {
	Notify(ctx context.Context, userID, tenantID, title, body string, metadata map[string]string) error
}

// LeadDeleter soft-deletes a lead.
type LeadDeleter interface {
	SoftDelete(ctx context.Context, leadID string) error
}

// LostColumnFinder finds the "lost" column for a funnel.
type LostColumnFinder interface {
	FindLostColumn(ctx context.Context, funnelID string) (columnID string, err error)
}
```

- [ ] **Step 7: Verify build**

Run: `go build ./internal/automation/...`

- [ ] **Step 8: Commit**

```bash
git add internal/automation/domain/
git commit -m "feat(automation): add domain entities, errors, and interfaces"
```

---

## Task 3: Infrastructure — GORM Repositories

**Files:**
- Create: `internal/automation/infrastructure/models.go`
- Create: `internal/automation/infrastructure/gorm_automation_repo.go`
- Create: `internal/automation/infrastructure/gorm_execution_log_repo.go`
- Create: `internal/automation/infrastructure/gorm_rate_limit_repo.go`

- [ ] **Step 1: Create GORM models with mappers**

Follow the pattern from `internal/permission/infrastructure/models.go`:
- `automationModel` — Config as JSON string, ColumnID as *string
- `executionLogModel` — ErrorMessage as *string
- `rateLimitCounterModel` — SpecialistID as *string
- Mapper functions for all three entities

- [ ] **Step 2: Create AutomationRepository**

- Create, FindByID (ErrAutomationNotFound), Update, Delete
- FindByTenantAndColumn: `WHERE tenant_id=? AND column_id=? AND active=1 ORDER BY priority ASC`
- FindByFunnelID: `WHERE tenant_id=? AND funnel_id=? ORDER BY priority ASC`
- FindActiveByType: `WHERE type=? AND active=1`

- [ ] **Step 3: Create ExecutionLogRepository**

- Create, FindByAutomationID (with limit/offset, order by executed_at DESC)

- [ ] **Step 4: Create RateLimitRepository**

- FindOrCreate: find by (tenantID, specialistID, startOfDay(now)). If not found, create new counter.
- Increment: `UPDATE SET message_count = message_count + 1 WHERE id = ?`

- [ ] **Step 5: Verify build**: `go build ./internal/automation/...`
- [ ] **Step 6: Commit**

---

## Task 4: Executors — Sync Types

**Files:**
- Create: `internal/automation/application/executors/move_funnel.go`
- Create: `internal/automation/application/executors/auto_note.go`
- Create: `internal/automation/application/executors/switch_specialist.go`
- Create: `internal/automation/application/executors/detect_product.go`

- [ ] **Step 1: Create move_funnel executor**

```go
// internal/automation/application/executors/move_funnel.go
package executors

import (
	"context"
	"github.com/sasrgita/crm-juridico/internal/automation/domain"
)

type MoveFunnelExecutor struct {
	leadMover domain.LeadMover
}

func NewMoveFunnelExecutor(leadMover domain.LeadMover) *MoveFunnelExecutor {
	return &MoveFunnelExecutor{leadMover: leadMover}
}

func (e *MoveFunnelExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error {
	targetFunnelID := a.ConfigString("target_funnel_id")
	targetColumnID := a.ConfigString("target_column_id")
	return e.leadMover.MoveLead(ctx, tenantID, leadID, targetColumnID, targetFunnelID)
}
```

- [ ] **Step 2: Create auto_note executor**

```go
package executors

type AutoNoteExecutor struct {
	noteSaver domain.NoteSaver
}

func NewAutoNoteExecutor(noteSaver domain.NoteSaver) *AutoNoteExecutor {
	return &AutoNoteExecutor{noteSaver: noteSaver}
}

func (e *AutoNoteExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error {
	template := a.ConfigString("template")
	if template == "" {
		template = "Automation executed"
	}
	return e.noteSaver.SaveNote(ctx, leadID, tenantID, template, "system")
}
```

- [ ] **Step 3: Create switch_specialist executor**

```go
package executors

type SwitchSpecialistExecutor struct {
	switcher   domain.SpecialistSwitcher
	leadFinder domain.LeadFinder
}

func (e *SwitchSpecialistExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error {
	specialistID := a.ConfigString("specialist_id")
	lead, err := e.leadFinder.FindByID(ctx, leadID)
	if err != nil { return err }
	return e.switcher.SwitchSpecialist(ctx, lead.ConversationID, specialistID)
}
```

- [ ] **Step 4: Create detect_product executor**

```go
package executors

type DetectProductExecutor struct {
	leadFinder  domain.LeadFinder
	router      domain.ProductRouter
	leadMover   domain.LeadMover
	switcher    domain.SpecialistSwitcher
	specFinder  domain.SpecialistForProduct
}

func (e *DetectProductExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error {
	lead, err := e.leadFinder.FindByID(ctx, leadID)
	if err != nil { return err }
	if lead.ProductID == "" { return nil } // no product detected, skip

	funnelID, columnID, err := e.router.FindFunnelForProduct(ctx, lead.ProductID)
	if err != nil { return err }
	if funnelID == "" { return nil } // no funnel configured for product

	if err := e.leadMover.MoveLead(ctx, tenantID, leadID, columnID, funnelID); err != nil {
		return err
	}

	if a.ConfigBool("switch_specialist") {
		specID, err := e.specFinder.FindByProductID(ctx, lead.ProductID)
		if err != nil || specID == "" { return err }
		return e.switcher.SwitchSpecialist(ctx, lead.ConversationID, specID)
	}
	return nil
}
```

- [ ] **Step 5: Verify build**: `go build ./internal/automation/...`
- [ ] **Step 6: Commit**

---

## Task 5: Executors — Async + Expiration

**Files:**
- Create: `internal/automation/application/executors/auto_message.go`
- Create: `internal/automation/application/executors/expiration.go`

- [ ] **Step 1: Create auto_message executor (async — caller wraps in goroutine)**

```go
package executors

type AutoMessageExecutor struct {
	sender     domain.MessageSender
	leadFinder domain.LeadFinder
}

func (e *AutoMessageExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error {
	lead, err := e.leadFinder.FindByID(ctx, leadID)
	if err != nil { return err }
	template := a.ConfigString("template")
	if template == "" { return nil }
	// Simple template — future: add variable substitution
	return e.sender.SendToContact(ctx, tenantID, lead.ContactID, template)
}
```

- [ ] **Step 2: Create expiration executor (used by ticker)**

```go
package executors

type ExpirationExecutor struct {
	leadFinder      domain.LeadFinder
	leadMover       domain.LeadMover
	leadDeleter     domain.LeadDeleter
	lostColFinder   domain.LostColumnFinder
}

func (e *ExpirationExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error {
	action := a.ConfigString("action")
	lead, err := e.leadFinder.FindByID(ctx, leadID)
	if err != nil { return err }

	switch action {
	case "archive":
		lostColID, err := e.lostColFinder.FindLostColumn(ctx, lead.FunnelID)
		if err != nil { return err }
		return e.leadMover.MoveLead(ctx, tenantID, leadID, lostColID, "")
	case "delete":
		return e.leadDeleter.SoftDelete(ctx, leadID)
	default:
		return domain.ErrInvalidConfig
	}
}

// FindExpiredLeads returns lead IDs that have been in the column longer than duration_hours.
func (e *ExpirationExecutor) FindExpiredLeads(ctx context.Context, a *domain.Automation) ([]string, error) {
	hours := a.ConfigFloat("duration_hours")
	if hours <= 0 { return nil, nil }
	maxAge := time.Duration(hours) * time.Hour
	return e.leadFinder.FindExpiredInColumn(ctx, a.ColumnID, maxAge)
}
```

- [ ] **Step 3: Verify build**: `go build ./internal/automation/...`
- [ ] **Step 4: Commit**

---

## Task 6: AutomationEngine

**Files:**
- Create: `internal/automation/application/engine.go`
- Create: `internal/automation/application/engine_test.go`
- Create: `internal/automation/application/mocks_test.go`

- [ ] **Step 1: Create mocks**

In-memory implementations of AutomationRepository, ExecutionLogRepository, and a mock Executor for tests.

- [ ] **Step 2: Write failing test for engine**

```go
func TestEngine_OnLeadEvent_ExecutesAutomations(t *testing.T) {
	autoRepo := newMockAutoRepo()
	logRepo := newMockLogRepo()
	executor := &mockExecutor{}

	// Create automation for column-1
	auto, _ := domain.NewAutomation("auto-1", "tenant-1", "funnel-1", "column-1", domain.TypeAutoNote, map[string]interface{}{"template": "test"}, 0)
	autoRepo.Create(context.Background(), auto)

	engine := NewAutomationEngine(autoRepo, logRepo)
	engine.RegisterExecutor(domain.TypeAutoNote, executor)

	err := engine.OnLeadEvent(context.Background(), "tenant-1", "lead-1", "column-1")
	require.NoError(t, err)
	assert.Equal(t, 1, executor.callCount)
	assert.Len(t, logRepo.logs, 1)
	assert.Equal(t, domain.StatusSuccess, logRepo.logs[0].Status)
}
```

- [ ] **Step 3: Implement AutomationEngine**

```go
// internal/automation/application/engine.go
package application

type AutomationEngine struct {
	autoRepo  domain.AutomationRepository
	logRepo   domain.ExecutionLogRepository
	executors map[domain.AutomationType]domain.Executor
}

func NewAutomationEngine(autoRepo domain.AutomationRepository, logRepo domain.ExecutionLogRepository) *AutomationEngine {
	return &AutomationEngine{
		autoRepo: autoRepo, logRepo: logRepo,
		executors: make(map[domain.AutomationType]domain.Executor),
	}
}

func (e *AutomationEngine) RegisterExecutor(t domain.AutomationType, exec domain.Executor) {
	e.executors[t] = exec
}

// AsyncTypes defines which automation types run in goroutines.
var AsyncTypes = map[domain.AutomationType]bool{
	domain.TypeAutoMessage: true,
}

func (e *AutomationEngine) OnLeadEvent(ctx context.Context, tenantID, leadID, columnID string) error {
	automations, err := e.autoRepo.FindByTenantAndColumn(ctx, tenantID, columnID)
	if err != nil { return err }

	for _, auto := range automations {
		exec, ok := e.executors[auto.Type]
		if !ok { continue }

		if AsyncTypes[auto.Type] {
			go e.executeAndLog(context.Background(), exec, &auto, leadID, tenantID)
		} else {
			e.executeAndLog(ctx, exec, &auto, leadID, tenantID)
		}
	}
	return nil
}

func (e *AutomationEngine) executeAndLog(ctx context.Context, exec domain.Executor, auto *domain.Automation, leadID, tenantID string) {
	err := exec.Execute(ctx, auto, leadID, tenantID)
	status := domain.StatusSuccess
	errMsg := ""
	if err != nil {
		status = domain.StatusError
		errMsg = err.Error()
	}
	log := domain.NewExecutionLog(uuid.New().String(), auto.ID, leadID, tenantID, status, errMsg)
	_ = e.logRepo.Create(ctx, log)
}
```

- [ ] **Step 4: Run tests**: `go test ./internal/automation/application/ -v -run TestEngine`
- [ ] **Step 5: Commit**

---

## Task 7: ExpirationTicker

**Files:**
- Create: `internal/automation/application/ticker.go`

- [ ] **Step 1: Implement ExpirationTicker**

```go
// internal/automation/application/ticker.go
package application

type ExpirationTicker struct {
	autoRepo    domain.AutomationRepository
	logRepo     domain.ExecutionLogRepository
	expExecutor *executors.ExpirationExecutor
	interval    time.Duration
	stop        chan struct{}
}

func NewExpirationTicker(autoRepo domain.AutomationRepository, logRepo domain.ExecutionLogRepository, expExecutor *executors.ExpirationExecutor) *ExpirationTicker {
	return &ExpirationTicker{
		autoRepo: autoRepo, logRepo: logRepo, expExecutor: expExecutor,
		interval: 5 * time.Minute, stop: make(chan struct{}),
	}
}

func (t *ExpirationTicker) Start() {
	go func() {
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				t.tick()
			case <-t.stop:
				return
			}
		}
	}()
}

func (t *ExpirationTicker) Stop() { close(t.stop) }

func (t *ExpirationTicker) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	automations, err := t.autoRepo.FindActiveByType(ctx, domain.TypeExpiration)
	if err != nil { return }

	for _, auto := range automations {
		leadIDs, err := t.expExecutor.FindExpiredLeads(ctx, &auto)
		if err != nil { continue }

		for _, leadID := range leadIDs {
			execErr := t.expExecutor.Execute(ctx, &auto, leadID, auto.TenantID)
			status := domain.StatusSuccess
			errMsg := ""
			if execErr != nil { status = domain.StatusError; errMsg = execErr.Error() }
			log := domain.NewExecutionLog(uuid.New().String(), auto.ID, leadID, auto.TenantID, status, errMsg)
			_ = t.logRepo.Create(ctx, log)
		}
	}
}
```

- [ ] **Step 2: Verify build**: `go build ./internal/automation/...`
- [ ] **Step 3: Commit**

---

## Task 8: CRUD Use Cases

**Files:**
- Create: `internal/automation/application/crud.go`
- Create: `internal/automation/application/crud_test.go`

- [ ] **Step 1: Write failing tests**

Tests: CreateAutomation_Success, CreateAutomation_InvalidType, ListByFunnel, ToggleAutomation, DeleteAutomation, GetLogs

- [ ] **Step 2: Implement CRUD**

```go
type CreateAutomationInput struct {
	TenantID string
	FunnelID string
	ColumnID string
	Type     string
	Config   map[string]interface{}
	Priority int
}

type AutomationOutput struct {
	ID, TenantID, FunnelID, ColumnID string
	Type string
	Config map[string]interface{}
	Active bool
	Priority int
}

type CRUDUseCase struct { autoRepo domain.AutomationRepository; logRepo domain.ExecutionLogRepository }

func (uc *CRUDUseCase) Create(ctx, input) → (AutomationOutput, error)
func (uc *CRUDUseCase) Update(ctx, id, input) → (AutomationOutput, error)
func (uc *CRUDUseCase) Delete(ctx, id) → error
func (uc *CRUDUseCase) ListByFunnel(ctx, tenantID, funnelID) → ([]AutomationOutput, error)
func (uc *CRUDUseCase) Toggle(ctx, id) → error
func (uc *CRUDUseCase) GetLogs(ctx, automationID, limit, offset) → ([]LogOutput, error)
```

- [ ] **Step 3: Run tests**: `go test ./internal/automation/application/ -v`
- [ ] **Step 4: Commit**

---

## Task 9: HTTP Handlers + Routes

**Files:**
- Create: `internal/automation/interfaces/http/handler.go`
- Create: `internal/automation/interfaces/http/routes.go`

- [ ] **Step 1: Create handler with all methods**

Handlers: ListByFunnel (GET /funnels/:id/automations), Create (POST), GetDetail (GET /automations/:id), Update (PUT), Delete (DELETE), Toggle (POST /toggle), GetLogs (GET /logs)

All endpoints require `automations:manage` permission.

- [ ] **Step 2: Create routes**

```go
func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, tenantMw gin.HandlerFunc, requirePerm func(string, string) gin.HandlerFunc) {
	funnels := router.Group("/tenant/leads/funnels")
	funnels.Use(authMw, tenantMw)
	funnels.GET("/:id/automations", requirePerm("automations", "manage"), h.ListByFunnel)
	funnels.POST("/:id/automations", requirePerm("automations", "manage"), h.Create)

	autos := router.Group("/tenant/leads/automations")
	autos.Use(authMw, tenantMw)
	autos.GET("/:id", requirePerm("automations", "manage"), h.GetDetail)
	autos.PUT("/:id", requirePerm("automations", "manage"), h.Update)
	autos.DELETE("/:id", requirePerm("automations", "manage"), h.Delete)
	autos.POST("/:id/toggle", requirePerm("automations", "manage"), h.Toggle)
	autos.GET("/:id/logs", requirePerm("automations", "manage"), h.GetLogs)
}
```

- [ ] **Step 3: Verify build**: `go build ./internal/automation/...`
- [ ] **Step 4: Commit**

---

## Task 10: Module Wiring + EventBus Subscription + main.go

**Files:**
- Create: `internal/automation/module.go`
- Create: `internal/automation/infrastructure/adapters.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Create adapters for cross-module dependencies**

Adapters wrap funnel, whatsapp, ai, product, notification modules to implement the domain interfaces (LeadMover, LeadFinder, NoteSaver, MessageSender, SpecialistSwitcher, etc.).

- [ ] **Step 2: Create module.go**

```go
func NewModule(db *gorm.DB, eventBus events.EventBus, deps ModuleDeps, log *zap.Logger) *Module {
	// Create repos, executors, engine, ticker, crud, handler
	// Subscribe to EventBus for lead-created and lead-moved events
	// Start ExpirationTicker
}

type ModuleDeps struct {
	LeadMover           domain.LeadMover
	LeadFinder          domain.LeadFinder
	NoteSaver           domain.NoteSaver
	MessageSender       domain.MessageSender
	SpecialistSwitcher  domain.SpecialistSwitcher
	ProductRouter       domain.ProductRouter
	SpecialistForProduct domain.SpecialistForProduct
	Notifier            domain.Notifier
	LeadDeleter         domain.LeadDeleter
	LostColumnFinder    domain.LostColumnFinder
}
```

The module subscribes to EventBus in a goroutine:
```go
go func() {
	ch, cleanup := eventBus.Subscribe("") // subscribe to all tenants
	defer cleanup()
	for event := range ch {
		if event.Type == events.EventLeadCreated || event.Type == events.EventLeadMoved {
			payload, _ := event.Payload.(map[string]string)
			engine.OnLeadEvent(context.Background(), event.TenantID, payload["lead_id"], payload["column_id"])
		}
	}
}()
```

Note: The current EventBus is tenant-scoped. For the automation engine that needs ALL tenant events, we need to either: subscribe to each tenant, or add a global subscribe. The simplest approach: subscribe with empty tenantID and modify MemoryEventBus.Publish to also send to global subscribers. OR — have the automation module subscribe per-tenant as tenants are discovered. For now, the engine will be called directly from the event bus subscription in the module, subscribing to specific tenant IDs.

Actually, looking at the current MemoryEventBus, it's tenant-scoped only. The simplest fix: have the automation module get called by the engine directly when events are published — we'll add a `SubscribeAll` method to EventBus, or have the module start subscriptions per-tenant.

Simpler approach: **Don't subscribe via EventBus**. Instead, have the funnel module call the automation engine directly after publishing events (via an interface). This avoids the tenant-scoping issue entirely and is more reliable.

Add to funnel domain:
```go
type AutomationTrigger interface {
    OnLeadEvent(ctx context.Context, tenantID, leadID, columnID string) error
}
```

Funnel's CreateLeadUseCase and MoveLeadUseCase call this after the event publish.

- [ ] **Step 3: Update funnel module to accept AutomationTrigger**
- [ ] **Step 4: Update main.go**

Wire automation module with all adapters and pass to funnel module.

- [ ] **Step 5: Run full build**: `go build ./...`
- [ ] **Step 6: Run all tests**: `go test ./internal/automation/... -v`
- [ ] **Step 7: Commit**

---

## Task 11: Verification + Coverage

- [ ] **Step 1: Run full test suite**: `go test ./... -count=1`
- [ ] **Step 2: Check coverage**: `go test ./internal/automation/application/ -cover` (>= 80%)
- [ ] **Step 3: Run refresh.sh and verify app starts**: `./scripts/refresh.sh`
- [ ] **Step 4: Health check**: `curl http://localhost:8533/health`
- [ ] **Step 5: Final commit if cleanup needed**
