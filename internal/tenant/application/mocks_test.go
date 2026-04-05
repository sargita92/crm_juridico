package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type mockTenantRepo struct {
	tenants    map[string]*domain.Tenant
	createErr  error
	updateErr  error
	documents  map[string]*domain.Tenant
}

func newMockTenantRepo() *mockTenantRepo {
	return &mockTenantRepo{
		tenants:   make(map[string]*domain.Tenant),
		documents: make(map[string]*domain.Tenant),
	}
}

func (m *mockTenantRepo) Create(_ context.Context, tenant *domain.Tenant) error {
	if m.createErr != nil {
		return m.createErr
	}
	if _, exists := m.documents[tenant.Document]; exists {
		return domain.ErrTenantDocumentExists
	}
	m.tenants[tenant.ID] = tenant
	m.documents[tenant.Document] = tenant
	return nil
}

func (m *mockTenantRepo) FindByID(_ context.Context, id string) (*domain.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, domain.ErrTenantNotFound
}

func (m *mockTenantRepo) FindByIDs(_ context.Context, ids []string) ([]domain.Tenant, error) {
	var result []domain.Tenant
	for _, id := range ids {
		if t, ok := m.tenants[id]; ok {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockTenantRepo) FindAll(_ context.Context) ([]domain.Tenant, error) {
	var result []domain.Tenant
	for _, t := range m.tenants {
		if t.Status == domain.TenantStatusActive {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockTenantRepo) Update(_ context.Context, tenant *domain.Tenant) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	// Check for duplicate document (excluding self)
	for _, t := range m.tenants {
		if t.ID != tenant.ID && t.Document == tenant.Document {
			return domain.ErrTenantDocumentExists
		}
	}
	// Update documents index
	for doc, t := range m.documents {
		if t.ID == tenant.ID {
			delete(m.documents, doc)
			break
		}
	}
	m.tenants[tenant.ID] = tenant
	m.documents[tenant.Document] = tenant
	return nil
}

func (m *mockTenantRepo) FindWithFilter(_ context.Context, filter domain.TenantFilter) (*domain.TenantList, error) {
	var result []domain.Tenant
	for _, t := range m.tenants {
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if filter.Type != "" && t.Type != filter.Type {
			continue
		}
		result = append(result, *t)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}

	return &domain.TenantList{
		Tenants: result,
		Total:   int64(len(result)),
		Page:    page,
		Limit:   limit,
	}, nil
}

func (m *mockTenantRepo) FindByDocument(_ context.Context, document string) (*domain.Tenant, error) {
	if t, ok := m.documents[document]; ok {
		return t, nil
	}
	return nil, domain.ErrTenantNotFound
}

func (m *mockTenantRepo) addTenant(tenant *domain.Tenant) {
	m.tenants[tenant.ID] = tenant
	m.documents[tenant.Document] = tenant
}
