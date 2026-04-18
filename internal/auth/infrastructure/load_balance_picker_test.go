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

type fakeGroupFunnelRepo struct {
	byFunnel map[string][]permdomain.GroupFunnel
}

func (f *fakeGroupFunnelRepo) FindByFunnelID(_ context.Context, funnelID string) ([]permdomain.GroupFunnel, error) {
	return f.byFunnel[funnelID], nil
}
func (f *fakeGroupFunnelRepo) CreateOrUpdate(context.Context, *permdomain.GroupFunnel) error {
	return nil
}
func (f *fakeGroupFunnelRepo) FindByGroupID(context.Context, string) ([]permdomain.GroupFunnel, error) {
	return nil, nil
}
func (f *fakeGroupFunnelRepo) FindByFunnelAndColumn(context.Context, string, string) ([]permdomain.GroupFunnel, error) {
	return nil, nil
}
func (f *fakeGroupFunnelRepo) Delete(context.Context, string, string) error { return nil }

type fakeLoadBalanceRepo struct {
	byGroup     map[string]*authdomain.LoadBalanceConfig
	updateErr   error
	lastUpdated *authdomain.LoadBalanceConfig
}

func (f *fakeLoadBalanceRepo) FindByGroupID(_ context.Context, _, groupID string) (*authdomain.LoadBalanceConfig, error) {
	if cfg, ok := f.byGroup[groupID]; ok {
		return cfg, nil
	}
	return nil, authdomain.ErrLoadBalanceNotFound
}
func (f *fakeLoadBalanceRepo) CreateOrUpdate(context.Context, *authdomain.LoadBalanceConfig) error {
	return nil
}
func (f *fakeLoadBalanceRepo) FindByTenantID(context.Context, string) ([]*authdomain.LoadBalanceConfig, error) {
	return nil, nil
}
func (f *fakeLoadBalanceRepo) Update(_ context.Context, cfg *authdomain.LoadBalanceConfig) error {
	f.lastUpdated = cfg
	return f.updateErr
}

type fakeUserGroupRepo struct {
	byGroup map[string][]permdomain.UserGroup
}

func (f *fakeUserGroupRepo) FindByGroupID(_ context.Context, gid string) ([]permdomain.UserGroup, error) {
	return f.byGroup[gid], nil
}
func (f *fakeUserGroupRepo) Create(context.Context, *permdomain.UserGroup) error { return nil }
func (f *fakeUserGroupRepo) Delete(context.Context, string, string) error       { return nil }
func (f *fakeUserGroupRepo) FindByUserAndTenant(context.Context, string, string) ([]permdomain.UserGroup, error) {
	return nil, nil
}
func (f *fakeUserGroupRepo) Exists(context.Context, string, string) (bool, error) {
	return false, nil
}

type fakeUserTenantRepo struct {
	ownerByTenant map[string]string
	memberActive  map[string]bool
}

func (f *fakeUserTenantRepo) FindByTenantID(_ context.Context, tenantID string) ([]*authdomain.UserTenant, error) {
	owner := f.ownerByTenant[tenantID]
	if owner == "" {
		return []*authdomain.UserTenant{}, nil
	}
	return []*authdomain.UserTenant{{UserID: owner, TenantID: tenantID, IsOwner: true}}, nil
}
func (f *fakeUserTenantRepo) FindByUserAndTenant(_ context.Context, uid, tid string) (*authdomain.UserTenant, error) {
	if f.memberActive == nil {
		return &authdomain.UserTenant{UserID: uid, TenantID: tid}, nil
	}
	if _, ok := f.memberActive[uid]; !ok {
		return nil, errors.New("not a member")
	}
	return &authdomain.UserTenant{UserID: uid, TenantID: tid}, nil
}

// stubs for unused methods
func (f *fakeUserTenantRepo) Associate(context.Context, string, string) error { return nil }
func (f *fakeUserTenantRepo) FindTenantIDsByUserID(context.Context, string) ([]string, error) {
	return nil, nil
}
func (f *fakeUserTenantRepo) UpdateIsOwner(context.Context, string, string, bool) error { return nil }
func (f *fakeUserTenantRepo) UpdateWhatsAppID(context.Context, string, string, string) error {
	return nil
}
func (f *fakeUserTenantRepo) RemoveFromTenant(context.Context, string, string) error { return nil }
func (f *fakeUserTenantRepo) IsOwner(context.Context, string, string) (bool, error) {
	return false, nil
}

type fakeLoadCounter struct {
	counts map[string]int
}

func (f *fakeLoadCounter) CountActiveByUsers(context.Context, string, []string) (map[string]int, error) {
	if f.counts == nil {
		return map[string]int{}, nil
	}
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

func TestLoadBalancePicker_RoundRobin_CyclesThroughMembers(t *testing.T) {
	cfg := &authdomain.LoadBalanceConfig{
		ID: "lb1", TenantID: "t1", GroupID: "g1",
		Algorithm: authdomain.AlgorithmRoundRobin, Active: true, LastIndex: 0,
	}
	lbRepo := &fakeLoadBalanceRepo{byGroup: map[string]*authdomain.LoadBalanceConfig{"g1": cfg}}
	picker := infrastructure.NewLoadBalancePicker(
		&fakeGroupFunnelRepo{byFunnel: map[string][]permdomain.GroupFunnel{
			"f1": {{ID: "gf1", GroupID: "g1", FunnelID: "f1"}},
		}},
		lbRepo,
		&fakeUserGroupRepo{byGroup: map[string][]permdomain.UserGroup{
			"g1": {{UserID: "u-c"}, {UserID: "u-a"}, {UserID: "u-b"}},
		}},
		&fakeUserTenantRepo{
			ownerByTenant: map[string]string{"t1": "owner-1"},
			memberActive:  map[string]bool{"u-a": true, "u-b": true, "u-c": true},
		},
		&fakeLoadCounter{},
		zap.NewNop(),
	)

	r1, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")
	require.NoError(t, err)
	require.Equal(t, "u-a", r1.UserID)

	r2, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")
	require.NoError(t, err)
	require.Equal(t, "u-b", r2.UserID)

	r3, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")
	require.NoError(t, err)
	require.Equal(t, "u-c", r3.UserID)

	r4, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")
	require.NoError(t, err)
	require.Equal(t, "u-a", r4.UserID)

	require.NotNil(t, lbRepo.lastUpdated, "LastIndex must be persisted via Update")
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
