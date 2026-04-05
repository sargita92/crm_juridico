package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

// --- mockSpecialistRepo ---

type mockSpecialistRepo struct {
	specialists map[string]*domain.Specialist
}

func newMockSpecialistRepo() *mockSpecialistRepo {
	return &mockSpecialistRepo{
		specialists: make(map[string]*domain.Specialist),
	}
}

func (m *mockSpecialistRepo) Create(_ context.Context, s *domain.Specialist) error {
	m.specialists[s.ID] = s
	return nil
}

func (m *mockSpecialistRepo) FindByID(_ context.Context, id string) (*domain.Specialist, error) {
	if s, ok := m.specialists[id]; ok {
		return s, nil
	}
	return nil, domain.ErrSpecialistNotFound
}

func (m *mockSpecialistRepo) Update(_ context.Context, s *domain.Specialist) error {
	if _, ok := m.specialists[s.ID]; !ok {
		return domain.ErrSpecialistNotFound
	}
	m.specialists[s.ID] = s
	return nil
}

func (m *mockSpecialistRepo) FindWithFilter(_ context.Context, filter domain.SpecialistFilter) (*domain.SpecialistList, error) {
	var result []domain.Specialist
	for _, s := range m.specialists {
		if filter.Status != "" && s.Status != filter.Status {
			continue
		}
		result = append(result, *s)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}

	return &domain.SpecialistList{
		Specialists: result,
		Total:       int64(len(result)),
		Page:        page,
		Limit:       limit,
	}, nil
}

func (m *mockSpecialistRepo) addSpecialist(s *domain.Specialist) {
	m.specialists[s.ID] = s
}

// --- mockSpecialistTenantRepo ---

type mockSpecialistTenantRepo struct {
	associations map[string]map[string]bool // specialistID -> tenantID -> true
}

func newMockSpecialistTenantRepo() *mockSpecialistTenantRepo {
	return &mockSpecialistTenantRepo{
		associations: make(map[string]map[string]bool),
	}
}

func (m *mockSpecialistTenantRepo) Associate(_ context.Context, specialistID, tenantID string) error {
	if m.associations[specialistID] == nil {
		m.associations[specialistID] = make(map[string]bool)
	}
	if m.associations[specialistID][tenantID] {
		return domain.ErrTenantAlreadyAssociated
	}
	m.associations[specialistID][tenantID] = true
	return nil
}

func (m *mockSpecialistTenantRepo) Dissociate(_ context.Context, specialistID, tenantID string) error {
	if m.associations[specialistID] == nil || !m.associations[specialistID][tenantID] {
		return domain.ErrTenantNotAssociated
	}
	delete(m.associations[specialistID], tenantID)
	return nil
}

func (m *mockSpecialistTenantRepo) FindTenantIDsBySpecialistID(_ context.Context, specialistID string) ([]string, error) {
	var ids []string
	for tenantID := range m.associations[specialistID] {
		ids = append(ids, tenantID)
	}
	return ids, nil
}

func (m *mockSpecialistTenantRepo) FindSpecialistIDsByTenantID(_ context.Context, tenantID string) ([]string, error) {
	var ids []string
	for specID, tenants := range m.associations {
		if tenants[tenantID] {
			ids = append(ids, specID)
		}
	}
	return ids, nil
}

func (m *mockSpecialistTenantRepo) Exists(_ context.Context, specialistID, tenantID string) (bool, error) {
	if m.associations[specialistID] == nil {
		return false, nil
	}
	return m.associations[specialistID][tenantID], nil
}

// --- mockTenantRepo ---

type mockTenantRepo struct {
	tenants map[string]*tenantdomain.Tenant
}

func newMockTenantRepo() *mockTenantRepo {
	return &mockTenantRepo{
		tenants: make(map[string]*tenantdomain.Tenant),
	}
}

func (m *mockTenantRepo) Create(_ context.Context, t *tenantdomain.Tenant) error {
	m.tenants[t.ID] = t
	return nil
}

func (m *mockTenantRepo) FindByID(_ context.Context, id string) (*tenantdomain.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, tenantdomain.ErrTenantNotFound
}

func (m *mockTenantRepo) FindByIDs(_ context.Context, ids []string) ([]tenantdomain.Tenant, error) {
	var result []tenantdomain.Tenant
	for _, id := range ids {
		if t, ok := m.tenants[id]; ok {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockTenantRepo) FindAll(_ context.Context) ([]tenantdomain.Tenant, error) {
	var result []tenantdomain.Tenant
	for _, t := range m.tenants {
		if t.Status == tenantdomain.TenantStatusActive {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockTenantRepo) Update(_ context.Context, t *tenantdomain.Tenant) error {
	m.tenants[t.ID] = t
	return nil
}

func (m *mockTenantRepo) FindWithFilter(_ context.Context, filter tenantdomain.TenantFilter) (*tenantdomain.TenantList, error) {
	var result []tenantdomain.Tenant
	for _, t := range m.tenants {
		if filter.Status != "" && t.Status != filter.Status {
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
	return &tenantdomain.TenantList{
		Tenants: result,
		Total:   int64(len(result)),
		Page:    page,
		Limit:   limit,
	}, nil
}

func (m *mockTenantRepo) FindByDocument(_ context.Context, document string) (*tenantdomain.Tenant, error) {
	for _, t := range m.tenants {
		if t.Document == document {
			return t, nil
		}
	}
	return nil, tenantdomain.ErrTenantNotFound
}

func (m *mockTenantRepo) addTenant(t *tenantdomain.Tenant) {
	m.tenants[t.ID] = t
}
