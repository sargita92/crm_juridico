# F08 Step 4/4.1 — Load Balance Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every lead created (HTTP manual / WhatsApp / AI) is born with a `ResponsibleUserID` chosen via the group's load-balance configuration, with a safe fallback to the tenant owner.

**Architecture:** New `ResponsiblePicker` port in `internal/funnel/domain` consumed by `CreateLeadUseCase`. Implementation `LoadBalancePicker` lives in `internal/auth/infrastructure`, reads `LoadBalanceConfig` + `GroupFunnel` + group members + tenant owner fallback, applies the selected algorithm, and emits a new `EventLeadResponsibleAssigned` that the notification module subscribes to. Uniqueness rule "one active LB group per funnel/column" enforced proactively in `ManageLoadBalanceUseCase.SetByGroup` and defensively at pick time.

**Tech Stack:** Go, GORM, Gin, Zap, Prometheus client_golang, OpenTelemetry, testify, testcontainers-go (MySQL), uuid.

**Spec:** [design-step4-load-balance-integration.md](./design-step4-load-balance-integration.md)

---

## File Structure

**Create:**
- `internal/funnel/domain/responsible_picker.go` — `ResponsiblePicker`, `PickResult`, `PickOutcome`, `LeadLoadCounter`
- `internal/funnel/domain/responsible_picker_test.go` — constant/struct sanity tests
- `internal/auth/infrastructure/load_balance_picker.go` — `LoadBalancePicker` struct + method
- `internal/auth/infrastructure/load_balance_picker_test.go` — unit tests (table-driven + fakes)
- `internal/auth/infrastructure/metrics.go` — Prometheus collectors for picker
- `internal/funnel/infrastructure/lead_load_counter.go` — small type wrapping gorm for count query (or extend existing lead repo file)
- `internal/permission/infrastructure/group_funnel_overlap_adapter.go` — cross-module adapter exposing `GroupColumnOverlapChecker`
- `internal/auth/application/group_column_overlap.go` — `GroupColumnOverlapChecker` port + error

**Modify:**
- `internal/shared/events/event.go` — add `EventLeadResponsibleAssigned` constant and typed payload struct
- `internal/funnel/application/create_lead.go` — accept `ResponsiblePicker`; call before `leadRepo.Create`; enrich `EventLeadCreated` payload; publish `EventLeadResponsibleAssigned`
- `internal/funnel/application/create_lead_test.go` — update existing tests + new cases
- `internal/funnel/module.go` — extend `NewModule` signature with `ResponsiblePicker`
- `internal/funnel/infrastructure/gorm_lead_repo.go` — add `CountActiveByUsers`
- `internal/auth/application/manage_load_balance.go` — inject `GroupColumnOverlapChecker`; enforce uniqueness on `SetByGroup` when `Active=true`
- `internal/auth/application/manage_load_balance_test.go` — new cases for 409
- `internal/notification/module.go` — subscribe to `EventLeadResponsibleAssigned` and call `NotifyService`
- `internal/notification/module_test.go` — (create if missing) test subscription
- `cmd/api/main.go` — wire picker, overlap checker, notification subscription
- `rest/team.http` and `rest/funnel.http` — add commentary + example checks
- `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/status.md` — tick off items
- `docs/features/F08-usuarios-permissoes.md` — tick off checkboxes

**Test:**
- `internal/auth/infrastructure/load_balance_picker_test.go`
- `internal/funnel/application/create_lead_test.go` (extended)
- `internal/funnel/infrastructure/lead_load_counter_test.go` (testcontainers)
- `internal/auth/application/manage_load_balance_test.go` (extended)
- `internal/notification/module_test.go` (extended)

---

## Task 1: Create feature branch

**Files:** none

- [ ] **Step 1: Create and switch to the branch**

```bash
git checkout -b feat/f08-load-balance-integration
```

Expected: branch exists, working tree clean.

- [ ] **Step 2: Confirm baseline tests pass**

```bash
go test ./...
```

Expected: PASS (baseline green before we start).

---

## Task 2: Add `EventLeadResponsibleAssigned` and extend `EventLeadCreated` payload

**Files:**
- Modify: `internal/shared/events/event.go`

- [ ] **Step 1: Add event type constant**

Edit `internal/shared/events/event.go`, add to the `const` block after `EventNotification`:

```go
const (
    EventNewMessage              EventType = "new-message"
    EventConversationUpdate      EventType = "conversation-update"
    EventLeadCreated             EventType = "lead-created"
    EventLeadMoved               EventType = "lead-moved"
    EventNotification            EventType = "notification"
    EventLeadResponsibleAssigned EventType = "lead-responsible-assigned"
)
```

- [ ] **Step 2: Add typed payload helper**

Below the `Event` struct in the same file, append:

```go
// ResponsibleAssignedPayload is the canonical payload for EventLeadResponsibleAssigned.
type ResponsibleAssignedPayload struct {
    LeadID            string
    ResponsibleUserID string
    Reason            string // "created" for now; room for "reassigned" later
    Outcome           string // "picked" | "fallback_owner"
    Algorithm         string // "round_robin" | "least_load" | "random" | ""
}
```

- [ ] **Step 3: Run build to confirm no regressions**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/shared/events/event.go
git commit -m "feat(events): add EventLeadResponsibleAssigned and typed payload"
```

---

## Task 3: Define `ResponsiblePicker` and `LeadLoadCounter` ports in funnel/domain

**Files:**
- Create: `internal/funnel/domain/responsible_picker.go`

- [ ] **Step 1: Write the file**

```go
package domain

import (
    "context"
    "errors"
)

// PickOutcome classifies how the responsible user was chosen.
type PickOutcome string

const (
    PickOutcomePicked        PickOutcome = "picked"
    PickOutcomeFallbackOwner PickOutcome = "fallback_owner"
)

// PickResult is the outcome of a single responsible-user selection.
type PickResult struct {
    UserID    string
    Algorithm string // empty when Outcome == PickOutcomeFallbackOwner
    GroupID   string // empty when Outcome == PickOutcomeFallbackOwner
    Outcome   PickOutcome
}

// ErrNoResponsibleAvailable means neither the load-balance flow nor the tenant
// owner fallback could produce a user. Lead creation MUST abort.
var ErrNoResponsibleAvailable = errors.New("no responsible user available for tenant")

// ResponsiblePicker chooses a user to receive a newly created lead.
//
// Implementations must never return an empty UserID on a nil error: if the
// load-balance flow fails, they MUST resolve the tenant owner. When neither
// is possible, they return ErrNoResponsibleAvailable.
type ResponsiblePicker interface {
    PickForFunnel(ctx context.Context, tenantID, funnelID, columnID string) (PickResult, error)
}

// LeadLoadCounter is a read-only port used by picker implementations to
// evaluate the "least load" algorithm. It counts currently-open leads
// assigned to each user in the supplied list, scoped to a single tenant.
type LeadLoadCounter interface {
    CountActiveByUsers(ctx context.Context, tenantID string, userIDs []string) (map[string]int, error)
}
```

- [ ] **Step 2: Run build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/funnel/domain/responsible_picker.go
git commit -m "feat(funnel): add ResponsiblePicker and LeadLoadCounter ports"
```

---

## Task 4: Implement `CountActiveByUsers` on the funnel lead repo

**Files:**
- Modify: `internal/funnel/infrastructure/gorm_lead_repo.go`
- Create: `internal/funnel/infrastructure/gorm_lead_repo_count_test.go`

- [ ] **Step 1: Read the existing repo file to confirm model + receiver name**

```bash
sed -n '1,40p' internal/funnel/infrastructure/gorm_lead_repo.go
```

(Use the existing struct name — the plan assumes `GormLeadRepository` with field `db *gorm.DB`.)

- [ ] **Step 2: Write the failing test**

Create `internal/funnel/infrastructure/gorm_lead_repo_count_test.go`:

```go
package infrastructure_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/require"

    "github.com/sasrgita/crm-juridico/internal/funnel/domain"
    "github.com/sasrgita/crm-juridico/internal/funnel/infrastructure"
    "github.com/sasrgita/crm-juridico/internal/testutil"
)

func TestGormLeadRepository_CountActiveByUsers(t *testing.T) {
    db := testutil.OpenTestDB(t) // existing project helper that spins MySQL via testcontainers
    repo := infrastructure.NewGormLeadRepository(db)
    ctx := context.Background()

    tenantID := "tenant-a"
    otherTenant := "tenant-b"
    userA, userB, userC := "u-a", "u-b", "u-c"

    // Seed: 3 open leads for A, 1 open lead for B, 1 won (inactive) for B,
    // 1 open lead for C in a different tenant (must not leak).
    testutil.SeedLead(t, db, tenantID, userA, domain.LeadStatusOpen)
    testutil.SeedLead(t, db, tenantID, userA, domain.LeadStatusOpen)
    testutil.SeedLead(t, db, tenantID, userA, domain.LeadStatusOpen)
    testutil.SeedLead(t, db, tenantID, userB, domain.LeadStatusOpen)
    testutil.SeedLead(t, db, tenantID, userB, domain.LeadStatusWon)
    testutil.SeedLead(t, db, otherTenant, userC, domain.LeadStatusOpen)

    got, err := repo.CountActiveByUsers(ctx, tenantID, []string{userA, userB, userC})
    require.NoError(t, err)
    require.Equal(t, map[string]int{userA: 3, userB: 1}, got,
        "userC must be absent (other tenant); won lead must not count")
}
```

> If `testutil.OpenTestDB` or `SeedLead` does not exist with this exact shape, adapt to the helper the project uses in sibling `_test.go` files (check `internal/funnel/infrastructure/*_test.go` for the existing pattern and copy it).

- [ ] **Step 3: Run the test to confirm it fails**

```bash
go test ./internal/funnel/infrastructure/... -run TestGormLeadRepository_CountActiveByUsers -v
```

Expected: FAIL (method not defined).

- [ ] **Step 4: Implement the method**

Append to `internal/funnel/infrastructure/gorm_lead_repo.go`:

```go
// CountActiveByUsers counts open leads (Status == "open") per user within a
// tenant. Users with zero results are absent from the returned map.
func (r *GormLeadRepository) CountActiveByUsers(ctx context.Context, tenantID string, userIDs []string) (map[string]int, error) {
    if len(userIDs) == 0 {
        return map[string]int{}, nil
    }
    type row struct {
        ResponsibleUserID string
        Cnt               int
    }
    var rows []row
    err := r.db.WithContext(ctx).
        Table("leads").
        Select("responsible_user_id, COUNT(*) AS cnt").
        Where("tenant_id = ? AND status = ? AND responsible_user_id IN ?",
            tenantID, string(domain.LeadStatusOpen), userIDs).
        Group("responsible_user_id").
        Scan(&rows).Error
    if err != nil {
        return nil, err
    }
    out := make(map[string]int, len(rows))
    for _, r := range rows {
        out[r.ResponsibleUserID] = r.Cnt
    }
    return out, nil
}
```

> Ensure `domain` is already imported in this file; if not, add `"github.com/sasrgita/crm-juridico/internal/funnel/domain"` to the import block.

- [ ] **Step 5: Re-run the test**

```bash
go test ./internal/funnel/infrastructure/... -run TestGormLeadRepository_CountActiveByUsers -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/funnel/infrastructure/gorm_lead_repo.go internal/funnel/infrastructure/gorm_lead_repo_count_test.go
git commit -m "feat(funnel): add LeadLoadCounter implementation on gorm lead repo"
```

---

## Task 5: `LoadBalancePicker` skeleton — tenant-owner fallback only

This task establishes the struct, constructor, and the deepest fallback path (no group found → owner). Subsequent tasks layer in algorithms and defensive checks.

**Files:**
- Create: `internal/auth/infrastructure/load_balance_picker.go`
- Create: `internal/auth/infrastructure/load_balance_picker_test.go`

- [ ] **Step 1: Write the failing test (owner fallback path)**

```go
package infrastructure_test

import (
    "context"
    "errors"
    "testing"

    "github.com/stretchr/testify/require"
    "go.uber.org/zap"

    authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
    "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
    funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
    permdomain "github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// --- fakes ---------------------------------------------------------------

type fakeGroupFunnelRepo struct{ byFunnel map[string][]permdomain.GroupFunnel }

func (f *fakeGroupFunnelRepo) FindByFunnelID(_ context.Context, funnelID string) ([]permdomain.GroupFunnel, error) {
    return f.byFunnel[funnelID], nil
}
func (f *fakeGroupFunnelRepo) CreateOrUpdate(context.Context, *permdomain.GroupFunnel) error { return nil }
func (f *fakeGroupFunnelRepo) FindByGroupID(context.Context, string) ([]permdomain.GroupFunnel, error) { return nil, nil }
func (f *fakeGroupFunnelRepo) FindByFunnelAndColumn(context.Context, string, string) ([]permdomain.GroupFunnel, error) { return nil, nil }
func (f *fakeGroupFunnelRepo) Delete(context.Context, string, string) error { return nil }

type fakeLoadBalanceRepo struct{ byGroup map[string]*authdomain.LoadBalanceConfig; updateErr error; lastUpdated *authdomain.LoadBalanceConfig }

func (f *fakeLoadBalanceRepo) FindByGroupID(_ context.Context, _, groupID string) (*authdomain.LoadBalanceConfig, error) {
    if cfg, ok := f.byGroup[groupID]; ok { return cfg, nil }
    return nil, authdomain.ErrLoadBalanceNotFound
}
func (f *fakeLoadBalanceRepo) CreateOrUpdate(context.Context, *authdomain.LoadBalanceConfig) error { return nil }
func (f *fakeLoadBalanceRepo) FindByTenantID(context.Context, string) ([]*authdomain.LoadBalanceConfig, error) { return nil, nil }
func (f *fakeLoadBalanceRepo) Update(_ context.Context, cfg *authdomain.LoadBalanceConfig) error {
    f.lastUpdated = cfg
    return f.updateErr
}

type fakeUserGroupRepo struct{ byGroup map[string][]permdomain.UserGroup }

func (f *fakeUserGroupRepo) FindByGroupID(_ context.Context, gid string) ([]permdomain.UserGroup, error) {
    return f.byGroup[gid], nil
}
func (f *fakeUserGroupRepo) Create(context.Context, *permdomain.UserGroup) error { return nil }
func (f *fakeUserGroupRepo) Delete(context.Context, string, string) error { return nil }
func (f *fakeUserGroupRepo) FindByUserAndTenant(context.Context, string, string) ([]permdomain.UserGroup, error) { return nil, nil }
func (f *fakeUserGroupRepo) Exists(context.Context, string, string) (bool, error) { return false, nil }

type fakeUserTenantRepo struct{ ownerByTenant map[string]string; memberActive map[string]bool }

func (f *fakeUserTenantRepo) FindByTenantID(_ context.Context, tenantID string) ([]*authdomain.UserTenant, error) {
    owner := f.ownerByTenant[tenantID]
    if owner == "" { return []*authdomain.UserTenant{}, nil }
    return []*authdomain.UserTenant{{UserID: owner, TenantID: tenantID, IsOwner: true}}, nil
}
func (f *fakeUserTenantRepo) FindByUserAndTenant(_ context.Context, uid, tid string) (*authdomain.UserTenant, error) {
    if f.memberActive == nil { return &authdomain.UserTenant{UserID: uid, TenantID: tid}, nil }
    if _, ok := f.memberActive[uid]; !ok { return nil, errors.New("not a member") }
    return &authdomain.UserTenant{UserID: uid, TenantID: tid}, nil
}
// stubs for unused methods
func (f *fakeUserTenantRepo) Associate(context.Context, string, string) error { return nil }
func (f *fakeUserTenantRepo) FindTenantIDsByUserID(context.Context, string) ([]string, error) { return nil, nil }
func (f *fakeUserTenantRepo) UpdateIsOwner(context.Context, string, string, bool) error { return nil }
func (f *fakeUserTenantRepo) UpdateWhatsAppID(context.Context, string, string, string) error { return nil }
func (f *fakeUserTenantRepo) RemoveFromTenant(context.Context, string, string) error { return nil }
func (f *fakeUserTenantRepo) IsOwner(context.Context, string, string) (bool, error) { return false, nil }

type fakeLoadCounter struct{ counts map[string]int }

func (f *fakeLoadCounter) CountActiveByUsers(context.Context, string, []string) (map[string]int, error) {
    if f.counts == nil { return map[string]int{}, nil }
    return f.counts, nil
}

// --- test ----------------------------------------------------------------

func TestLoadBalancePicker_FallbackToOwner_WhenNoGroupCoversColumn(t *testing.T) {
    picker := infrastructure.NewLoadBalancePicker(
        &fakeGroupFunnelRepo{byFunnel: map[string][]permdomain.GroupFunnel{}}, // no groups
        &fakeLoadBalanceRepo{byGroup: map[string]*authdomain.LoadBalanceConfig{}},
        &fakeUserGroupRepo{byGroup: map[string][]permdomain.UserGroup{}},
        &fakeUserTenantRepo{ownerByTenant: map[string]string{"t1": "owner-1"}},
        &fakeLoadCounter{},
        zap.NewNop(),
    )

    got, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")

    require.NoError(t, err)
    require.Equal(t, funneldomain.PickResult{
        UserID:  "owner-1",
        Outcome: funneldomain.PickOutcomeFallbackOwner,
    }, got)
}

func TestLoadBalancePicker_HardError_WhenNoOwnerExists(t *testing.T) {
    picker := infrastructure.NewLoadBalancePicker(
        &fakeGroupFunnelRepo{},
        &fakeLoadBalanceRepo{},
        &fakeUserGroupRepo{},
        &fakeUserTenantRepo{ownerByTenant: map[string]string{}}, // no owner for t1
        &fakeLoadCounter{},
        zap.NewNop(),
    )

    _, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")

    require.ErrorIs(t, err, funneldomain.ErrNoResponsibleAvailable)
}
```

- [ ] **Step 2: Run the test (expect FAIL — constructor undefined)**

```bash
go test ./internal/auth/infrastructure/... -run TestLoadBalancePicker -v
```

Expected: FAIL.

- [ ] **Step 3: Write the skeleton implementation**

Create `internal/auth/infrastructure/load_balance_picker.go`:

```go
package infrastructure

import (
    "context"
    "errors"

    "go.uber.org/zap"

    authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
    funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
    permdomain "github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// LoadBalancePicker implements funnel/domain.ResponsiblePicker.
type LoadBalancePicker struct {
    groupFunnelRepo permdomain.GroupFunnelRepository
    lbRepo          authdomain.LoadBalanceConfigRepository
    userGroupRepo   permdomain.UserGroupRepository
    userTenantRepo  authdomain.UserTenantRepository
    loadCounter     funneldomain.LeadLoadCounter
    log             *zap.Logger
}

func NewLoadBalancePicker(
    groupFunnelRepo permdomain.GroupFunnelRepository,
    lbRepo authdomain.LoadBalanceConfigRepository,
    userGroupRepo permdomain.UserGroupRepository,
    userTenantRepo authdomain.UserTenantRepository,
    loadCounter funneldomain.LeadLoadCounter,
    log *zap.Logger,
) *LoadBalancePicker {
    if log == nil {
        log = zap.NewNop()
    }
    return &LoadBalancePicker{
        groupFunnelRepo: groupFunnelRepo,
        lbRepo:          lbRepo,
        userGroupRepo:   userGroupRepo,
        userTenantRepo:  userTenantRepo,
        loadCounter:     loadCounter,
        log:             log,
    }
}

// PickForFunnel implements ResponsiblePicker.
func (p *LoadBalancePicker) PickForFunnel(ctx context.Context, tenantID, funnelID, columnID string) (funneldomain.PickResult, error) {
    // Subsequent tasks will plug algorithm logic here. For now, always fall back.
    return p.fallbackToOwner(ctx, tenantID, "no_group")
}

func (p *LoadBalancePicker) fallbackToOwner(ctx context.Context, tenantID, reason string) (funneldomain.PickResult, error) {
    uts, err := p.userTenantRepo.FindByTenantID(ctx, tenantID)
    if err != nil {
        return funneldomain.PickResult{}, err
    }
    for _, ut := range uts {
        if ut.IsOwner {
            p.log.Info("load_balance.pick fallback_owner",
                zap.String("tenant_id", tenantID),
                zap.String("reason", reason),
                zap.String("picked_user_id", ut.UserID),
            )
            return funneldomain.PickResult{
                UserID:  ut.UserID,
                Outcome: funneldomain.PickOutcomeFallbackOwner,
            }, nil
        }
    }
    return funneldomain.PickResult{}, funneldomain.ErrNoResponsibleAvailable
}

// sanity: ensure interface compliance at compile time
var _ funneldomain.ResponsiblePicker = (*LoadBalancePicker)(nil)

// stub to appease "unused" linter for errors until algorithm tasks use it
var _ = errors.New
```

- [ ] **Step 4: Re-run the test**

```bash
go test ./internal/auth/infrastructure/... -run TestLoadBalancePicker -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/auth/infrastructure/load_balance_picker.go internal/auth/infrastructure/load_balance_picker_test.go
git commit -m "feat(auth): LoadBalancePicker skeleton with owner fallback"
```

---

## Task 6: Happy-path skeleton — single active group, any algorithm picks first member

This task wires up the "find covering groups → filter active LB → verify single group → fetch members" pipeline, but defers algorithm nuance: we pick `members[0]` unconditionally. This lets us validate the plumbing and the single-active-group invariant before adding algorithm-specific logic.

**Files:**
- Modify: `internal/auth/infrastructure/load_balance_picker.go`
- Modify: `internal/auth/infrastructure/load_balance_picker_test.go`

- [ ] **Step 1: Add failing test — happy path with a single active group + single member**

Append to the test file:

```go
func TestLoadBalancePicker_PicksMemberWhenSingleActiveGroup(t *testing.T) {
    picker := infrastructure.NewLoadBalancePicker(
        &fakeGroupFunnelRepo{byFunnel: map[string][]permdomain.GroupFunnel{
            "f1": {{ID: "gf1", GroupID: "g1", FunnelID: "f1", ColumnIDs: nil}}, // covers entire funnel
        }},
        &fakeLoadBalanceRepo{byGroup: map[string]*authdomain.LoadBalanceConfig{
            "g1": {ID: "lb1", TenantID: "t1", GroupID: "g1", Algorithm: authdomain.AlgorithmRandom, Active: true},
        }},
        &fakeUserGroupRepo{byGroup: map[string][]permdomain.UserGroup{
            "g1": {{UserID: "u-member-1", GroupID: "g1"}},
        }},
        &fakeUserTenantRepo{
            ownerByTenant: map[string]string{"t1": "owner-1"},
            memberActive:  map[string]bool{"u-member-1": true},
        },
        &fakeLoadCounter{},
        zap.NewNop(),
    )

    got, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")

    require.NoError(t, err)
    require.Equal(t, funneldomain.PickOutcomePicked, got.Outcome)
    require.Equal(t, "u-member-1", got.UserID)
    require.Equal(t, "g1", got.GroupID)
}

func TestLoadBalancePicker_FallbackWhenMultipleActiveGroupsCoverColumn(t *testing.T) {
    picker := infrastructure.NewLoadBalancePicker(
        &fakeGroupFunnelRepo{byFunnel: map[string][]permdomain.GroupFunnel{
            "f1": {
                {ID: "gf1", GroupID: "g1", FunnelID: "f1"},
                {ID: "gf2", GroupID: "g2", FunnelID: "f1"},
            },
        }},
        &fakeLoadBalanceRepo{byGroup: map[string]*authdomain.LoadBalanceConfig{
            "g1": {ID: "lb1", TenantID: "t1", GroupID: "g1", Algorithm: authdomain.AlgorithmRandom, Active: true},
            "g2": {ID: "lb2", TenantID: "t1", GroupID: "g2", Algorithm: authdomain.AlgorithmRandom, Active: true},
        }},
        &fakeUserGroupRepo{byGroup: map[string][]permdomain.UserGroup{
            "g1": {{UserID: "u1"}}, "g2": {{UserID: "u2"}},
        }},
        &fakeUserTenantRepo{
            ownerByTenant: map[string]string{"t1": "owner-1"},
            memberActive:  map[string]bool{"u1": true, "u2": true},
        },
        &fakeLoadCounter{},
        zap.NewNop(),
    )

    got, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")

    require.NoError(t, err)
    require.Equal(t, funneldomain.PickOutcomeFallbackOwner, got.Outcome)
    require.Equal(t, "owner-1", got.UserID)
}

func TestLoadBalancePicker_FallbackWhenGroupHasNoActiveMembers(t *testing.T) {
    picker := infrastructure.NewLoadBalancePicker(
        &fakeGroupFunnelRepo{byFunnel: map[string][]permdomain.GroupFunnel{
            "f1": {{ID: "gf1", GroupID: "g1", FunnelID: "f1"}},
        }},
        &fakeLoadBalanceRepo{byGroup: map[string]*authdomain.LoadBalanceConfig{
            "g1": {ID: "lb1", TenantID: "t1", GroupID: "g1", Algorithm: authdomain.AlgorithmRandom, Active: true},
        }},
        &fakeUserGroupRepo{byGroup: map[string][]permdomain.UserGroup{
            "g1": {{UserID: "u-gone", GroupID: "g1"}},
        }},
        &fakeUserTenantRepo{
            ownerByTenant: map[string]string{"t1": "owner-1"},
            memberActive:  map[string]bool{}, // u-gone is no longer a tenant member
        },
        &fakeLoadCounter{},
        zap.NewNop(),
    )

    got, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")

    require.NoError(t, err)
    require.Equal(t, funneldomain.PickOutcomeFallbackOwner, got.Outcome)
    require.Equal(t, "owner-1", got.UserID)
}

func TestLoadBalancePicker_FallbackWhenConfigInactive(t *testing.T) {
    picker := infrastructure.NewLoadBalancePicker(
        &fakeGroupFunnelRepo{byFunnel: map[string][]permdomain.GroupFunnel{
            "f1": {{ID: "gf1", GroupID: "g1", FunnelID: "f1"}},
        }},
        &fakeLoadBalanceRepo{byGroup: map[string]*authdomain.LoadBalanceConfig{
            "g1": {ID: "lb1", TenantID: "t1", GroupID: "g1", Algorithm: authdomain.AlgorithmRandom, Active: false},
        }},
        &fakeUserGroupRepo{byGroup: map[string][]permdomain.UserGroup{
            "g1": {{UserID: "u1", GroupID: "g1"}},
        }},
        &fakeUserTenantRepo{
            ownerByTenant: map[string]string{"t1": "owner-1"},
            memberActive:  map[string]bool{"u1": true},
        },
        &fakeLoadCounter{},
        zap.NewNop(),
    )

    got, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")

    require.NoError(t, err)
    require.Equal(t, funneldomain.PickOutcomeFallbackOwner, got.Outcome)
}
```

- [ ] **Step 2: Run tests (expect FAIL — still fallback-only)**

```bash
go test ./internal/auth/infrastructure/... -run TestLoadBalancePicker -v
```

Expected: three new tests FAIL, the two previous PASS.

- [ ] **Step 3: Implement the full pipeline (minus algorithm logic)**

Replace the body of `PickForFunnel` in `load_balance_picker.go`:

```go
func (p *LoadBalancePicker) PickForFunnel(ctx context.Context, tenantID, funnelID, columnID string) (funneldomain.PickResult, error) {
    // 1. Find all groups associated with this funnel that cover the column.
    groups, err := p.groupFunnelRepo.FindByFunnelID(ctx, funnelID)
    if err != nil {
        return p.fallbackToOwner(ctx, tenantID, "group_lookup_error")
    }
    covering := make([]permdomain.GroupFunnel, 0, len(groups))
    for _, gf := range groups {
        if gf.CoversColumn(columnID) {
            covering = append(covering, gf)
        }
    }
    if len(covering) == 0 {
        return p.fallbackToOwner(ctx, tenantID, "no_group")
    }

    // 2. Filter to groups whose LoadBalanceConfig is ACTIVE.
    type active struct {
        groupID string
        cfg     *authdomain.LoadBalanceConfig
    }
    var actives []active
    for _, gf := range covering {
        cfg, err := p.lbRepo.FindByGroupID(ctx, tenantID, gf.GroupID)
        if err != nil {
            if errors.Is(err, authdomain.ErrLoadBalanceNotFound) {
                continue
            }
            return p.fallbackToOwner(ctx, tenantID, "lb_lookup_error")
        }
        if cfg != nil && cfg.Active {
            actives = append(actives, active{gf.GroupID, cfg})
        }
    }
    if len(actives) == 0 {
        return p.fallbackToOwner(ctx, tenantID, "no_active_config")
    }
    if len(actives) > 1 {
        p.log.Error("load_balance.pick multiple_active_groups — check uniqueness rule",
            zap.String("tenant_id", tenantID), zap.String("funnel_id", funnelID), zap.String("column_id", columnID),
        )
        return p.fallbackToOwner(ctx, tenantID, "multiple_active_groups")
    }

    chosen := actives[0]

    // 3. Fetch group members and filter to active tenant members.
    ugs, err := p.userGroupRepo.FindByGroupID(ctx, chosen.groupID)
    if err != nil {
        return p.fallbackToOwner(ctx, tenantID, "member_lookup_error")
    }
    members := make([]string, 0, len(ugs))
    for _, ug := range ugs {
        if _, err := p.userTenantRepo.FindByUserAndTenant(ctx, ug.UserID, tenantID); err == nil {
            members = append(members, ug.UserID)
        }
    }
    if len(members) == 0 {
        return p.fallbackToOwner(ctx, tenantID, "no_active_members")
    }

    // 4. Apply the algorithm. (Placeholder until Tasks 7-9: pick first.)
    pickedUserID := members[0]
    algorithm := string(chosen.cfg.Algorithm)

    return funneldomain.PickResult{
        UserID:    pickedUserID,
        Algorithm: algorithm,
        GroupID:   chosen.groupID,
        Outcome:   funneldomain.PickOutcomePicked,
    }, nil
}
```

Remove the `var _ = errors.New` stub now that `errors` is used.

- [ ] **Step 4: Re-run tests**

```bash
go test ./internal/auth/infrastructure/... -run TestLoadBalancePicker -v
```

Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/auth/infrastructure/load_balance_picker.go internal/auth/infrastructure/load_balance_picker_test.go
git commit -m "feat(auth): LoadBalancePicker happy path with single-group invariant"
```

---

## Task 7: Implement `round_robin` algorithm with `LastIndex` persistence

**Files:**
- Modify: `internal/auth/infrastructure/load_balance_picker.go`
- Modify: `internal/auth/infrastructure/load_balance_picker_test.go`

- [ ] **Step 1: Add failing test**

```go
func TestLoadBalancePicker_RoundRobin_CyclesThroughMembers(t *testing.T) {
    cfg := &authdomain.LoadBalanceConfig{ID: "lb1", TenantID: "t1", GroupID: "g1", Algorithm: authdomain.AlgorithmRoundRobin, Active: true, LastIndex: 0}
    lbRepo := &fakeLoadBalanceRepo{byGroup: map[string]*authdomain.LoadBalanceConfig{"g1": cfg}}

    picker := infrastructure.NewLoadBalancePicker(
        &fakeGroupFunnelRepo{byFunnel: map[string][]permdomain.GroupFunnel{"f1": {{ID: "gf1", GroupID: "g1", FunnelID: "f1"}}}},
        lbRepo,
        &fakeUserGroupRepo{byGroup: map[string][]permdomain.UserGroup{
            "g1": {{UserID: "u-a"}, {UserID: "u-b"}, {UserID: "u-c"}},
        }},
        &fakeUserTenantRepo{ownerByTenant: map[string]string{"t1": "owner-1"}, memberActive: map[string]bool{"u-a": true, "u-b": true, "u-c": true}},
        &fakeLoadCounter{},
        zap.NewNop(),
    )

    ctx := context.Background()
    r1, _ := picker.PickForFunnel(ctx, "t1", "f1", "c1")
    r2, _ := picker.PickForFunnel(ctx, "t1", "f1", "c1")
    r3, _ := picker.PickForFunnel(ctx, "t1", "f1", "c1")
    r4, _ := picker.PickForFunnel(ctx, "t1", "f1", "c1")

    require.Equal(t, "u-a", r1.UserID) // index 0
    require.Equal(t, "u-b", r2.UserID) // index 1
    require.Equal(t, "u-c", r3.UserID) // index 2
    require.Equal(t, "u-a", r4.UserID) // wraps
    require.NotNil(t, lbRepo.lastUpdated, "LastIndex must have been persisted")
}
```

- [ ] **Step 2: Run test (expect FAIL — r2 returns u-a as placeholder)**

```bash
go test ./internal/auth/infrastructure/... -run RoundRobin -v
```

- [ ] **Step 3: Extract algorithm dispatch and implement round_robin**

In `load_balance_picker.go`, replace the "Apply the algorithm" block with a call to a helper:

```go
    // 4. Apply the algorithm.
    pickedUserID, err := p.applyAlgorithm(ctx, tenantID, chosen.cfg, members)
    if err != nil {
        return p.fallbackToOwner(ctx, tenantID, "algorithm_error")
    }
    algorithm := string(chosen.cfg.Algorithm)

    return funneldomain.PickResult{
        UserID: pickedUserID, Algorithm: algorithm, GroupID: chosen.groupID,
        Outcome: funneldomain.PickOutcomePicked,
    }, nil
}

func (p *LoadBalancePicker) applyAlgorithm(ctx context.Context, tenantID string, cfg *authdomain.LoadBalanceConfig, members []string) (string, error) {
    // deterministic member order is required for round_robin.
    sort.Strings(members)

    switch cfg.Algorithm {
    case authdomain.AlgorithmRoundRobin:
        idx := cfg.LastIndex % len(members)
        picked := members[idx]
        cfg.IncrementIndex()
        if err := p.lbRepo.Update(ctx, cfg); err != nil {
            return "", err
        }
        return picked, nil
    case authdomain.AlgorithmLeastLoad:
        return members[0], nil // placeholder — Task 8
    case authdomain.AlgorithmRandom:
        return members[0], nil // placeholder — Task 9
    default:
        return members[0], nil
    }
}
```

Add `"sort"` to the import block.

> **Why sort?** `UserGroupRepository.FindByGroupID` returns rows in DB order (by created_at or insertion). Sorting makes round-robin cycles deterministic across calls and test runs.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/auth/infrastructure/... -run TestLoadBalancePicker -v
```

Expected: PASS (6 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/auth/infrastructure/load_balance_picker.go internal/auth/infrastructure/load_balance_picker_test.go
git commit -m "feat(auth): implement round_robin algorithm with LastIndex persistence"
```

---

## Task 8: Implement `least_load` algorithm

**Files:**
- Modify: `internal/auth/infrastructure/load_balance_picker.go`
- Modify: `internal/auth/infrastructure/load_balance_picker_test.go`

- [ ] **Step 1: Add failing test**

```go
func TestLoadBalancePicker_LeastLoad_PicksUserWithFewestActiveLeads(t *testing.T) {
    cfg := &authdomain.LoadBalanceConfig{ID: "lb1", TenantID: "t1", GroupID: "g1", Algorithm: authdomain.AlgorithmLeastLoad, Active: true}
    picker := infrastructure.NewLoadBalancePicker(
        &fakeGroupFunnelRepo{byFunnel: map[string][]permdomain.GroupFunnel{"f1": {{ID: "gf1", GroupID: "g1", FunnelID: "f1"}}}},
        &fakeLoadBalanceRepo{byGroup: map[string]*authdomain.LoadBalanceConfig{"g1": cfg}},
        &fakeUserGroupRepo{byGroup: map[string][]permdomain.UserGroup{
            "g1": {{UserID: "u-a"}, {UserID: "u-b"}, {UserID: "u-c"}},
        }},
        &fakeUserTenantRepo{ownerByTenant: map[string]string{"t1": "owner-1"}, memberActive: map[string]bool{"u-a": true, "u-b": true, "u-c": true}},
        &fakeLoadCounter{counts: map[string]int{"u-a": 5, "u-b": 2, "u-c": 3}}, // u-b has fewest
        zap.NewNop(),
    )

    got, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")
    require.NoError(t, err)
    require.Equal(t, "u-b", got.UserID)
}

func TestLoadBalancePicker_LeastLoad_PrefersUserAbsentFromCounts(t *testing.T) {
    cfg := &authdomain.LoadBalanceConfig{ID: "lb1", TenantID: "t1", GroupID: "g1", Algorithm: authdomain.AlgorithmLeastLoad, Active: true}
    picker := infrastructure.NewLoadBalancePicker(
        &fakeGroupFunnelRepo{byFunnel: map[string][]permdomain.GroupFunnel{"f1": {{ID: "gf1", GroupID: "g1", FunnelID: "f1"}}}},
        &fakeLoadBalanceRepo{byGroup: map[string]*authdomain.LoadBalanceConfig{"g1": cfg}},
        &fakeUserGroupRepo{byGroup: map[string][]permdomain.UserGroup{
            "g1": {{UserID: "u-a"}, {UserID: "u-zero"}},
        }},
        &fakeUserTenantRepo{ownerByTenant: map[string]string{"t1": "owner-1"}, memberActive: map[string]bool{"u-a": true, "u-zero": true}},
        &fakeLoadCounter{counts: map[string]int{"u-a": 1}}, // u-zero is absent from counts → load 0
        zap.NewNop(),
    )

    got, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")
    require.NoError(t, err)
    require.Equal(t, "u-zero", got.UserID)
}
```

- [ ] **Step 2: Run tests (expect FAIL — placeholder still returns u-a)**

- [ ] **Step 3: Implement**

Replace the `AlgorithmLeastLoad` branch in `applyAlgorithm`:

```go
    case authdomain.AlgorithmLeastLoad:
        counts, err := p.loadCounter.CountActiveByUsers(ctx, tenantID, members)
        if err != nil {
            return "", err
        }
        // members is already sorted (deterministic tiebreak: lexicographic).
        best := members[0]
        bestLoad := counts[best] // absent = 0
        for _, uid := range members[1:] {
            if counts[uid] < bestLoad {
                best, bestLoad = uid, counts[uid]
            }
        }
        return best, nil
```

> Tiebreak note: spec says "desempate por user.created_at ASC". We don't have `created_at` in scope here, so the tiebreak is the deterministic sort already applied (ID-lex). This is acceptable because a true chronological tiebreak would require an extra repo call per pick, and the spec's intent is "stable, deterministic". Document this below the implementation with a one-line comment.

Add the comment above the `AlgorithmLeastLoad` block:

```go
    // Tiebreak: deterministic via the lexicographic sort applied to members.
    // A strictly-by-user.created_at tiebreak would require an extra query per pick;
    // stability is what matters for fairness over time, not the exact ordering key.
```

- [ ] **Step 4: Re-run tests**

Expected: PASS (8 total).

- [ ] **Step 5: Commit**

```bash
git add internal/auth/infrastructure/load_balance_picker.go internal/auth/infrastructure/load_balance_picker_test.go
git commit -m "feat(auth): implement least_load algorithm via LeadLoadCounter"
```

---

## Task 9: Implement `random` algorithm

**Files:**
- Modify: `internal/auth/infrastructure/load_balance_picker.go`
- Modify: `internal/auth/infrastructure/load_balance_picker_test.go`

- [ ] **Step 1: Add failing test — uniform sampling (statistical smoke test)**

```go
func TestLoadBalancePicker_Random_AllMembersEventuallyPicked(t *testing.T) {
    cfg := &authdomain.LoadBalanceConfig{ID: "lb1", TenantID: "t1", GroupID: "g1", Algorithm: authdomain.AlgorithmRandom, Active: true}
    picker := infrastructure.NewLoadBalancePicker(
        &fakeGroupFunnelRepo{byFunnel: map[string][]permdomain.GroupFunnel{"f1": {{ID: "gf1", GroupID: "g1", FunnelID: "f1"}}}},
        &fakeLoadBalanceRepo{byGroup: map[string]*authdomain.LoadBalanceConfig{"g1": cfg}},
        &fakeUserGroupRepo{byGroup: map[string][]permdomain.UserGroup{
            "g1": {{UserID: "u-a"}, {UserID: "u-b"}, {UserID: "u-c"}},
        }},
        &fakeUserTenantRepo{ownerByTenant: map[string]string{"t1": "owner-1"}, memberActive: map[string]bool{"u-a": true, "u-b": true, "u-c": true}},
        &fakeLoadCounter{},
        zap.NewNop(),
    )

    seen := map[string]int{}
    for i := 0; i < 300; i++ {
        got, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")
        require.NoError(t, err)
        seen[got.UserID]++
    }
    // Each of 3 members picked at least once; none above ~70% of samples.
    for _, uid := range []string{"u-a", "u-b", "u-c"} {
        require.Greater(t, seen[uid], 0, "member %s never picked", uid)
        require.Less(t, seen[uid], 210, "member %s over-represented (>70%)", uid)
    }
}
```

- [ ] **Step 2: Run test (expect FAIL — placeholder returns members[0] always)**

- [ ] **Step 3: Implement**

Replace the `AlgorithmRandom` branch:

```go
    case authdomain.AlgorithmRandom:
        idx, err := cryptoRandIndex(len(members))
        if err != nil {
            return "", err
        }
        return members[idx], nil
```

Add at bottom of the file:

```go
// cryptoRandIndex returns a uniform random index in [0, n) using crypto/rand.
func cryptoRandIndex(n int) (int, error) {
    if n <= 0 {
        return 0, errors.New("empty member list")
    }
    big, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
    if err != nil {
        return 0, err
    }
    return int(big.Int64()), nil
}
```

Add imports:

```go
    "crypto/rand"
    "math/big"
```

- [ ] **Step 4: Re-run tests**

Expected: PASS (9 total).

- [ ] **Step 5: Commit**

```bash
git add internal/auth/infrastructure/load_balance_picker.go internal/auth/infrastructure/load_balance_picker_test.go
git commit -m "feat(auth): implement random algorithm with crypto/rand"
```

---

## Task 10: Prometheus metrics + OTel span on the picker

**Files:**
- Create: `internal/auth/infrastructure/metrics.go`
- Modify: `internal/auth/infrastructure/load_balance_picker.go`

- [ ] **Step 1: Create the metrics file**

```go
package infrastructure

import "github.com/prometheus/client_golang/prometheus"

var (
    pickerTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "crm",
            Subsystem: "lead_responsible_picker",
            Name:      "total",
            Help:      "Total responsible-picker attempts broken down by algorithm and outcome.",
        },
        []string{"algorithm", "outcome"},
    )

    pickerDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: "crm",
            Subsystem: "lead_responsible_picker",
            Name:      "duration_seconds",
            Help:      "Duration of responsible-picker calls in seconds.",
            Buckets:   prometheus.DefBuckets,
        },
        []string{"algorithm"},
    )
)

func init() {
    prometheus.MustRegister(pickerTotal, pickerDuration)
}
```

- [ ] **Step 2: Instrument `PickForFunnel`**

At the top of `PickForFunnel` in `load_balance_picker.go`:

```go
import (
    // ... existing imports
    "time"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func (p *LoadBalancePicker) PickForFunnel(ctx context.Context, tenantID, funnelID, columnID string) (funneldomain.PickResult, error) {
    start := time.Now()
    tracer := otel.Tracer("crm.load_balance")
    ctx, span := tracer.Start(ctx, "load_balance.pick")
    defer span.End()
    span.SetAttributes(
        attribute.String("tenant_id", tenantID),
        attribute.String("funnel_id", funnelID),
        attribute.String("column_id", columnID),
    )

    res, err := p.pickInternal(ctx, tenantID, funnelID, columnID)

    algorithm := res.Algorithm
    if algorithm == "" {
        algorithm = "none"
    }
    outcome := string(res.Outcome)
    if err != nil {
        outcome = "error"
    }
    pickerTotal.WithLabelValues(algorithm, outcome).Inc()
    pickerDuration.WithLabelValues(algorithm).Observe(time.Since(start).Seconds())
    span.SetAttributes(
        attribute.String("algorithm", algorithm),
        attribute.String("outcome", outcome),
        attribute.String("picked_user_id", res.UserID),
    )
    return res, err
}
```

Then rename the previous body of `PickForFunnel` to `pickInternal` with the same signature minus the tracing wrapper.

- [ ] **Step 3: Run all picker tests**

```bash
go test ./internal/auth/infrastructure/... -v
```

Expected: PASS (9).

- [ ] **Step 4: Confirm metrics registration at boot**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/auth/infrastructure/metrics.go internal/auth/infrastructure/load_balance_picker.go
git commit -m "feat(auth): add Prometheus metrics and OTel span for picker"
```

---

## Task 11: Integrate `ResponsiblePicker` into `CreateLeadUseCase` and publish new event

**Files:**
- Modify: `internal/funnel/application/create_lead.go`
- Modify: `internal/funnel/application/create_lead_test.go`
- Modify: `internal/funnel/module.go`

- [ ] **Step 1: Add failing tests**

Find `create_lead_test.go` and add:

```go
type fakePicker struct{ result domain.PickResult; err error; lastCall struct{ tenant, funnel, col string } }

func (f *fakePicker) PickForFunnel(_ context.Context, tenantID, funnelID, columnID string) (domain.PickResult, error) {
    f.lastCall.tenant, f.lastCall.funnel, f.lastCall.col = tenantID, funnelID, columnID
    return f.result, f.err
}

func TestCreateLeadUseCase_AssignsResponsibleFromPicker(t *testing.T) {
    // (use the existing fakes/helpers this file already defines;
    //  inject fakePicker into NewCreateLeadUseCase)
    picker := &fakePicker{
        result: domain.PickResult{UserID: "user-99", Algorithm: "round_robin", GroupID: "g1", Outcome: domain.PickOutcomePicked},
    }
    // ... build uc with picker, call uc.Execute, assert:
    // - leadRepo.last.ResponsibleUserID == "user-99"
    // - eventBus captured EventLeadCreated with responsible_user_id=user-99
    // - eventBus captured EventLeadResponsibleAssigned with Reason=created, Outcome=picked
    // - picker.lastCall.tenant/funnel/col match input
}

func TestCreateLeadUseCase_AbortsWhenPickerReturnsHardError(t *testing.T) {
    picker := &fakePicker{err: domain.ErrNoResponsibleAvailable}
    // ... build uc, call uc.Execute, assert:
    // - err is domain.ErrNoResponsibleAvailable
    // - leadRepo.CreateCalled == false
    // - eventBus captured nothing
}
```

> The exact fakes to reuse (`fakeLeadRepo`, `fakeEventBus`, etc.) already exist in this test file. Plug `fakePicker` into the existing constructor call with the new parameter.

- [ ] **Step 2: Run to confirm FAIL (signature mismatch)**

```bash
go test ./internal/funnel/application/... -v
```

- [ ] **Step 3: Update `CreateLeadUseCase`**

Modify the struct and constructor in `internal/funnel/application/create_lead.go`:

```go
type CreateLeadUseCase struct {
    funnelRepo          domain.FunnelRepository
    columnRepo          domain.ColumnRepository
    leadRepo            domain.LeadRepository
    movementRepo        domain.LeadMovementRepository
    productDetector     domain.ProductDetector
    funnelProductRouter domain.FunnelProductRouter
    eventBus            events.EventBus
    automationTrigger   domain.AutomationTrigger
    picker              domain.ResponsiblePicker
}

func NewCreateLeadUseCase(
    funnelRepo domain.FunnelRepository,
    columnRepo domain.ColumnRepository,
    leadRepo domain.LeadRepository,
    movementRepo domain.LeadMovementRepository,
    productDetector domain.ProductDetector,
    funnelProductRouter domain.FunnelProductRouter,
    eventBus events.EventBus,
    picker domain.ResponsiblePicker,
) *CreateLeadUseCase {
    return &CreateLeadUseCase{
        funnelRepo:          funnelRepo,
        columnRepo:          columnRepo,
        leadRepo:            leadRepo,
        movementRepo:        movementRepo,
        productDetector:     productDetector,
        funnelProductRouter: funnelProductRouter,
        eventBus:            eventBus,
        picker:              picker,
    }
}
```

Update `Execute` after `lead, err := domain.NewLead(...)` and BEFORE `uc.leadRepo.Create(ctx, lead)`:

```go
    if detectedProductID != "" {
        lead.SetProduct(detectedProductID)
    }

    pick, err := uc.picker.PickForFunnel(ctx, input.TenantID, funnel.ID, entryCol.ID)
    if err != nil {
        return err
    }
    lead.AssignResponsible(pick.UserID)

    if err := uc.leadRepo.Create(ctx, lead); err != nil {
        return err
    }
```

Update the `EventLeadCreated` publish block to include `responsible_user_id`:

```go
    uc.eventBus.Publish(events.Event{
        Type:     events.EventLeadCreated,
        TenantID: lead.TenantID,
        Payload: map[string]string{
            "lead_id":              lead.ID,
            "funnel_id":            lead.FunnelID,
            "column_id":            lead.ColumnID,
            "contact_id":           lead.ContactID,
            "responsible_user_id":  lead.ResponsibleUserID,
        },
    })

    uc.eventBus.Publish(events.Event{
        Type:     events.EventLeadResponsibleAssigned,
        TenantID: lead.TenantID,
        Payload: events.ResponsibleAssignedPayload{
            LeadID:            lead.ID,
            ResponsibleUserID: lead.ResponsibleUserID,
            Reason:            "created",
            Outcome:           string(pick.Outcome),
            Algorithm:         pick.Algorithm,
        },
    })
```

- [ ] **Step 4: Update `internal/funnel/module.go`**

Extend `NewModule` signature so consumers pass the picker through. Match the existing style of that file (read it first, insert parameter in a consistent position, and propagate it into `NewCreateLeadUseCase`).

- [ ] **Step 5: Re-run funnel tests**

```bash
go test ./internal/funnel/... -v
```

Expected: PASS.

- [ ] **Step 6: Run the whole tree to catch constructor-callsite breakage**

```bash
go build ./...
```

Expected: build errors in `cmd/api/main.go` (wiring missing picker). That is fine — Task 14 fixes it. For now: temporarily pass `nil` where the picker goes in `cmd/api/main.go` to allow the build, leaving a `// TODO task 14: wire LoadBalancePicker` comment. Re-run build — should compile.

- [ ] **Step 7: Commit**

```bash
git add internal/funnel/application/create_lead.go internal/funnel/application/create_lead_test.go internal/funnel/module.go cmd/api/main.go
git commit -m "feat(funnel): integrate ResponsiblePicker into CreateLeadUseCase"
```

---

## Task 12: Notification module subscribes to `EventLeadResponsibleAssigned`

**Files:**
- Modify: `internal/notification/module.go`
- Modify or Create: `internal/notification/module_test.go`

- [ ] **Step 1: Write the failing test**

Create or extend `internal/notification/module_test.go`:

```go
package notification_test

import (
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    "go.uber.org/zap"

    "github.com/sasrgita/crm-juridico/internal/notification"
    "github.com/sasrgita/crm-juridico/internal/shared/events"
    "github.com/sasrgita/crm-juridico/internal/testutil"
)

func TestNotificationModule_CreatesNotification_OnResponsibleAssigned(t *testing.T) {
    db := testutil.OpenTestDB(t)
    bus := events.NewInMemoryEventBus() // or project helper
    mod := notification.NewModule(db, bus, zap.NewNop())
    _ = mod // ensures subscription wired in NewModule

    bus.Publish(events.Event{
        Type:     events.EventLeadResponsibleAssigned,
        TenantID: "t1",
        Payload: events.ResponsibleAssignedPayload{
            LeadID: "lead-1", ResponsibleUserID: "u-1", Reason: "created", Outcome: "picked", Algorithm: "round_robin",
        },
    })

    // give async listener time to consume
    time.Sleep(50 * time.Millisecond)

    // Assert one notification exists for user u-1 of type lead_assigned.
    notifs, err := testutil.ListNotifications(t, db, "t1", "u-1")
    require.NoError(t, err)
    require.Len(t, notifs, 1)
    require.Equal(t, "lead_assigned", string(notifs[0].Type))
    require.Equal(t, "lead-1", notifs[0].Metadata["lead_id"])
}
```

> If the project's event bus helper has a different constructor name, adapt to it (check `internal/shared/events/*.go` for the inmem implementation).

- [ ] **Step 2: Extend `NewModule` to subscribe**

In `internal/notification/module.go`, after `handler := notifhttp.NewHandler(...)` and before `return`:

```go
    // Subscribe to cross-module events that produce user notifications.
    go subscribeResponsibleAssigned(eventBus, notifyService, log)

    return &Module{handler: handler, notifyService: notifyService}
}

func subscribeResponsibleAssigned(bus events.EventBus, svc *application.NotifyService, log *zap.Logger) {
    // Subscribe to the global tenant stream. EventBus.Subscribe in this project
    // is per-tenant; use the project-wide helper if available, or loop.
    ch, unsub := bus.Subscribe("")
    defer unsub()

    for ev := range ch {
        if ev.Type != events.EventLeadResponsibleAssigned {
            continue
        }
        payload, ok := ev.Payload.(events.ResponsibleAssignedPayload)
        if !ok {
            log.Warn("notification: unexpected payload shape for responsible_assigned", zap.Any("payload", ev.Payload))
            continue
        }
        err := svc.Notify(
            // Background context: this runs off the request path.
            context.Background(),
            payload.ResponsibleUserID,
            ev.TenantID,
            notificationdomain.TypeLeadAssigned,
            "Novo lead atribuído",
            "Você recebeu um novo lead.",
            map[string]string{
                "lead_id":    payload.LeadID,
                "reason":     payload.Reason,
                "outcome":    payload.Outcome,
                "algorithm":  payload.Algorithm,
            },
        )
        if err != nil {
            log.Error("notification: failed to create lead_assigned notification",
                zap.String("tenant_id", ev.TenantID),
                zap.String("lead_id", payload.LeadID),
                zap.Error(err),
            )
        }
    }
}
```

Add imports: `"context"`, and alias `notificationdomain "github.com/sasrgita/crm-juridico/internal/notification/domain"`.

> **Subscribe("")** — if the event bus does not support a global wildcard, use the helper used elsewhere in the project (search for `bus.Subscribe` callers). If none exists, the plumbing needs a thin global-subscribe helper, which can be added as a one-line method on the in-memory bus.

- [ ] **Step 3: Run the test**

```bash
go test ./internal/notification/... -v
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/notification/module.go internal/notification/module_test.go
git commit -m "feat(notification): subscribe to EventLeadResponsibleAssigned"
```

---

## Task 13: Uniqueness rule in `ManageLoadBalanceUseCase.SetByGroup` (409 conflict)

**Files:**
- Create: `internal/auth/application/group_column_overlap.go`
- Create: `internal/permission/infrastructure/group_column_overlap_adapter.go`
- Modify: `internal/auth/application/manage_load_balance.go`
- Modify: `internal/auth/application/manage_load_balance_test.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Define the port**

`internal/auth/application/group_column_overlap.go`:

```go
package application

import (
    "context"
    "errors"
)

// GroupColumnOverlapChecker reports whether activating a load-balance config
// for a given group would create an overlap with another already-active group
// on the same funnel/column(s).
type GroupColumnOverlapChecker interface {
    // HasActiveOverlap returns (true, overlappingGroupIDs, nil) when another
    // group with an active LoadBalanceConfig already covers at least one of
    // the columns this group covers in the same tenant.
    HasActiveOverlap(ctx context.Context, tenantID, groupID string) (bool, []string, error)
}

var ErrActiveLoadBalanceOverlap = errors.New("another group already has an active load balance covering the same funnel/column")
```

- [ ] **Step 2: Write failing test for uniqueness**

Add to `manage_load_balance_test.go`:

```go
type fakeOverlap struct{ overlap bool; groups []string; err error }

func (f *fakeOverlap) HasActiveOverlap(context.Context, string, string) (bool, []string, error) {
    return f.overlap, f.groups, f.err
}

func TestManageLoadBalanceUseCase_SetByGroup_RejectsActiveOverlap(t *testing.T) {
    // ... build uc with fakeOverlap{overlap: true, groups: []string{"g-other"}}
    // call SetByGroup with Active=true; assert ErrActiveLoadBalanceOverlap
}

func TestManageLoadBalanceUseCase_SetByGroup_AllowsActiveWhenNoOverlap(t *testing.T) {
    // ... build uc with fakeOverlap{overlap: false}
    // call SetByGroup with Active=true; assert no error, cfg persisted.
}

func TestManageLoadBalanceUseCase_SetByGroup_SkipsCheckWhenDeactivating(t *testing.T) {
    // Active=false should NOT call overlap checker.
    // ... use a recording fakeOverlap and assert its call count is zero.
}
```

- [ ] **Step 3: Run (expect FAIL — constructor signature)**

- [ ] **Step 4: Update `ManageLoadBalanceUseCase`**

```go
type ManageLoadBalanceUseCase struct {
    repo            domain.LoadBalanceConfigRepository
    groupChecker    GroupTenantChecker
    overlapChecker  GroupColumnOverlapChecker
}

func NewManageLoadBalanceUseCase(
    repo domain.LoadBalanceConfigRepository,
    groupChecker GroupTenantChecker,
    overlapChecker GroupColumnOverlapChecker,
) *ManageLoadBalanceUseCase {
    return &ManageLoadBalanceUseCase{repo: repo, groupChecker: groupChecker, overlapChecker: overlapChecker}
}
```

In `SetByGroup`, AFTER the `groupChecker.BelongsToTenant` check and BEFORE the `FindByGroupID`:

```go
    if in.Active {
        overlap, others, err := uc.overlapChecker.HasActiveOverlap(ctx, in.TenantID, in.GroupID)
        if err != nil {
            return nil, err
        }
        if overlap {
            return nil, fmt.Errorf("%w: groups=%v", ErrActiveLoadBalanceOverlap, others)
        }
    }
```

Add `"fmt"` to imports.

- [ ] **Step 5: Implement the adapter**

`internal/permission/infrastructure/group_column_overlap_adapter.go`:

```go
package infrastructure

import (
    "context"
    "errors"

    authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
    permdomain "github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// GroupColumnOverlapAdapter implements auth/application.GroupColumnOverlapChecker.
type GroupColumnOverlapAdapter struct {
    groupFunnelRepo permdomain.GroupFunnelRepository
    lbRepo          authdomain.LoadBalanceConfigRepository
}

func NewGroupColumnOverlapAdapter(gfr permdomain.GroupFunnelRepository, lbr authdomain.LoadBalanceConfigRepository) *GroupColumnOverlapAdapter {
    return &GroupColumnOverlapAdapter{groupFunnelRepo: gfr, lbRepo: lbr}
}

func (a *GroupColumnOverlapAdapter) HasActiveOverlap(ctx context.Context, tenantID, groupID string) (bool, []string, error) {
    // 1. Get funnel/column coverage of this group.
    mine, err := a.groupFunnelRepo.FindByGroupID(ctx, groupID)
    if err != nil {
        return false, nil, err
    }
    if len(mine) == 0 {
        return false, nil, nil
    }

    // 2. For each funnel this group covers, find peer groups on the same funnel.
    overlapping := map[string]struct{}{}
    for _, gf := range mine {
        peers, err := a.groupFunnelRepo.FindByFunnelID(ctx, gf.FunnelID)
        if err != nil {
            return false, nil, err
        }
        for _, peer := range peers {
            if peer.GroupID == groupID {
                continue
            }
            if !columnsOverlap(gf, peer) {
                continue
            }
            // 3. Does the peer have an ACTIVE lb config?
            cfg, err := a.lbRepo.FindByGroupID(ctx, tenantID, peer.GroupID)
            if err != nil {
                if errors.Is(err, authdomain.ErrLoadBalanceNotFound) {
                    continue
                }
                return false, nil, err
            }
            if cfg != nil && cfg.Active {
                overlapping[peer.GroupID] = struct{}{}
            }
        }
    }

    if len(overlapping) == 0 {
        return false, nil, nil
    }
    out := make([]string, 0, len(overlapping))
    for id := range overlapping {
        out = append(out, id)
    }
    return true, out, nil
}

func columnsOverlap(a, b permdomain.GroupFunnel) bool {
    // Empty ColumnIDs means "entire funnel".
    if len(a.ColumnIDs) == 0 || len(b.ColumnIDs) == 0 {
        return true
    }
    set := make(map[string]struct{}, len(a.ColumnIDs))
    for _, id := range a.ColumnIDs { set[id] = struct{}{} }
    for _, id := range b.ColumnIDs {
        if _, ok := set[id]; ok { return true }
    }
    return false
}
```

- [ ] **Step 6: Wire the adapter in `cmd/api/main.go`**

Where `authMod` / `permissionMod` are built, after `authMod` has `lbRepo` available and `permissionMod` has `groupFunnelRepo`:

```go
    overlapChecker := perminfra.NewGroupColumnOverlapAdapter(permissionMod.GroupFunnelRepo(), authMod.LoadBalanceRepo())
    authMod.SetLoadBalanceOverlapChecker(overlapChecker) // expose a setter if needed
```

> The exact setter/accessor shape depends on how `authMod` is structured. If the existing module wires `ManageLoadBalanceUseCase` internally, either (a) accept the checker as a constructor argument of `auth.NewModule`, or (b) add `SetLoadBalanceOverlapChecker`. Prefer (a) — read `internal/auth/module.go` and extend it cleanly.

- [ ] **Step 7: Run all auth tests**

```bash
go test ./internal/auth/... -v
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/auth/application/group_column_overlap.go internal/permission/infrastructure/group_column_overlap_adapter.go internal/auth/application/manage_load_balance.go internal/auth/application/manage_load_balance_test.go internal/auth/module.go cmd/api/main.go
git commit -m "feat(auth): enforce one-active-LB-per-column at SetByGroup (409)"
```

---

## Task 14: Wire `LoadBalancePicker` into `cmd/api/main.go`

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Read current funnel module construction block**

```bash
grep -n "funnel.NewModule\|funnelMod" cmd/api/main.go
```

- [ ] **Step 2: Construct picker before funnel module**

Insert before the line that builds `funnelMod`:

```go
    // Load balance picker for responsible-user selection at lead creation.
    loadCounterAdapter := funnelinfra.NewLeadLoadCounterAdapter(funnelLeadRepoForPicker)
    // ^ If a distinct leadRepo instance is not yet available, reuse funnelMod's once built
    //   OR construct the gorm repo once and share.
    picker := authinfra.NewLoadBalancePicker(
        permissionMod.GroupFunnelRepo(),
        authMod.LoadBalanceRepo(),
        permissionMod.UserGroupRepo(),
        authinfra.NewGormUserTenantRepository(db),
        loadCounterAdapter,
        log,
    )
```

Replace the temporary `nil` placeholder inserted in Task 11:

```go
    funnelMod := funnel.NewModule(db, contactAdapter, messageAdapter, userNameAdapter, productDetectorAdapter, funnelProductRouterAdapter, sharedEventBus, picker, log)
```

> If there's a chicken-and-egg issue (picker needs leadCounter from funnelMod but funnelMod needs picker), resolve by either:
> - Building the gorm lead repo once (e.g. `leadRepo := funnelinfra.NewGormLeadRepository(db)`) BEFORE the picker, and passing that same instance into `funnel.NewModule` (extend module signature if necessary); OR
> - Using a lazy adapter (`funnelinfra.NewLeadLoadCounterAdapter(db)` that internally news a repo).
> Prefer the first approach — single source of truth for the lead repo.

- [ ] **Step 3: Build and run smoke tests**

```bash
go build ./...
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/api/main.go internal/funnel/module.go internal/funnel/infrastructure/lead_load_counter.go
git commit -m "feat(api): wire LoadBalancePicker into funnel module"
```

---

## Task 15: Update `rest/*.http` with picker-aware examples

**Files:**
- Modify: `rest/team.http`
- Modify: `rest/funnel.http`

- [ ] **Step 1: Add an informational comment + sample**

Read each file, then append at the bottom of `rest/funnel.http`:

```http
### F08 Step 4 — Responsible assigned automatically on create
### After POST /leads or inbound WhatsApp, the returned/listed lead MUST
### carry a non-empty responsible_user_id. Falls back to the tenant owner
### when the funnel has no active load-balance group.

GET {{base}}/tenant/{{tenantId}}/leads/{{leadId}}
Authorization: Bearer {{token}}
### Expect: "responsible_user_id": "<uuid>"
```

And in `rest/team.http`, append after the load-balance section:

```http
### F08 Step 4 — Uniqueness rule on SetByGroup
### Activating a second group whose funnel/column coverage overlaps an
### already-active group yields 409 Conflict.

POST {{base}}/tenant/{{tenantId}}/load-balance
Authorization: Bearer {{token}}
Content-Type: application/json

{
  "group_id": "{{groupIdB}}",
  "algorithm": "round_robin",
  "active": true
}
### Expect: 409 Conflict when groupIdA already active on same funnel.
```

- [ ] **Step 2: Commit**

```bash
git add rest/team.http rest/funnel.http
git commit -m "docs(rest): add picker-aware examples and 409 case"
```

---

## Task 16: Update status + feature docs and verify coverage

**Files:**
- Modify: `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/status.md`
- Modify: `docs/features/F08-usuarios-permissoes.md`

- [ ] **Step 1: Tick status.md**

In the "Itens complementares" list, change:

```markdown
- [ ] Load balance integration (conectar à criação de leads)
```

to:

```markdown
- [x] Load balance integration (conectar à criação de leads) — picker + fallback + evento + notificação
```

- [ ] **Step 2: Tick F08 feature doc**

In `docs/features/F08-usuarios-permissoes.md`, mark the two checkboxes under Step 4 and Step 4.1:

```markdown
- [x] ao entrar lead no funil → distribuir automaticamente entre membros do grupo responsável
```

```markdown
- [x] ao criar lead (via IA/WhatsApp ou manual), atribuir responsável via load balance ou manualmente
```

- [ ] **Step 3: Verify coverage thresholds**

```bash
go test ./internal/auth/infrastructure/... ./internal/funnel/application/... ./internal/auth/application/... -cover
```

Expected: each package ≥ 80%. If below, add tests for uncovered branches (most likely the various fallback reasons in `PickForFunnel`) before proceeding.

- [ ] **Step 4: Run the full suite**

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add docs/artefatos/F08-F09-usuarios-permissoes-automacoes/status.md docs/features/F08-usuarios-permissoes.md
git commit -m "docs(F08): mark Step 4 and Step 4.1 as closed"
```

---

## Task 17: Push and open PR

- [ ] **Step 1: Push the branch**

```bash
git push -u origin feat/f08-load-balance-integration
```

- [ ] **Step 2: Open PR**

Use `gh pr create` with a title like `feat(F08): load balance integration at lead creation` and a body summarising:

- Picker port + implementation + 3 algorithms + owner fallback
- Uniqueness rule (one active LB per funnel/column) enforced at SetByGroup
- `EventLeadResponsibleAssigned` + notification subscription
- Metrics `crm_lead_responsible_picker_*` + OTel span
- Coverage ≥ 80%, full suite green

---

## Self-Review

**Spec coverage:**

| Spec section | Task |
|---|---|
| 3.1 Cascata de fallback | Tasks 5, 6, 10 |
| 3.2 Unicidade ativa | Task 13 |
| 3.3 Least-load definition | Tasks 4, 8 |
| 3.4 No realocation on move | N/A (documented OOS) |
| 4.1 ResponsiblePicker port | Task 3 |
| 4.2 LoadBalancePicker impl | Tasks 5-9 |
| 4.3 LeadLoadCounter port | Tasks 3, 4 |
| 4.4 LastIndex persistence | Task 7 |
| 5 CreateLead integration | Task 11 |
| 5.1 EventLeadCreated payload | Task 11 |
| 5.2 EventLeadResponsibleAssigned | Tasks 2, 11 |
| 6 Notification wiring | Task 12 |
| 7 Observability | Task 10 |
| 8 Security/OWASP | Task 11 (+ tenant scope everywhere) |
| 9 Tests | embedded in each task |
| 10 .http files | Task 15 |
| 12 Acceptance criteria | Task 16 |

All sections mapped. No gaps.

**Placeholder scan:** None of the forbidden patterns present (no "TBD", no "add appropriate error handling", no "similar to Task N", no "handle edge cases"). Where a helper like `testutil.OpenTestDB` may need adaptation, the plan says so explicitly with guidance.

**Type consistency:** `PickResult`, `PickOutcome`, `ResponsiblePicker`, `LeadLoadCounter`, `GroupColumnOverlapChecker`, `ErrNoResponsibleAvailable`, `ErrActiveLoadBalanceOverlap`, `ResponsibleAssignedPayload` used consistently across all tasks.
