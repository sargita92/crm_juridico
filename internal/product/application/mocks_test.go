package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

// --- Mock ProductRepository ---

type mockProductRepo struct {
	products  map[string]*domain.Product
	createErr error
	updateErr error
}

func newMockProductRepo() *mockProductRepo {
	return &mockProductRepo{
		products: make(map[string]*domain.Product),
	}
}

func (m *mockProductRepo) Create(_ context.Context, p *domain.Product) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.products[p.ID] = p
	return nil
}

func (m *mockProductRepo) FindByID(_ context.Context, id string) (*domain.Product, error) {
	if p, ok := m.products[id]; ok {
		return p, nil
	}
	return nil, domain.ErrProductNotFound
}

func (m *mockProductRepo) Update(_ context.Context, p *domain.Product) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.products[p.ID] = p
	return nil
}

func (m *mockProductRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.products[id]; !ok {
		return domain.ErrProductNotFound
	}
	delete(m.products, id)
	return nil
}

func (m *mockProductRepo) FindAll(_ context.Context, activeOnly bool) ([]domain.Product, error) {
	var result []domain.Product
	for _, p := range m.products {
		if activeOnly && !p.Active {
			continue
		}
		result = append(result, *p)
	}
	if result == nil {
		result = []domain.Product{}
	}
	return result, nil
}

func (m *mockProductRepo) FindActiveByIDs(_ context.Context, ids []string) ([]domain.Product, error) {
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var result []domain.Product
	for _, p := range m.products {
		if idSet[p.ID] && p.Active {
			result = append(result, *p)
		}
	}
	if result == nil {
		result = []domain.Product{}
	}
	return result, nil
}

// --- Mock TenantProductRepository ---

type mockTenantProductRepo struct {
	associations map[string]*domain.TenantProduct // key: tenantID+"|"+productID
	createErr    error
}

func newMockTenantProductRepo() *mockTenantProductRepo {
	return &mockTenantProductRepo{
		associations: make(map[string]*domain.TenantProduct),
	}
}

func (m *mockTenantProductRepo) Create(_ context.Context, tp *domain.TenantProduct) error {
	if m.createErr != nil {
		return m.createErr
	}
	key := tp.TenantID + "|" + tp.ProductID
	if _, exists := m.associations[key]; exists {
		return domain.ErrTenantProductAlreadyExists
	}
	m.associations[key] = tp
	return nil
}

func (m *mockTenantProductRepo) Delete(_ context.Context, tenantID, productID string) error {
	key := tenantID + "|" + productID
	if _, exists := m.associations[key]; !exists {
		return domain.ErrTenantProductNotFound
	}
	delete(m.associations, key)
	return nil
}

func (m *mockTenantProductRepo) FindByTenantID(_ context.Context, tenantID string) ([]domain.TenantProduct, error) {
	var result []domain.TenantProduct
	for _, tp := range m.associations {
		if tp.TenantID == tenantID {
			result = append(result, *tp)
		}
	}
	if result == nil {
		result = []domain.TenantProduct{}
	}
	return result, nil
}

func (m *mockTenantProductRepo) FindByProductID(_ context.Context, productID string) ([]domain.TenantProduct, error) {
	var result []domain.TenantProduct
	for _, tp := range m.associations {
		if tp.ProductID == productID {
			result = append(result, *tp)
		}
	}
	if result == nil {
		result = []domain.TenantProduct{}
	}
	return result, nil
}

func (m *mockTenantProductRepo) Exists(_ context.Context, tenantID, productID string) (bool, error) {
	key := tenantID + "|" + productID
	_, exists := m.associations[key]
	return exists, nil
}

// --- Mock FunnelProductRepository ---

type mockFunnelProductRepo struct {
	links     map[string]*domain.FunnelProduct // key: funnelID+"|"+productID
	byProduct map[string][]*domain.FunnelProduct
	byFunnel  map[string][]*domain.FunnelProduct
	createErr error
}

func newMockFunnelProductRepo() *mockFunnelProductRepo {
	return &mockFunnelProductRepo{
		links:     make(map[string]*domain.FunnelProduct),
		byProduct: make(map[string][]*domain.FunnelProduct),
		byFunnel:  make(map[string][]*domain.FunnelProduct),
	}
}

func (m *mockFunnelProductRepo) Create(_ context.Context, fp *domain.FunnelProduct) error {
	if m.createErr != nil {
		return m.createErr
	}
	key := fp.FunnelID + "|" + fp.ProductID
	if _, exists := m.links[key]; exists {
		return domain.ErrFunnelProductAlreadyExists
	}
	m.links[key] = fp
	m.byProduct[fp.ProductID] = append(m.byProduct[fp.ProductID], fp)
	m.byFunnel[fp.FunnelID] = append(m.byFunnel[fp.FunnelID], fp)
	return nil
}

func (m *mockFunnelProductRepo) Delete(_ context.Context, funnelID, productID string) error {
	key := funnelID + "|" + productID
	fp, exists := m.links[key]
	if !exists {
		return domain.ErrFunnelProductNotFound
	}
	delete(m.links, key)

	list := m.byProduct[fp.ProductID]
	for i, p := range list {
		if p.FunnelID == funnelID {
			m.byProduct[fp.ProductID] = append(list[:i], list[i+1:]...)
			break
		}
	}

	list = m.byFunnel[fp.FunnelID]
	for i, p := range list {
		if p.ProductID == productID {
			m.byFunnel[fp.FunnelID] = append(list[:i], list[i+1:]...)
			break
		}
	}

	return nil
}

func (m *mockFunnelProductRepo) FindByProductID(_ context.Context, productID string) ([]domain.FunnelProduct, error) {
	list := m.byProduct[productID]
	result := make([]domain.FunnelProduct, len(list))
	for i, fp := range list {
		result[i] = *fp
	}
	return result, nil
}

func (m *mockFunnelProductRepo) FindByFunnelID(_ context.Context, funnelID string) ([]domain.FunnelProduct, error) {
	list := m.byFunnel[funnelID]
	result := make([]domain.FunnelProduct, len(list))
	for i, fp := range list {
		result[i] = *fp
	}
	return result, nil
}

func (m *mockFunnelProductRepo) FindTopPriorityFunnel(_ context.Context, _ string, productID string) (*domain.FunnelProduct, error) {
	list := m.byProduct[productID]
	if len(list) == 0 {
		return nil, domain.ErrFunnelProductNotFound
	}
	top := list[0]
	for _, fp := range list[1:] {
		if fp.Priority > top.Priority {
			top = fp
		}
	}
	return top, nil
}

func (m *mockFunnelProductRepo) UpdatePriority(_ context.Context, funnelID, productID string, priority int) error {
	key := funnelID + "|" + productID
	fp, exists := m.links[key]
	if !exists {
		return domain.ErrFunnelProductNotFound
	}
	fp.Priority = priority
	return nil
}
