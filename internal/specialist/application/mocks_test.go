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

func (m *mockSpecialistTenantRepo) FindDefaultByTenantID(_ context.Context, tenantID string) (string, error) {
	for specID, tenants := range m.associations {
		if tenants[tenantID] {
			return specID, nil
		}
	}
	return "", domain.ErrSpecialistNotFound
}

func (m *mockSpecialistTenantRepo) SetDefault(_ context.Context, specialistID, tenantID string) error {
	if m.associations[specialistID] == nil || !m.associations[specialistID][tenantID] {
		return domain.ErrTenantNotAssociated
	}
	return nil
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

// --- mockGuardrailRepo ---

type mockGuardrailRepo struct {
	guardrails map[string]*domain.Guardrail
}

func newMockGuardrailRepo() *mockGuardrailRepo {
	return &mockGuardrailRepo{guardrails: make(map[string]*domain.Guardrail)}
}

func (m *mockGuardrailRepo) Create(_ context.Context, g *domain.Guardrail) error {
	m.guardrails[g.ID] = g
	return nil
}

func (m *mockGuardrailRepo) FindByID(_ context.Context, id string) (*domain.Guardrail, error) {
	if g, ok := m.guardrails[id]; ok {
		return g, nil
	}
	return nil, domain.ErrGuardrailNotFound
}

func (m *mockGuardrailRepo) Update(_ context.Context, g *domain.Guardrail) error {
	if _, ok := m.guardrails[g.ID]; !ok {
		return domain.ErrGuardrailNotFound
	}
	m.guardrails[g.ID] = g
	return nil
}

func (m *mockGuardrailRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.guardrails[id]; !ok {
		return domain.ErrGuardrailNotFound
	}
	delete(m.guardrails, id)
	return nil
}

func (m *mockGuardrailRepo) FindBySpecialistID(_ context.Context, specialistID string) ([]domain.Guardrail, error) {
	var result []domain.Guardrail
	for _, g := range m.guardrails {
		if g.SpecialistID == specialistID {
			result = append(result, *g)
		}
	}
	return result, nil
}

func (m *mockGuardrailRepo) addGuardrail(g *domain.Guardrail) {
	m.guardrails[g.ID] = g
}

// --- mockStepRepo ---

type mockStepRepo struct {
	steps map[string]*domain.Step
}

func newMockStepRepo() *mockStepRepo {
	return &mockStepRepo{steps: make(map[string]*domain.Step)}
}

func (m *mockStepRepo) Create(_ context.Context, s *domain.Step) error {
	m.steps[s.ID] = s
	return nil
}

func (m *mockStepRepo) FindByID(_ context.Context, id string) (*domain.Step, error) {
	if s, ok := m.steps[id]; ok {
		return s, nil
	}
	return nil, domain.ErrStepNotFound
}

func (m *mockStepRepo) Update(_ context.Context, s *domain.Step) error {
	if _, ok := m.steps[s.ID]; !ok {
		return domain.ErrStepNotFound
	}
	m.steps[s.ID] = s
	return nil
}

func (m *mockStepRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.steps[id]; !ok {
		return domain.ErrStepNotFound
	}
	delete(m.steps, id)
	return nil
}

func (m *mockStepRepo) FindBySpecialistID(_ context.Context, specialistID string) ([]domain.Step, error) {
	var result []domain.Step
	for _, s := range m.steps {
		if s.SpecialistID == specialistID {
			result = append(result, *s)
		}
	}
	// sort by OrderIndex to maintain consistent ordering
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].OrderIndex > result[j].OrderIndex {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, nil
}

func (m *mockStepRepo) GetMaxOrderIndex(_ context.Context, specialistID string) (int, error) {
	max := 0
	for _, s := range m.steps {
		if s.SpecialistID == specialistID && s.OrderIndex > max {
			max = s.OrderIndex
		}
	}
	return max, nil
}

func (m *mockStepRepo) ReorderAfterDelete(_ context.Context, specialistID string, deletedOrder int) error {
	for _, s := range m.steps {
		if s.SpecialistID == specialistID && s.OrderIndex > deletedOrder {
			s.OrderIndex--
		}
	}
	return nil
}

func (m *mockStepRepo) SwapOrder(_ context.Context, step1ID string, order1 int, step2ID string, order2 int) error {
	s1, ok1 := m.steps[step1ID]
	s2, ok2 := m.steps[step2ID]
	if !ok1 || !ok2 {
		return domain.ErrStepNotFound
	}
	s1.OrderIndex = order2
	s2.OrderIndex = order1
	return nil
}

func (m *mockStepRepo) addStep(s *domain.Step) {
	m.steps[s.ID] = s
}

// --- mockScoringConfigRepo ---

type mockScoringConfigRepo struct {
	configs map[string]*domain.ScoringConfig
}

func newMockScoringConfigRepo() *mockScoringConfigRepo {
	return &mockScoringConfigRepo{configs: make(map[string]*domain.ScoringConfig)}
}

func (m *mockScoringConfigRepo) CreateOrUpdate(_ context.Context, config *domain.ScoringConfig) error {
	m.configs[config.SpecialistID] = config
	return nil
}

func (m *mockScoringConfigRepo) FindBySpecialistID(_ context.Context, specialistID string) (*domain.ScoringConfig, error) {
	if c, ok := m.configs[specialistID]; ok {
		return c, nil
	}
	return nil, domain.ErrScoringConfigNotFound
}

func (m *mockScoringConfigRepo) addScoringConfig(c *domain.ScoringConfig) {
	m.configs[c.SpecialistID] = c
}
