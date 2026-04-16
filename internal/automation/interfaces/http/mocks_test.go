package http

import (
	"context"
	"errors"

	automationdomain "github.com/sasrgita/crm-juridico/internal/automation/domain"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// --- Automation repositories ---

type mockAutomationRepo struct {
	items map[string]*automationdomain.Automation
}

func newMockAutomationRepo() *mockAutomationRepo {
	return &mockAutomationRepo{items: make(map[string]*automationdomain.Automation)}
}

func (m *mockAutomationRepo) Create(_ context.Context, a *automationdomain.Automation) error {
	cp := *a
	m.items[a.ID] = &cp
	return nil
}

func (m *mockAutomationRepo) FindByID(_ context.Context, id string) (*automationdomain.Automation, error) {
	if a, ok := m.items[id]; ok {
		cp := *a
		return &cp, nil
	}
	return nil, automationdomain.ErrAutomationNotFound
}

func (m *mockAutomationRepo) Update(_ context.Context, a *automationdomain.Automation) error {
	if _, ok := m.items[a.ID]; !ok {
		return automationdomain.ErrAutomationNotFound
	}
	cp := *a
	m.items[a.ID] = &cp
	return nil
}

func (m *mockAutomationRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.items[id]; !ok {
		return automationdomain.ErrAutomationNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *mockAutomationRepo) FindByTenantAndColumn(_ context.Context, tenantID, columnID string) ([]automationdomain.Automation, error) {
	var out []automationdomain.Automation
	for _, a := range m.items {
		if a.TenantID == tenantID && a.ColumnID == columnID && a.Active {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (m *mockAutomationRepo) FindByFunnelID(_ context.Context, tenantID, funnelID string) ([]automationdomain.Automation, error) {
	var out []automationdomain.Automation
	for _, a := range m.items {
		if a.TenantID == tenantID && a.FunnelID == funnelID {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (m *mockAutomationRepo) FindActiveByType(_ context.Context, t automationdomain.AutomationType) ([]automationdomain.Automation, error) {
	var out []automationdomain.Automation
	for _, a := range m.items {
		if a.Type == t && a.Active {
			out = append(out, *a)
		}
	}
	return out, nil
}

type mockLogRepo struct {
	logs []automationdomain.ExecutionLog
}

func (m *mockLogRepo) Create(_ context.Context, log *automationdomain.ExecutionLog) error {
	m.logs = append(m.logs, *log)
	return nil
}

func (m *mockLogRepo) FindByAutomationID(_ context.Context, automationID string, limit, offset int) ([]automationdomain.ExecutionLog, error) {
	var matched []automationdomain.ExecutionLog
	for _, l := range m.logs {
		if l.AutomationID == automationID {
			matched = append(matched, l)
		}
	}
	if offset >= len(matched) {
		return nil, nil
	}
	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}
	return matched[offset:end], nil
}

// --- Funnel repositories ---

type mockFunnelRepo struct {
	funnels map[string]*funneldomain.Funnel
}

func newMockFunnelRepo() *mockFunnelRepo {
	return &mockFunnelRepo{funnels: make(map[string]*funneldomain.Funnel)}
}

func (m *mockFunnelRepo) Create(_ context.Context, f *funneldomain.Funnel) error {
	m.funnels[f.ID] = f
	return nil
}

func (m *mockFunnelRepo) FindByID(_ context.Context, id string) (*funneldomain.Funnel, error) {
	if f, ok := m.funnels[id]; ok {
		return f, nil
	}
	return nil, funneldomain.ErrFunnelNotFound
}

func (m *mockFunnelRepo) Update(_ context.Context, _ *funneldomain.Funnel) error { return nil }

func (m *mockFunnelRepo) FindByTenantID(_ context.Context, tenantID string) ([]funneldomain.Funnel, error) {
	var out []funneldomain.Funnel
	for _, f := range m.funnels {
		if f.TenantID == tenantID {
			out = append(out, *f)
		}
	}
	return out, nil
}

func (m *mockFunnelRepo) FindDefaultByTenantID(_ context.Context, _ string) (*funneldomain.Funnel, error) {
	return nil, funneldomain.ErrFunnelNotFound
}

type mockColumnRepo struct {
	columns map[string]*funneldomain.Column
}

func newMockColumnRepo() *mockColumnRepo {
	return &mockColumnRepo{columns: make(map[string]*funneldomain.Column)}
}

func (m *mockColumnRepo) Create(_ context.Context, c *funneldomain.Column) error {
	m.columns[c.ID] = c
	return nil
}

func (m *mockColumnRepo) FindByID(_ context.Context, id string) (*funneldomain.Column, error) {
	if c, ok := m.columns[id]; ok {
		return c, nil
	}
	return nil, funneldomain.ErrColumnNotFound
}

func (m *mockColumnRepo) Update(_ context.Context, _ *funneldomain.Column) error { return nil }
func (m *mockColumnRepo) Delete(_ context.Context, _ string) error               { return nil }

func (m *mockColumnRepo) FindByFunnelID(_ context.Context, funnelID string) ([]funneldomain.Column, error) {
	var out []funneldomain.Column
	for _, c := range m.columns {
		if c.FunnelID == funnelID {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (m *mockColumnRepo) FindEntryByFunnelID(_ context.Context, _ string) (*funneldomain.Column, error) {
	return nil, funneldomain.ErrColumnNotFound
}

func (m *mockColumnRepo) CountByFunnelID(_ context.Context, funnelID string) (int, error) {
	count := 0
	for _, c := range m.columns {
		if c.FunnelID == funnelID {
			count++
		}
	}
	return count, nil
}

func (m *mockColumnRepo) GetMaxOrderIndex(_ context.Context, _ string) (int, error) { return 0, nil }
func (m *mockColumnRepo) SwapOrder(_ context.Context, _ string, _ int, _ string, _ int) error {
	return nil
}

type mockLeadRepo struct {
	leads map[string]*funneldomain.Lead
}

func newMockLeadRepo() *mockLeadRepo {
	return &mockLeadRepo{leads: make(map[string]*funneldomain.Lead)}
}

func (m *mockLeadRepo) Create(_ context.Context, l *funneldomain.Lead) error {
	m.leads[l.ID] = l
	return nil
}

func (m *mockLeadRepo) FindByID(_ context.Context, id string) (*funneldomain.Lead, error) {
	if l, ok := m.leads[id]; ok {
		return l, nil
	}
	return nil, funneldomain.ErrLeadNotFound
}

func (m *mockLeadRepo) Update(_ context.Context, _ *funneldomain.Lead) error { return nil }

func (m *mockLeadRepo) FindByContactAndTenant(_ context.Context, _, _ string) (*funneldomain.Lead, error) {
	return nil, funneldomain.ErrLeadNotFound
}

func (m *mockLeadRepo) FindByConversationID(_ context.Context, _ string) (*funneldomain.Lead, error) {
	return nil, funneldomain.ErrLeadNotFound
}

func (m *mockLeadRepo) FindByFunnelID(_ context.Context, _ string, _ funneldomain.LeadFilter) (*funneldomain.LeadList, error) {
	return &funneldomain.LeadList{}, nil
}

func (m *mockLeadRepo) CountByColumnID(_ context.Context, _ string) (int, error) { return 0, nil }

func (m *mockLeadRepo) FindByTenantAndSearch(_ context.Context, _, _ string, _ int) ([]funneldomain.Lead, error) {
	return nil, nil
}

type mockContactProvider struct {
	contacts map[string]funneldomain.ContactInfo
}

func (m *mockContactProvider) FindByID(_ context.Context, contactID string) (funneldomain.ContactInfo, error) {
	if info, ok := m.contacts[contactID]; ok {
		return info, nil
	}
	return funneldomain.ContactInfo{}, errors.New("contact not found")
}

// --- Specialist repositories ---

type mockSpecialistRepo struct {
	items map[string]*specialistdomain.Specialist
}

func newMockSpecialistRepo() *mockSpecialistRepo {
	return &mockSpecialistRepo{items: make(map[string]*specialistdomain.Specialist)}
}

func (m *mockSpecialistRepo) Create(_ context.Context, s *specialistdomain.Specialist) error {
	m.items[s.ID] = s
	return nil
}

func (m *mockSpecialistRepo) FindByID(_ context.Context, id string) (*specialistdomain.Specialist, error) {
	if s, ok := m.items[id]; ok {
		return s, nil
	}
	return nil, errors.New("specialist not found")
}

func (m *mockSpecialistRepo) Update(_ context.Context, _ *specialistdomain.Specialist) error {
	return nil
}

func (m *mockSpecialistRepo) FindWithFilter(_ context.Context, _ specialistdomain.SpecialistFilter) (*specialistdomain.SpecialistList, error) {
	return &specialistdomain.SpecialistList{}, nil
}

type mockSpecialistTenantRepo struct {
	byTenant map[string][]string
}

func newMockSpecialistTenantRepo() *mockSpecialistTenantRepo {
	return &mockSpecialistTenantRepo{byTenant: make(map[string][]string)}
}

func (m *mockSpecialistTenantRepo) Associate(_ context.Context, specialistID, tenantID string) error {
	m.byTenant[tenantID] = append(m.byTenant[tenantID], specialistID)
	return nil
}

func (m *mockSpecialistTenantRepo) Dissociate(_ context.Context, _, _ string) error { return nil }

func (m *mockSpecialistTenantRepo) FindTenantIDsBySpecialistID(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (m *mockSpecialistTenantRepo) FindBySpecialistID(_ context.Context, _ string) ([]specialistdomain.SpecialistTenant, error) {
	return nil, nil
}

func (m *mockSpecialistTenantRepo) FindSpecialistIDsByTenantID(_ context.Context, tenantID string) ([]string, error) {
	return m.byTenant[tenantID], nil
}

func (m *mockSpecialistTenantRepo) Exists(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (m *mockSpecialistTenantRepo) FindDefaultByTenantID(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (m *mockSpecialistTenantRepo) SetDefault(_ context.Context, _, _ string) error { return nil }
