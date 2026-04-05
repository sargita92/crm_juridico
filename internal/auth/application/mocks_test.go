package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

// --- Mock UserRepository ---

type mockUserRepo struct {
	users map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (m *mockUserRepo) Create(_ context.Context, user *domain.User) error {
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, ok := m.users[email]
	return ok, nil
}

// --- Mock UserTenantRepository ---

type mockUserTenantRepo struct {
	associations map[string][]string // userID -> []tenantID
}

func newMockUserTenantRepo() *mockUserTenantRepo {
	return &mockUserTenantRepo{associations: make(map[string][]string)}
}

func (m *mockUserTenantRepo) Associate(_ context.Context, userID, tenantID string) error {
	m.associations[userID] = append(m.associations[userID], tenantID)
	return nil
}

func (m *mockUserTenantRepo) FindTenantIDsByUserID(_ context.Context, userID string) ([]string, error) {
	return m.associations[userID], nil
}

// --- Mock TenantRepository ---

type mockTenantRepo struct {
	tenants map[string]*tenantdomain.Tenant
}

func newMockTenantRepo() *mockTenantRepo {
	return &mockTenantRepo{tenants: make(map[string]*tenantdomain.Tenant)}
}

func (m *mockTenantRepo) Create(_ context.Context, tenant *tenantdomain.Tenant) error {
	m.tenants[tenant.ID] = tenant
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

func (m *mockTenantRepo) Update(_ context.Context, tenant *tenantdomain.Tenant) error {
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *mockTenantRepo) FindWithFilter(_ context.Context, _ tenantdomain.TenantFilter) (*tenantdomain.TenantList, error) {
	return &tenantdomain.TenantList{}, nil
}

func (m *mockTenantRepo) FindByDocument(_ context.Context, document string) (*tenantdomain.Tenant, error) {
	for _, t := range m.tenants {
		if t.Document == document {
			return t, nil
		}
	}
	return nil, tenantdomain.ErrTenantNotFound
}

// --- Mock PasswordHasher ---

type mockHasher struct{}

func (m *mockHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (m *mockHasher) Compare(hash, password string) error {
	if hash == "hashed:"+password {
		return nil
	}
	return domain.ErrInvalidCredentials
}

// --- Mock TokenProvider ---

type mockTokenProvider struct {
	lastClaims *domain.TokenClaims
}

func (m *mockTokenProvider) Generate(claims domain.TokenClaims) (string, error) {
	m.lastClaims = &claims
	token := "token:" + claims.UserID
	if claims.TenantID != "" {
		token += ":" + claims.TenantID
	}
	return token, nil
}

func (m *mockTokenProvider) Validate(token string) (*domain.TokenClaims, error) {
	if m.lastClaims != nil {
		return m.lastClaims, nil
	}
	return nil, domain.ErrInvalidCredentials
}
