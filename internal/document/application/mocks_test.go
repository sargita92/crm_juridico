package application

import (
	"context"
	"strings"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// --- mockDocumentRepo ---

type mockDocumentRepo struct {
	documents map[string]*domain.Document
}

func newMockDocumentRepo() *mockDocumentRepo {
	return &mockDocumentRepo{documents: make(map[string]*domain.Document)}
}

func (m *mockDocumentRepo) addDocument(d *domain.Document) {
	m.documents[d.ID] = d
}

func (m *mockDocumentRepo) Create(_ context.Context, doc *domain.Document) error {
	m.documents[doc.ID] = doc
	return nil
}

func (m *mockDocumentRepo) FindByID(_ context.Context, id string) (*domain.Document, error) {
	doc, ok := m.documents[id]
	if !ok {
		return nil, domain.ErrDocumentNotFound
	}
	return doc, nil
}

func (m *mockDocumentRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.documents[id]; !ok {
		return domain.ErrDocumentNotFound
	}
	delete(m.documents, id)
	return nil
}

func (m *mockDocumentRepo) FindWithFilter(_ context.Context, filter domain.DocumentFilter) (*domain.DocumentList, error) {
	var filtered []domain.Document
	for _, doc := range m.documents {
		if filter.Search != "" && !strings.Contains(strings.ToLower(doc.Name), strings.ToLower(filter.Search)) {
			continue
		}
		filtered = append(filtered, *doc)
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

	return &domain.DocumentList{
		Documents: filtered[start:end],
		Total:     total,
		Page:      page,
		Limit:     limit,
	}, nil
}

func (m *mockDocumentRepo) FindByIDs(_ context.Context, ids []string) ([]domain.Document, error) {
	var result []domain.Document
	for _, id := range ids {
		if doc, ok := m.documents[id]; ok {
			result = append(result, *doc)
		}
	}
	return result, nil
}

// --- mockSpecialistDocumentRepo ---

type mockSpecialistDocumentRepo struct {
	associations map[string]map[string]bool // specialistID -> documentID -> true
}

func newMockSpecialistDocumentRepo() *mockSpecialistDocumentRepo {
	return &mockSpecialistDocumentRepo{associations: make(map[string]map[string]bool)}
}

func (m *mockSpecialistDocumentRepo) Associate(_ context.Context, specialistID, documentID string) error {
	if m.associations[specialistID] == nil {
		m.associations[specialistID] = make(map[string]bool)
	}
	if m.associations[specialistID][documentID] {
		return domain.ErrDocumentAlreadyAssociated
	}
	m.associations[specialistID][documentID] = true
	return nil
}

func (m *mockSpecialistDocumentRepo) Dissociate(_ context.Context, specialistID, documentID string) error {
	if m.associations[specialistID] == nil || !m.associations[specialistID][documentID] {
		return domain.ErrDocumentNotAssociated
	}
	delete(m.associations[specialistID], documentID)
	return nil
}

func (m *mockSpecialistDocumentRepo) DissociateAllByDocumentID(_ context.Context, documentID string) error {
	for specID := range m.associations {
		delete(m.associations[specID], documentID)
	}
	return nil
}

func (m *mockSpecialistDocumentRepo) FindDocumentIDsBySpecialistID(_ context.Context, specialistID string) ([]string, error) {
	var ids []string
	for docID := range m.associations[specialistID] {
		ids = append(ids, docID)
	}
	return ids, nil
}

func (m *mockSpecialistDocumentRepo) FindSpecialistIDsByDocumentID(_ context.Context, documentID string) ([]string, error) {
	var ids []string
	for specID, docs := range m.associations {
		if docs[documentID] {
			ids = append(ids, specID)
		}
	}
	return ids, nil
}

func (m *mockSpecialistDocumentRepo) Exists(_ context.Context, specialistID, documentID string) (bool, error) {
	if m.associations[specialistID] == nil {
		return false, nil
	}
	return m.associations[specialistID][documentID], nil
}

func (m *mockSpecialistDocumentRepo) CountByDocumentID(_ context.Context, documentID string) (int, error) {
	count := 0
	for _, docs := range m.associations {
		if docs[documentID] {
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
