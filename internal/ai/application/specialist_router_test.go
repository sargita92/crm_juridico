package application

import (
	"context"
	"errors"
	"testing"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock implementations ---

type mockPhoneFinder struct {
	productID string
	err       error
}

func (m *mockPhoneFinder) FindProductIDByPhone(_ context.Context, _ string) (string, error) {
	return m.productID, m.err
}

type mockSPFinder struct {
	results []domain.SpecialistProduct
	err     error
}

func (m *mockSPFinder) FindByProductID(_ context.Context, _ string) ([]domain.SpecialistProduct, error) {
	return m.results, m.err
}

type mockProductDetector struct {
	productID string
	found     bool
	err       error
}

func (m *mockProductDetector) DetectFromMessage(_ context.Context, _, _ string) (string, bool, error) {
	return m.productID, m.found, m.err
}

type mockDefaultSPFinder struct {
	specialistID string
	err          error
}

func (m *mockDefaultSPFinder) FindDefaultByTenantID(_ context.Context, _ string) (string, error) {
	return m.specialistID, m.err
}

// mockRouterSpecFinder returns active specialists by default; any ID listed in
// `inactive` comes back deactivated.
type mockRouterSpecFinder struct {
	inactive map[string]bool
}

func (m *mockRouterSpecFinder) FindByID(_ context.Context, id string) (*specDomain.Specialist, error) {
	s, _ := specDomain.NewSpecialist(id, "n", "d", "p")
	if m.inactive[id] {
		_ = s.Deactivate()
	}
	return s, nil
}

// --- tests ---

func TestSpecialistRouter_RouteByPhone(t *testing.T) {
	router := NewSpecialistRouter(
		&mockPhoneFinder{productID: "prod-1"},
		&mockSPFinder{results: []domain.SpecialistProduct{{SpecialistID: "spec-1", ProductID: "prod-1"}}},
		&mockProductDetector{},
		&mockDefaultSPFinder{specialistID: "spec-default"},
		&mockRouterSpecFinder{},
	)

	sid, pid, err := router.Route(context.Background(), "tenant-1", "+5511999999999", "oi")
	require.NoError(t, err)
	assert.Equal(t, "spec-1", sid)
	assert.Equal(t, "prod-1", pid)
}

func TestSpecialistRouter_RouteByMessage(t *testing.T) {
	router := NewSpecialistRouter(
		&mockPhoneFinder{err: errors.New("not found")},
		&mockSPFinder{results: []domain.SpecialistProduct{{SpecialistID: "spec-2", ProductID: "prod-2"}}},
		&mockProductDetector{productID: "prod-2", found: true},
		&mockDefaultSPFinder{specialistID: "spec-default"},
		&mockRouterSpecFinder{},
	)

	sid, pid, err := router.Route(context.Background(), "tenant-1", "+5511999999999", "quero contratar seguro")
	require.NoError(t, err)
	assert.Equal(t, "spec-2", sid)
	assert.Equal(t, "prod-2", pid)
}

func TestSpecialistRouter_RouteDefault(t *testing.T) {
	router := NewSpecialistRouter(
		&mockPhoneFinder{err: errors.New("not found")},
		&mockSPFinder{err: errors.New("not found")},
		&mockProductDetector{found: false},
		&mockDefaultSPFinder{specialistID: "spec-default"},
		&mockRouterSpecFinder{},
	)

	sid, pid, err := router.Route(context.Background(), "tenant-1", "+5511999999999", "oi")
	require.NoError(t, err)
	assert.Equal(t, "spec-default", sid)
	assert.Equal(t, "", pid)
}

func TestSpecialistRouter_NoSpecialistFound(t *testing.T) {
	router := NewSpecialistRouter(
		&mockPhoneFinder{err: errors.New("not found")},
		&mockSPFinder{err: errors.New("not found")},
		&mockProductDetector{found: false},
		&mockDefaultSPFinder{err: errors.New("not found")},
		&mockRouterSpecFinder{},
	)

	_, _, err := router.Route(context.Background(), "tenant-1", "+5511999999999", "oi")
	assert.ErrorIs(t, err, ErrNoSpecialistAvailable)
}

// An inactive specialist mapped to the product must be skipped in favour of the
// first active one — deactivating a specialist takes it out of routing.
func TestSpecialistRouter_SkipsInactiveInProduct(t *testing.T) {
	router := NewSpecialistRouter(
		&mockPhoneFinder{productID: "prod-1"},
		&mockSPFinder{results: []domain.SpecialistProduct{
			{SpecialistID: "spec-inactive", ProductID: "prod-1"},
			{SpecialistID: "spec-active", ProductID: "prod-1"},
		}},
		&mockProductDetector{},
		&mockDefaultSPFinder{specialistID: "spec-default"},
		&mockRouterSpecFinder{inactive: map[string]bool{"spec-inactive": true}},
	)

	sid, _, err := router.Route(context.Background(), "tenant-1", "+5511999999999", "oi")
	require.NoError(t, err)
	assert.Equal(t, "spec-active", sid)
}

// When the tenant default is inactive and nothing else matches, no specialist is
// routed (rather than using the inactive one).
func TestSpecialistRouter_InactiveDefault_NotUsed(t *testing.T) {
	router := NewSpecialistRouter(
		&mockPhoneFinder{err: errors.New("not found")},
		&mockSPFinder{err: errors.New("not found")},
		&mockProductDetector{found: false},
		&mockDefaultSPFinder{specialistID: "spec-default"},
		&mockRouterSpecFinder{inactive: map[string]bool{"spec-default": true}},
	)

	_, _, err := router.Route(context.Background(), "tenant-1", "+5511999999999", "oi")
	assert.ErrorIs(t, err, ErrNoSpecialistAvailable)
}
