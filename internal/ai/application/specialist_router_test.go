package application

import (
	"context"
	"errors"
	"testing"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
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

// --- tests ---

func TestSpecialistRouter_RouteByPhone(t *testing.T) {
	router := NewSpecialistRouter(
		&mockPhoneFinder{productID: "prod-1"},
		&mockSPFinder{results: []domain.SpecialistProduct{{SpecialistID: "spec-1", ProductID: "prod-1"}}},
		&mockProductDetector{},
		&mockDefaultSPFinder{specialistID: "spec-default"},
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
	)

	_, _, err := router.Route(context.Background(), "tenant-1", "+5511999999999", "oi")
	assert.ErrorIs(t, err, ErrNoSpecialistAvailable)
}
