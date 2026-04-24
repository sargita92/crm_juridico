package application

import (
	"context"
	"sync"

	"github.com/sasrgita/crm-juridico/internal/audit/domain"
)

// mockAuditRepo e o duble usado pelos testes da camada de aplicacao.
//
// Captura as chamadas para que os testes inspecionem o argumento passado
// para Create (ex.: validar que metadata foi sanitizada antes do persist).
type mockAuditRepo struct {
	mu        sync.Mutex
	createErr error
	created   []*domain.AuditLog

	listFn       func(ctx context.Context, filter domain.Filter) ([]*domain.AuditLog, int64, error)
	findByIDFn   func(ctx context.Context, id string) (*domain.AuditLog, error)
	createCalled int
}

func newMockAuditRepo() *mockAuditRepo {
	return &mockAuditRepo{}
}

func (m *mockAuditRepo) Create(_ context.Context, log *domain.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCalled++
	if m.createErr != nil {
		return m.createErr
	}
	m.created = append(m.created, log)
	return nil
}

func (m *mockAuditRepo) List(ctx context.Context, filter domain.Filter) ([]*domain.AuditLog, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockAuditRepo) FindByID(ctx context.Context, id string) (*domain.AuditLog, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, domain.ErrAuditLogNotFound
}

func (m *mockAuditRepo) lastCreated() *domain.AuditLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.created) == 0 {
		return nil
	}
	return m.created[len(m.created)-1]
}
