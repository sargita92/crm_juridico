package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

// --- mockFileRepo ---

type mockFileRepo struct {
	byID      map[string]*domain.File
	createErr error
	listErr   error
}

func newMockFileRepo() *mockFileRepo {
	return &mockFileRepo{byID: make(map[string]*domain.File)}
}

func (m *mockFileRepo) Create(_ context.Context, f *domain.File) error {
	if m.createErr != nil {
		return m.createErr
	}
	clone := *f
	m.byID[f.ID] = &clone
	return nil
}

func (m *mockFileRepo) FindByID(_ context.Context, tenantID, id string) (*domain.File, error) {
	f, ok := m.byID[id]
	if !ok || f.TenantID != tenantID {
		return nil, domain.ErrFileNotFound
	}
	clone := *f
	return &clone, nil
}

func (m *mockFileRepo) List(_ context.Context, q domain.ListQuery) (*domain.ListResult, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var items []domain.FileWithContext
	for _, f := range m.byID {
		if f.TenantID != q.TenantID {
			continue
		}
		if q.LeadID != nil && (f.LeadID == nil || *f.LeadID != *q.LeadID) {
			continue
		}
		if q.MediaType != nil && f.MediaType != *q.MediaType {
			continue
		}
		if s := strings.ToLower(strings.TrimSpace(q.Search)); s != "" && !strings.Contains(strings.ToLower(f.Name), s) {
			continue
		}
		if q.From != nil && f.CreatedAt.Before(*q.From) {
			continue
		}
		if q.To != nil && f.CreatedAt.After(*q.To) {
			continue
		}
		items = append(items, domain.FileWithContext{File: *f})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })

	total := int64(len(items))
	page := q.Page
	if page < 1 {
		page = 1
	}
	size := q.PageSize
	if size <= 0 {
		size = domain.DefaultPageSize
	}
	start := (page - 1) * size
	end := start + size
	if start > len(items) {
		start = len(items)
	}
	if end > len(items) {
		end = len(items)
	}
	return &domain.ListResult{
		Items:    items[start:end],
		Total:    total,
		Page:     page,
		PageSize: size,
	}, nil
}

func (m *mockFileRepo) CountByLead(_ context.Context, tenantID, leadID string) (int64, error) {
	var c int64
	for _, f := range m.byID {
		if f.TenantID == tenantID && f.LeadID != nil && *f.LeadID == leadID {
			c++
		}
	}
	return c, nil
}

func (m *mockFileRepo) ListRecentByLead(_ context.Context, tenantID, leadID string, limit int) ([]domain.File, error) {
	var out []domain.File
	for _, f := range m.byID {
		if f.TenantID == tenantID && f.LeadID != nil && *f.LeadID == leadID {
			out = append(out, *f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// --- mockStorage ---

type mockStorage struct {
	blobs   map[string][]byte
	saveErr error
	openErr error
}

func newMockStorage() *mockStorage {
	return &mockStorage{blobs: make(map[string][]byte)}
}

func (m *mockStorage) Save(_ context.Context, key string, content []byte) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	b := make([]byte, len(content))
	copy(b, content)
	m.blobs[key] = b
	return nil
}

func (m *mockStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	if m.openErr != nil {
		return nil, m.openErr
	}
	b, ok := m.blobs[key]
	if !ok {
		return nil, domain.ErrFileContentUnavailable
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (m *mockStorage) Size(_ context.Context, key string) (int64, error) {
	b, ok := m.blobs[key]
	if !ok {
		return 0, domain.ErrFileContentUnavailable
	}
	return int64(len(b)), nil
}

// --- mockLeadLookup ---

type mockLeadLookup struct {
	// byConv[tenantID][conversationID] = leadID
	byConv map[string]map[string]string
	err    error
}

func newMockLeadLookup() *mockLeadLookup {
	return &mockLeadLookup{byConv: make(map[string]map[string]string)}
}

func (m *mockLeadLookup) set(tenantID, conversationID, leadID string) {
	if _, ok := m.byConv[tenantID]; !ok {
		m.byConv[tenantID] = make(map[string]string)
	}
	m.byConv[tenantID][conversationID] = leadID
}

func (m *mockLeadLookup) FindByConversation(_ context.Context, tenantID, conversationID string) (string, bool, error) {
	if m.err != nil {
		return "", false, m.err
	}
	if inner, ok := m.byConv[tenantID]; ok {
		if id, ok := inner[conversationID]; ok {
			return id, true, nil
		}
	}
	return "", false, nil
}

var errBoom = errors.New("boom")
