package application

import (
	"context"
	"strings"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// --- mockMcpServerRepo ---

type mockMcpServerRepo struct {
	mcps map[string]*domain.McpServer
}

func newMockMcpServerRepo() *mockMcpServerRepo {
	return &mockMcpServerRepo{mcps: make(map[string]*domain.McpServer)}
}

func (m *mockMcpServerRepo) addMcp(mcp *domain.McpServer) {
	m.mcps[mcp.ID] = mcp
}

func (m *mockMcpServerRepo) Create(_ context.Context, mcp *domain.McpServer) error {
	m.mcps[mcp.ID] = mcp
	return nil
}

func (m *mockMcpServerRepo) FindByID(_ context.Context, id string) (*domain.McpServer, error) {
	mcp, ok := m.mcps[id]
	if !ok {
		return nil, domain.ErrMcpNotFound
	}
	return mcp, nil
}

func (m *mockMcpServerRepo) Update(_ context.Context, mcp *domain.McpServer) error {
	if _, ok := m.mcps[mcp.ID]; !ok {
		return domain.ErrMcpNotFound
	}
	m.mcps[mcp.ID] = mcp
	return nil
}

func (m *mockMcpServerRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.mcps[id]; !ok {
		return domain.ErrMcpNotFound
	}
	delete(m.mcps, id)
	return nil
}

func (m *mockMcpServerRepo) FindWithFilter(_ context.Context, filter domain.McpFilter) (*domain.McpList, error) {
	var filtered []domain.McpServer
	for _, mcp := range m.mcps {
		if filter.Search != "" && !strings.Contains(strings.ToLower(mcp.Name), strings.ToLower(filter.Search)) {
			continue
		}
		filtered = append(filtered, *mcp)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}

	total := int64(len(filtered))
	start := (page - 1) * limit
	if start > int(total) {
		start = int(total)
	}
	end := start + limit
	if end > int(total) {
		end = int(total)
	}

	return &domain.McpList{
		McpServers: filtered[start:end],
		Total:      total,
		Page:       page,
		Limit:      limit,
	}, nil
}

func (m *mockMcpServerRepo) FindByIDs(_ context.Context, ids []string) ([]domain.McpServer, error) {
	var result []domain.McpServer
	for _, id := range ids {
		if mcp, ok := m.mcps[id]; ok {
			result = append(result, *mcp)
		}
	}
	return result, nil
}

// --- mockSpecialistMcpRepo ---

type mockSpecialistMcpRepo struct {
	associations map[string]map[string]bool // specialistID -> mcpID -> true
}

func newMockSpecialistMcpRepo() *mockSpecialistMcpRepo {
	return &mockSpecialistMcpRepo{associations: make(map[string]map[string]bool)}
}

func (m *mockSpecialistMcpRepo) Associate(_ context.Context, specialistID, mcpID string) error {
	if m.associations[specialistID] == nil {
		m.associations[specialistID] = make(map[string]bool)
	}
	if m.associations[specialistID][mcpID] {
		return domain.ErrMcpAlreadyAssociated
	}
	m.associations[specialistID][mcpID] = true
	return nil
}

func (m *mockSpecialistMcpRepo) Dissociate(_ context.Context, specialistID, mcpID string) error {
	if m.associations[specialistID] == nil || !m.associations[specialistID][mcpID] {
		return domain.ErrMcpNotAssociated
	}
	delete(m.associations[specialistID], mcpID)
	return nil
}

func (m *mockSpecialistMcpRepo) DissociateAllByMcpID(_ context.Context, mcpID string) error {
	for specID := range m.associations {
		delete(m.associations[specID], mcpID)
	}
	return nil
}

func (m *mockSpecialistMcpRepo) FindMcpIDsBySpecialistID(_ context.Context, specialistID string) ([]string, error) {
	var ids []string
	for mcpID := range m.associations[specialistID] {
		ids = append(ids, mcpID)
	}
	return ids, nil
}

func (m *mockSpecialistMcpRepo) FindSpecialistIDsByMcpID(_ context.Context, mcpID string) ([]string, error) {
	var ids []string
	for specID, mcps := range m.associations {
		if mcps[mcpID] {
			ids = append(ids, specID)
		}
	}
	return ids, nil
}

func (m *mockSpecialistMcpRepo) Exists(_ context.Context, specialistID, mcpID string) (bool, error) {
	if m.associations[specialistID] == nil {
		return false, nil
	}
	return m.associations[specialistID][mcpID], nil
}

func (m *mockSpecialistMcpRepo) CountByMcpID(_ context.Context, mcpID string) (int, error) {
	count := 0
	for _, mcps := range m.associations {
		if mcps[mcpID] {
			count++
		}
	}
	return count, nil
}

// --- mockSpecialistRepo ---

type mockSpecialistRepo struct {
	specialists map[string]*specialistdomain.Specialist
}

func newMockSpecialistRepo() *mockSpecialistRepo {
	return &mockSpecialistRepo{specialists: make(map[string]*specialistdomain.Specialist)}
}

func (m *mockSpecialistRepo) addSpecialist(s *specialistdomain.Specialist) {
	m.specialists[s.ID] = s
}

func (m *mockSpecialistRepo) FindByID(_ context.Context, id string) (*specialistdomain.Specialist, error) {
	s, ok := m.specialists[id]
	if !ok {
		return nil, specialistdomain.ErrSpecialistNotFound
	}
	return s, nil
}
