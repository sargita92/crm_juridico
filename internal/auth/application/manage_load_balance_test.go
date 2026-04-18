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

func TestSetByGroup_InvalidAlgorithmOnUpdate(t *testing.T) {
	repo := new(mockLBRepo)
	chk := new(mockGroupChecker)
	uc := NewManageLoadBalanceUseCase(repo, chk)

	chk.On("BelongsToTenant", mock.Anything, "t1", "g1").Return(true, nil)

	_, err := uc.SetByGroup(context.Background(), SetLoadBalanceInput{
		TenantID: "t1", GroupID: "g1", Algorithm: "weird", Active: true,
	})
	require.ErrorIs(t, err, domain.ErrInvalidAlgorithm)
	repo.AssertNotCalled(t, "CreateOrUpdate", mock.Anything, mock.Anything)
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
