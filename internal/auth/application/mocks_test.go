package application

import (
	"context"

	auditapp "github.com/sasrgita/crm-juridico/internal/audit/application"
	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

// --- Mock UserRepository ---

type mockUserRepo struct {
	users     map[string]*domain.User
	createErr error
	updateErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (m *mockUserRepo) Create(_ context.Context, user *domain.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) Update(_ context.Context, user *domain.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	// preservar o lookup por email se o email mudou
	for k, u := range m.users {
		if u.ID == user.ID {
			delete(m.users, k)
			break
		}
	}
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
	associations map[string][]string           // userID -> []tenantID
	userTenants  map[string]*domain.UserTenant // "userID:tenantID" -> UserTenant
	owners       map[string]bool               // "userID:tenantID" -> isOwner
	removed      map[string]bool               // "userID:tenantID" -> removed
}

func newMockUserTenantRepo() *mockUserTenantRepo {
	return &mockUserTenantRepo{
		associations: make(map[string][]string),
		userTenants:  make(map[string]*domain.UserTenant),
		owners:       make(map[string]bool),
		removed:      make(map[string]bool),
	}
}

func utKey(userID, tenantID string) string { return userID + ":" + tenantID }

func (m *mockUserTenantRepo) Associate(_ context.Context, userID, tenantID string) error {
	m.associations[userID] = append(m.associations[userID], tenantID)
	key := utKey(userID, tenantID)
	m.userTenants[key] = &domain.UserTenant{UserID: userID, TenantID: tenantID}
	return nil
}

func (m *mockUserTenantRepo) FindTenantIDsByUserID(_ context.Context, userID string) ([]string, error) {
	return m.associations[userID], nil
}

func (m *mockUserTenantRepo) FindByTenantID(_ context.Context, tenantID string) ([]*domain.UserTenant, error) {
	var result []*domain.UserTenant
	for _, ut := range m.userTenants {
		if ut.TenantID == tenantID {
			result = append(result, ut)
		}
	}
	return result, nil
}

func (m *mockUserTenantRepo) FindByUserAndTenant(_ context.Context, userID, tenantID string) (*domain.UserTenant, error) {
	key := utKey(userID, tenantID)
	if ut, ok := m.userTenants[key]; ok {
		return ut, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserTenantRepo) UpdateIsOwner(_ context.Context, userID, tenantID string, isOwner bool) error {
	key := utKey(userID, tenantID)
	m.owners[key] = isOwner
	if ut, ok := m.userTenants[key]; ok {
		ut.IsOwner = isOwner
	}
	return nil
}

func (m *mockUserTenantRepo) UpdateWhatsAppID(_ context.Context, userID, tenantID string, whatsAppID string) error {
	key := utKey(userID, tenantID)
	if ut, ok := m.userTenants[key]; ok {
		ut.WhatsAppID = whatsAppID
	}
	return nil
}

func (m *mockUserTenantRepo) RemoveFromTenant(_ context.Context, userID, tenantID string) error {
	key := utKey(userID, tenantID)
	m.removed[key] = true
	delete(m.userTenants, key)
	return nil
}

func (m *mockUserTenantRepo) IsOwner(_ context.Context, userID, tenantID string) (bool, error) {
	key := utKey(userID, tenantID)
	return m.owners[key], nil
}

// --- Mock InviteTokenRepository ---

type mockInviteTokenRepo struct {
	tokens  map[string]*domain.InviteToken // id -> token
	byToken map[string]*domain.InviteToken // token string -> token
}

func newMockInviteTokenRepo() *mockInviteTokenRepo {
	return &mockInviteTokenRepo{
		tokens:  make(map[string]*domain.InviteToken),
		byToken: make(map[string]*domain.InviteToken),
	}
}

func (m *mockInviteTokenRepo) Create(_ context.Context, token *domain.InviteToken) error {
	m.tokens[token.ID] = token
	m.byToken[token.Token] = token
	return nil
}

func (m *mockInviteTokenRepo) FindByToken(_ context.Context, token string) (*domain.InviteToken, error) {
	if t, ok := m.byToken[token]; ok {
		return t, nil
	}
	return nil, domain.ErrInviteTokenNotFound
}

func (m *mockInviteTokenRepo) FindByTenantID(_ context.Context, tenantID string) ([]*domain.InviteToken, error) {
	var result []*domain.InviteToken
	for _, t := range m.tokens {
		if t.TenantID == tenantID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockInviteTokenRepo) Update(_ context.Context, token *domain.InviteToken) error {
	m.tokens[token.ID] = token
	m.byToken[token.Token] = token
	return nil
}

func (m *mockInviteTokenRepo) Delete(_ context.Context, id string) error {
	if t, ok := m.tokens[id]; ok {
		delete(m.byToken, t.Token)
		delete(m.tokens, id)
	}
	return nil
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

// --- spyAuditPublisher (F12 Step 7) ---
//
// Captura chamadas de Publish para asserts. Implementa auditapp.Publisher.
// Se publishErr != nil, retorna o erro sem registrar a chamada — usado
// para validar S1-C17 (falha de auditoria nao aborta operacao).
type spyAuditPublisher struct {
	calls      []auditapp.RegisterAuditLogInput
	publishErr error
}

func (s *spyAuditPublisher) Publish(_ context.Context, in auditapp.RegisterAuditLogInput) error {
	if s.publishErr != nil {
		return s.publishErr
	}
	s.calls = append(s.calls, in)
	return nil
}
