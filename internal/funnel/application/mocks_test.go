package application

import (
	"context"
	"errors"
	"strings"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

// --- Mock FunnelRepository ---

type mockFunnelRepo struct {
	funnels   map[string]*domain.Funnel
	byTenant  map[string][]*domain.Funnel
	createErr error
	updateErr error
}

func newMockFunnelRepo() *mockFunnelRepo {
	return &mockFunnelRepo{
		funnels:  make(map[string]*domain.Funnel),
		byTenant: make(map[string][]*domain.Funnel),
	}
}

func (m *mockFunnelRepo) Create(_ context.Context, f *domain.Funnel) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.funnels[f.ID] = f
	m.byTenant[f.TenantID] = append(m.byTenant[f.TenantID], f)
	return nil
}

func (m *mockFunnelRepo) FindByID(_ context.Context, id string) (*domain.Funnel, error) {
	if f, ok := m.funnels[id]; ok {
		return f, nil
	}
	return nil, domain.ErrFunnelNotFound
}

func (m *mockFunnelRepo) Update(_ context.Context, f *domain.Funnel) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.funnels[f.ID] = f
	return nil
}

func (m *mockFunnelRepo) FindByTenantID(_ context.Context, tenantID string) ([]domain.Funnel, error) {
	funnels := m.byTenant[tenantID]
	result := make([]domain.Funnel, len(funnels))
	for i, f := range funnels {
		result[i] = *f
	}
	return result, nil
}

func (m *mockFunnelRepo) FindDefaultByTenantID(_ context.Context, tenantID string) (*domain.Funnel, error) {
	for _, f := range m.byTenant[tenantID] {
		if f.IsDefault && f.Active {
			return f, nil
		}
	}
	return nil, domain.ErrFunnelNotFound
}

// --- Mock ColumnRepository ---

type mockColumnRepo struct {
	columns   map[string]*domain.Column
	byFunnel  map[string][]*domain.Column
	createErr error
}

func newMockColumnRepo() *mockColumnRepo {
	return &mockColumnRepo{
		columns:  make(map[string]*domain.Column),
		byFunnel: make(map[string][]*domain.Column),
	}
}

func (m *mockColumnRepo) Create(_ context.Context, col *domain.Column) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.columns[col.ID] = col
	m.byFunnel[col.FunnelID] = append(m.byFunnel[col.FunnelID], col)
	return nil
}

func (m *mockColumnRepo) FindByID(_ context.Context, id string) (*domain.Column, error) {
	if c, ok := m.columns[id]; ok {
		return c, nil
	}
	return nil, domain.ErrColumnNotFound
}

func (m *mockColumnRepo) Update(_ context.Context, col *domain.Column) error {
	m.columns[col.ID] = col
	return nil
}

func (m *mockColumnRepo) Delete(_ context.Context, id string) error {
	col, ok := m.columns[id]
	if !ok {
		return domain.ErrColumnNotFound
	}
	delete(m.columns, id)
	// Remove from byFunnel slice
	funnelCols := m.byFunnel[col.FunnelID]
	for i, c := range funnelCols {
		if c.ID == id {
			m.byFunnel[col.FunnelID] = append(funnelCols[:i], funnelCols[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockColumnRepo) FindByFunnelID(_ context.Context, funnelID string) ([]domain.Column, error) {
	cols := m.byFunnel[funnelID]
	result := make([]domain.Column, len(cols))
	for i, c := range cols {
		result[i] = *c
	}
	// Sort by order_index
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].OrderIndex < result[i].OrderIndex {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, nil
}

func (m *mockColumnRepo) FindEntryByFunnelID(_ context.Context, funnelID string) (*domain.Column, error) {
	cols := m.byFunnel[funnelID]
	var entry *domain.Column
	for _, c := range cols {
		if c.Type == domain.ColumnTypeEntry {
			if entry == nil || c.OrderIndex < entry.OrderIndex {
				entry = c
			}
		}
	}
	if entry == nil {
		return nil, domain.ErrColumnNotFound
	}
	return entry, nil
}

func (m *mockColumnRepo) CountByFunnelID(_ context.Context, funnelID string) (int, error) {
	return len(m.byFunnel[funnelID]), nil
}

func (m *mockColumnRepo) GetMaxOrderIndex(_ context.Context, funnelID string) (int, error) {
	cols := m.byFunnel[funnelID]
	if len(cols) == 0 {
		return -1, nil
	}
	max := cols[0].OrderIndex
	for _, c := range cols[1:] {
		if c.OrderIndex > max {
			max = c.OrderIndex
		}
	}
	return max, nil
}

func (m *mockColumnRepo) SwapOrder(_ context.Context, col1ID string, order1 int, col2ID string, order2 int) error {
	if c, ok := m.columns[col1ID]; ok {
		c.OrderIndex = order1
	}
	if c, ok := m.columns[col2ID]; ok {
		c.OrderIndex = order2
	}
	return nil
}

// --- Mock LeadRepository ---

type mockLeadRepo struct {
	leads        map[string]*domain.Lead
	byContactKey map[string]*domain.Lead // key: tenantID+"|"+contactID
	byColumn     map[string]int          // columnID -> count
	createErr    error
}

func newMockLeadRepo() *mockLeadRepo {
	return &mockLeadRepo{
		leads:        make(map[string]*domain.Lead),
		byContactKey: make(map[string]*domain.Lead),
		byColumn:     make(map[string]int),
	}
}

func (m *mockLeadRepo) Create(_ context.Context, lead *domain.Lead) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.leads[lead.ID] = lead
	m.byContactKey[lead.TenantID+"|"+lead.ContactID] = lead
	m.byColumn[lead.ColumnID]++
	return nil
}

func (m *mockLeadRepo) FindByID(_ context.Context, id string) (*domain.Lead, error) {
	if l, ok := m.leads[id]; ok {
		return l, nil
	}
	return nil, domain.ErrLeadNotFound
}

func (m *mockLeadRepo) Update(_ context.Context, lead *domain.Lead) error {
	old, ok := m.leads[lead.ID]
	if ok && old.ColumnID != lead.ColumnID {
		m.byColumn[old.ColumnID]--
		m.byColumn[lead.ColumnID]++
	}
	m.leads[lead.ID] = lead
	return nil
}

func (m *mockLeadRepo) FindByContactAndTenant(_ context.Context, tenantID, contactID string) (*domain.Lead, error) {
	if l, ok := m.byContactKey[tenantID+"|"+contactID]; ok {
		return l, nil
	}
	return nil, domain.ErrLeadNotFound
}

func (m *mockLeadRepo) FindByFunnelID(_ context.Context, funnelID string, filter domain.LeadFilter) (*domain.LeadList, error) {
	var result []domain.LeadWithContact
	for _, l := range m.leads {
		if l.FunnelID != funnelID {
			continue
		}
		if filter.ColumnID != "" && l.ColumnID != filter.ColumnID {
			continue
		}
		if filter.Search != "" {
			// Simple mock search: skip (include all)
			_ = strings.Contains("", filter.Search)
		}
		result = append(result, domain.LeadWithContact{Lead: *l})
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}
	return &domain.LeadList{
		Leads: result,
		Total: int64(len(result)),
		Page:  page,
		Limit: limit,
	}, nil
}

func (m *mockLeadRepo) CountByColumnID(_ context.Context, columnID string) (int, error) {
	return m.byColumn[columnID], nil
}

func (m *mockLeadRepo) FindByConversationID(_ context.Context, conversationID string) (*domain.Lead, error) {
	return nil, domain.ErrLeadNotFound
}

func (m *mockLeadRepo) FindByTenantAndSearch(_ context.Context, tenantID, query string, limit int) ([]domain.Lead, error) {
	var result []domain.Lead
	for _, l := range m.leads {
		if l.TenantID == tenantID {
			result = append(result, *l)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// --- Mock LeadMovementRepository ---

type mockLeadMovementRepo struct {
	movements map[string][]*domain.LeadMovement // by leadID
	createErr error
}

func newMockLeadMovementRepo() *mockLeadMovementRepo {
	return &mockLeadMovementRepo{
		movements: make(map[string][]*domain.LeadMovement),
	}
}

func (m *mockLeadMovementRepo) Create(_ context.Context, mv *domain.LeadMovement) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.movements[mv.LeadID] = append(m.movements[mv.LeadID], mv)
	return nil
}

func (m *mockLeadMovementRepo) FindByLeadID(_ context.Context, leadID string) ([]domain.LeadMovement, error) {
	mvs := m.movements[leadID]
	result := make([]domain.LeadMovement, len(mvs))
	for i, mv := range mvs {
		result[i] = *mv
	}
	return result, nil
}

// --- Mock ContactProvider ---

type mockContactProvider struct {
	contacts map[string]domain.ContactInfo
}

func newMockContactProvider() *mockContactProvider {
	return &mockContactProvider{contacts: make(map[string]domain.ContactInfo)}
}

func (m *mockContactProvider) FindByID(_ context.Context, contactID string) (domain.ContactInfo, error) {
	if info, ok := m.contacts[contactID]; ok {
		return info, nil
	}
	return domain.ContactInfo{}, errors.New("contact not found")
}

// --- Mock MessageProvider ---

type mockMessageProvider struct {
	messages map[string][]domain.MessageSummary
}

func newMockMessageProvider() *mockMessageProvider {
	return &mockMessageProvider{messages: make(map[string][]domain.MessageSummary)}
}

func (m *mockMessageProvider) FindRecentByConversationID(_ context.Context, conversationID string, limit int) ([]domain.MessageSummary, error) {
	msgs := m.messages[conversationID]
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[:limit]
	}
	return msgs, nil
}

// --- Mock LeadNoteRepository ---

type mockLeadNoteRepo struct {
	notes     map[string][]*domain.LeadNote // by leadID
	createErr error
}

func newMockLeadNoteRepo() *mockLeadNoteRepo {
	return &mockLeadNoteRepo{notes: make(map[string][]*domain.LeadNote)}
}

func (m *mockLeadNoteRepo) Create(_ context.Context, note *domain.LeadNote) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.notes[note.LeadID] = append(m.notes[note.LeadID], note)
	return nil
}

func (m *mockLeadNoteRepo) FindByLeadID(_ context.Context, leadID string) ([]domain.LeadNote, error) {
	notes := m.notes[leadID]
	result := make([]domain.LeadNote, len(notes))
	for i, n := range notes {
		result[i] = *n
	}
	return result, nil
}

// --- Mock UserNameProvider ---

type mockUserNameProvider struct {
	names map[string]string
}

func newMockUserNameProvider() *mockUserNameProvider {
	return &mockUserNameProvider{names: make(map[string]string)}
}

func (m *mockUserNameProvider) FindNameByID(_ context.Context, userID string) (string, error) {
	if name, ok := m.names[userID]; ok {
		return name, nil
	}
	return "", errors.New("user not found")
}

// --- Mock ProductDetector ---

type mockProductDetector struct {
	// tenantID -> messageText -> (productID, found)
	results map[string]map[string]string
	err     error
}

func newMockProductDetector() *mockProductDetector {
	return &mockProductDetector{results: make(map[string]map[string]string)}
}

func (m *mockProductDetector) AddResult(tenantID, keyword, productID string) {
	if m.results[tenantID] == nil {
		m.results[tenantID] = make(map[string]string)
	}
	m.results[tenantID][keyword] = productID
}

func (m *mockProductDetector) DetectFromMessage(_ context.Context, tenantID, messageText string) (string, bool, error) {
	if m.err != nil {
		return "", false, m.err
	}
	if tenant, ok := m.results[tenantID]; ok {
		// Simple substring matching for testing
		for keyword, productID := range tenant {
			if strings.Contains(strings.ToLower(messageText), strings.ToLower(keyword)) {
				return productID, true, nil
			}
		}
	}
	return "", false, nil
}

// --- Mock FunnelProductRouter ---

type mockFunnelProductRouter struct {
	// productID -> funnelID
	routes map[string]string
	err    error
}

func newMockFunnelProductRouter() *mockFunnelProductRouter {
	return &mockFunnelProductRouter{routes: make(map[string]string)}
}

func (m *mockFunnelProductRouter) FindTopPriorityFunnelID(_ context.Context, tenantID, productID string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	// Routes keyed by "tenantID|productID" take precedence (tenant-scoped);
	// fall back to productID-only keys for tests that predate tenant scoping.
	if funnelID, ok := m.routes[tenantID+"|"+productID]; ok {
		return funnelID, nil
	}
	if funnelID, ok := m.routes[productID]; ok {
		return funnelID, nil
	}
	return "", errors.New("no funnel-product mapping found")
}
