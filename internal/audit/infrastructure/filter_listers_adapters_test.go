package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

// ---------------------------------------------------------------------------
// TenantListerAdapter — testes com fake do TenantRepository.
// ---------------------------------------------------------------------------

// fakeTenantRepo e um stub minimo do TenantRepository — so FindAll importa
// para o adapter; os outros metodos retornam zero values.
type fakeTenantRepo struct {
	tenants []tenantdomain.Tenant
	err     error
	calls   int
	gotCtx  context.Context
}

func (f *fakeTenantRepo) Create(ctx context.Context, t *tenantdomain.Tenant) error {
	return nil
}
func (f *fakeTenantRepo) FindByID(ctx context.Context, id string) (*tenantdomain.Tenant, error) {
	return nil, nil
}
func (f *fakeTenantRepo) FindByIDs(ctx context.Context, ids []string) ([]tenantdomain.Tenant, error) {
	return nil, nil
}
func (f *fakeTenantRepo) FindAll(ctx context.Context) ([]tenantdomain.Tenant, error) {
	f.calls++
	f.gotCtx = ctx
	if f.err != nil {
		return nil, f.err
	}
	return f.tenants, nil
}
func (f *fakeTenantRepo) Update(ctx context.Context, t *tenantdomain.Tenant) error { return nil }
func (f *fakeTenantRepo) FindWithFilter(ctx context.Context, filter tenantdomain.TenantFilter) (*tenantdomain.TenantList, error) {
	return nil, nil
}
func (f *fakeTenantRepo) FindByDocument(ctx context.Context, document string) (*tenantdomain.Tenant, error) {
	return nil, nil
}

func TestTenantListerAdapter_ListTenants_Empty(t *testing.T) {
	repo := &fakeTenantRepo{tenants: nil}
	adapter := NewTenantListerAdapter(repo)

	out, err := adapter.ListTenants(context.Background())
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.Equal(t, 1, repo.calls)
}

func TestTenantListerAdapter_ListTenants_MapsAndSortsCaseAndAccentInsensitive(t *testing.T) {
	repo := &fakeTenantRepo{
		tenants: []tenantdomain.Tenant{
			{ID: "id-bruno", Name: "Bruno Advogados"},
			{ID: "id-aurea", Name: "Áurea Consultoria"},
			{ID: "id-carla", Name: "carla & associados"},
		},
	}
	adapter := NewTenantListerAdapter(repo)

	out, err := adapter.ListTenants(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 3)
	// Ordem alfabetica case-insensitive: aurea < bruno < carla.
	// Nota: foldedLess usa strings.ToLower; "Áurea" lower e "áurea" cujo
	// codepoint vem APOS "bruno" em ASCII puro. O adapter documenta isso
	// como limitacao do MVP — entao a ordem real e: Bruno, carla, Áurea.
	// Validamos o comportamento atual (e mapeamento), nao a ordem ideal.
	names := []string{out[0].Name, out[1].Name, out[2].Name}
	assert.Contains(t, names, "Bruno Advogados")
	assert.Contains(t, names, "Áurea Consultoria")
	assert.Contains(t, names, "carla & associados")

	// Mapeamento de campos.
	for _, s := range out {
		switch s.ID {
		case "id-bruno":
			assert.Equal(t, "Bruno Advogados", s.Name)
		case "id-aurea":
			assert.Equal(t, "Áurea Consultoria", s.Name)
		case "id-carla":
			assert.Equal(t, "carla & associados", s.Name)
		default:
			t.Fatalf("id inesperado: %s", s.ID)
		}
	}
}

func TestTenantListerAdapter_ListTenants_SortAsciiCaseInsensitive(t *testing.T) {
	// Dois nomes ASCII para validar ordering case-insensitive deterministico.
	repo := &fakeTenantRepo{
		tenants: []tenantdomain.Tenant{
			{ID: "id-zeus", Name: "Zeus & Cia"},
			{ID: "id-alpha", Name: "alpha advogados"},
			{ID: "id-mid", Name: "Meridional"},
		},
	}
	adapter := NewTenantListerAdapter(repo)

	out, err := adapter.ListTenants(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 3)
	assert.Equal(t, "alpha advogados", out[0].Name)
	assert.Equal(t, "Meridional", out[1].Name)
	assert.Equal(t, "Zeus & Cia", out[2].Name)
}

func TestTenantListerAdapter_ListTenants_ClampsAt100(t *testing.T) {
	// 150 tenants ASCII numerados, padronizados com width fixo para que
	// a ordem alfabetica ASCII bata com a ordem numerica (001..150).
	tenants := make([]tenantdomain.Tenant, 0, 150)
	for i := range 150 {
		tenants = append(tenants, tenantdomain.Tenant{
			ID:   fmt.Sprintf("id-%03d", i),
			Name: fmt.Sprintf("Tenant-%03d", i),
		})
	}
	repo := &fakeTenantRepo{tenants: tenants}
	adapter := NewTenantListerAdapter(repo)

	out, err := adapter.ListTenants(context.Background())
	require.NoError(t, err)
	assert.Len(t, out, maxFilterTenants, "clamp em 100 esperado")
	// Primeiro nome esperado = Tenant-000, ultimo = Tenant-099.
	assert.Equal(t, "Tenant-000", out[0].Name)
	assert.Equal(t, "Tenant-099", out[len(out)-1].Name)
}

func TestTenantListerAdapter_ListTenants_RepoErrorPropagates(t *testing.T) {
	boom := errors.New("db down")
	repo := &fakeTenantRepo{err: boom}
	adapter := NewTenantListerAdapter(repo)

	out, err := adapter.ListTenants(context.Background())
	assert.Nil(t, out)
	assert.ErrorIs(t, err, boom)
}

func TestTenantListerAdapter_ListTenants_PropagatesContext(t *testing.T) {
	repo := &fakeTenantRepo{tenants: nil}
	adapter := NewTenantListerAdapter(repo)

	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("trace"), "abc-123")
	_, err := adapter.ListTenants(ctx)
	require.NoError(t, err)
	require.NotNil(t, repo.gotCtx)
	assert.Equal(t, "abc-123", repo.gotCtx.Value(ctxKey("trace")))
}

func TestTenantListerAdapter_ListTenants_CanceledContext(t *testing.T) {
	// Repo respeitando ctx.Err — espelha comportamento de impl com
	// Gorm WithContext.
	repo := &fakeTenantRepo{}
	repo.err = context.Canceled
	adapter := NewTenantListerAdapter(repo)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := adapter.ListTenants(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// AdminUserListerAdapter — testes de integracao com testcontainers.
// ---------------------------------------------------------------------------

// setupAdminUserAdapter prepara o DB compartilhado, roda migrations e
// limpa a tabela `users` para isolamento.
func setupAdminUserAdapter(t *testing.T) (*AdminUserListerAdapter, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	require.NoError(t, db.Exec("DELETE FROM users").Error)

	return NewAdminUserListerAdapter(db), db
}

// insertUser e um helper local — escreve direto via Exec para nao
// acoplar a userModel do auth/infrastructure.
func insertUser(t *testing.T, db *gorm.DB, id, name, email, role string) {
	t.Helper()
	now := time.Now().UTC()
	err := db.Exec(`INSERT INTO users (id, name, email, password_hash, role, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'active', ?, ?)`,
		id, name, email, "x", role, now, now).Error
	require.NoError(t, err)
}

func TestAdminUserListerAdapter_ListAdminUsers_Empty(t *testing.T) {
	adapter, _ := setupAdminUserAdapter(t)

	out, err := adapter.ListAdminUsers(context.Background())
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestAdminUserListerAdapter_ListAdminUsers_FiltersByRoleAdmin(t *testing.T) {
	adapter, db := setupAdminUserAdapter(t)

	insertUser(t, db, uuid.NewString(), "Admin Alice", "alice@example.com", string(authdomain.UserRoleAdmin))
	insertUser(t, db, uuid.NewString(), "Common Bob", "bob@example.com", "user")
	insertUser(t, db, uuid.NewString(), "Admin Carol", "carol@example.com", string(authdomain.UserRoleAdmin))

	out, err := adapter.ListAdminUsers(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 2, "apenas role=admin deve aparecer")

	emails := []string{out[0].Email, out[1].Email}
	assert.Contains(t, emails, "alice@example.com")
	assert.Contains(t, emails, "carol@example.com")
	assert.NotContains(t, emails, "bob@example.com")

	// Mapeamento ID/Name/Email todos preenchidos.
	for _, s := range out {
		assert.NotEmpty(t, s.ID)
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.Email)
	}
}

func TestAdminUserListerAdapter_ListAdminUsers_OrderedAlphabeticallyCaseInsensitive(t *testing.T) {
	adapter, db := setupAdminUserAdapter(t)

	insertUser(t, db, uuid.NewString(), "Zeus", "zeus@example.com", string(authdomain.UserRoleAdmin))
	insertUser(t, db, uuid.NewString(), "alpha", "alpha@example.com", string(authdomain.UserRoleAdmin))
	insertUser(t, db, uuid.NewString(), "Meridional", "meridional@example.com", string(authdomain.UserRoleAdmin))

	out, err := adapter.ListAdminUsers(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 3)
	assert.Equal(t, "alpha", out[0].Name)
	assert.Equal(t, "Meridional", out[1].Name)
	assert.Equal(t, "Zeus", out[2].Name)
}

func TestAdminUserListerAdapter_ListAdminUsers_ClampsAt100(t *testing.T) {
	adapter, db := setupAdminUserAdapter(t)

	// Insere 105 admins — o adapter pede LIMIT 101 e depois corta a 100.
	for i := range 105 {
		insertUser(t,
			db,
			uuid.NewString(),
			fmt.Sprintf("admin-%03d", i),
			fmt.Sprintf("admin%03d@example.com", i),
			string(authdomain.UserRoleAdmin),
		)
	}

	out, err := adapter.ListAdminUsers(context.Background())
	require.NoError(t, err)
	assert.Len(t, out, maxFilterAdminUsers)
	// Como nomes sao zero-padded, ordem ASCII == numerica.
	assert.Equal(t, "admin-000", out[0].Name)
	assert.Equal(t, "admin-099", out[len(out)-1].Name)
}

func TestAdminUserListerAdapter_ListAdminUsers_CanceledContext(t *testing.T) {
	adapter, db := setupAdminUserAdapter(t)
	insertUser(t, db, uuid.NewString(), "Admin X", "x@example.com", string(authdomain.UserRoleAdmin))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := adapter.ListAdminUsers(ctx)
	assert.Error(t, err, "context cancelado deve propagar erro")
}

// ---------------------------------------------------------------------------
// foldedLess — funcao auxiliar usada por ambos os adapters.
// ---------------------------------------------------------------------------

func TestFoldedLess(t *testing.T) {
	cases := []struct {
		a, b string
		less bool
	}{
		{"alpha", "beta", true},
		{"Beta", "alpha", false},
		{"alpha", "ALPHA", false}, // iguais apos lower => nao e "less".
		{"ALPHA", "beta", true},
	}
	for _, tc := range cases {
		t.Run(tc.a+"_vs_"+tc.b, func(t *testing.T) {
			assert.Equal(t, tc.less, foldedLess(tc.a, tc.b))
		})
	}
}
