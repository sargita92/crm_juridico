package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

// --- Mock ProductRepository ---

type mockProductRepo struct {
	products  map[string]*domain.Product
	byTenant  map[string][]*domain.Product
	createErr error
	updateErr error
}

func newMockProductRepo() *mockProductRepo {
	return &mockProductRepo{
		products: make(map[string]*domain.Product),
		byTenant: make(map[string][]*domain.Product),
	}
}

func (m *mockProductRepo) Create(_ context.Context, p *domain.Product) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.products[p.ID] = p
	m.byTenant[p.TenantID] = append(m.byTenant[p.TenantID], p)
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

func (m *mockProductRepo) FindByTenantID(_ context.Context, tenantID string, activeOnly bool) ([]domain.Product, error) {
	list := m.byTenant[tenantID]
	var result []domain.Product
	for _, p := range list {
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

func (m *mockProductRepo) FindActiveByTenantID(_ context.Context, tenantID string) ([]domain.Product, error) {
	list := m.byTenant[tenantID]
	var result []domain.Product
	for _, p := range list {
		if p.Active {
			result = append(result, *p)
		}
	}
	if result == nil {
		result = []domain.Product{}
	}
	return result, nil
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

func (m *mockFunnelProductRepo) FindTopPriorityFunnel(_ context.Context, productID string) (*domain.FunnelProduct, error) {
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
