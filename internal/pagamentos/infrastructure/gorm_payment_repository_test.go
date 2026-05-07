package infrastructure

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	tenantDomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
	tenantInfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
)

var sharedContainer *testhelper.MySQLContainer

func TestMain(m *testing.M) {
	short := false
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "-short" || strings.HasPrefix(arg, "-test.short=") || strings.HasPrefix(arg, "-short=") {
			short = true
			break
		}
	}
	if !short {
		ctx := context.Background()
		sharedContainer = testhelper.NewMySQLContainerForMain(ctx)
		code := m.Run()
		_ = sharedContainer.Container.Terminate(ctx)
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func setupPaymentRepo(t *testing.T) (*GormPaymentRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()
	require.NoError(t, database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath()))

	// FK-safe order
	db.Exec("DELETE FROM payments")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	return NewGormPaymentRepository(db), db
}

func seedTenant(t *testing.T, db *gorm.DB) string {
	t.Helper()
	repo := tenantInfra.NewGormTenantRepository(db)
	tn, err := tenantDomain.NewTenant(uuid.New().String(), "Tenant Teste", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), tn))
	return tn.ID
}

func seedTenantWithName(t *testing.T, db *gorm.DB, name string) string {
	t.Helper()
	repo := tenantInfra.NewGormTenantRepository(db)
	tn, err := tenantDomain.NewTenant(uuid.New().String(), name, tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), tn))
	return tn.ID
}

func TestCreateAndFindByID(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantID := seedTenant(t, db)

	p, err := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "Setup", 15000,
		time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), "obs inicial")
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), p))

	got, err := repo.FindByID(context.Background(), tenantID, p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, int64(15000), got.ValorCents)
	assert.Equal(t, "Setup", got.Descricao)
	assert.Equal(t, "obs inicial", got.Observacao)
	assert.Equal(t, domain.StatusPendente, got.Status)
}

func TestFindByID_NotFound(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantID := seedTenant(t, db)
	_, err := repo.FindByID(context.Background(), tenantID, uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrPaymentNotFound)
}

func TestFindByID_TenantMismatch(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantA := seedTenant(t, db)
	tenantB := seedTenant(t, db)
	p, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantA, "X", 1000, time.Now(), "")
	require.NoError(t, repo.Create(context.Background(), p))

	_, err := repo.FindByID(context.Background(), tenantB, p.ID)
	assert.ErrorIs(t, err, domain.ErrPaymentNotFound)
}

func TestFindByIDAdmin_IgnoresTenant(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantID := seedTenant(t, db)
	p, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "X", 1000, time.Now(), "")
	require.NoError(t, repo.Create(context.Background(), p))

	got, err := repo.FindByIDAdmin(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ID)
}

func TestUpdate_Persists(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantID := seedTenant(t, db)
	p, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "X", 1000, time.Now(), "")
	require.NoError(t, repo.Create(context.Background(), p))

	require.NoError(t, p.MarkAsPaid("user-1", time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)))
	require.NoError(t, repo.Update(context.Background(), p))

	got, _ := repo.FindByID(context.Background(), tenantID, p.ID)
	assert.Equal(t, domain.StatusPago, got.Status)
	require.NotNil(t, got.PaidByUserID)
	assert.Equal(t, "user-1", *got.PaidByUserID)
	require.NotNil(t, got.DataPagamento)
}

func TestExistsRecorrente_Idempotency(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantID := seedTenant(t, db)
	comp := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	exists, err := repo.ExistsRecorrente(context.Background(), tenantID, comp)
	require.NoError(t, err)
	assert.False(t, exists)

	p, _ := domain.NewRecorrentePayment(uuid.NewString(), tenantID, 50000, comp, comp.AddDate(0, 0, 10))
	require.NoError(t, repo.Create(context.Background(), p))

	exists, err = repo.ExistsRecorrente(context.Background(), tenantID, comp)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestCreate_DuplicateRecorrente_Rejected(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantID := seedTenant(t, db)
	comp := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	venc := comp.AddDate(0, 0, 10)

	p1, _ := domain.NewRecorrentePayment(uuid.NewString(), tenantID, 50000, comp, venc)
	require.NoError(t, repo.Create(context.Background(), p1))

	p2, _ := domain.NewRecorrentePayment(uuid.NewString(), tenantID, 50000, comp, venc)
	err := repo.Create(context.Background(), p2)
	assert.Error(t, err) // unique key conflict
}

func TestCreate_AvulsosPodemRepetirSemCompetencia(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantID := seedTenant(t, db)
	for i := 0; i < 3; i++ {
		p, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "Setup", int64((i+1)*1000), time.Now(), "")
		require.NoError(t, repo.Create(context.Background(), p))
	}
}

func TestList_RequiresTenant(t *testing.T) {
	repo, _ := setupPaymentRepo(t)
	_, err := repo.List(context.Background(), domain.ListFilters{})
	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}

func TestList_FiltersByStatus(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantID := seedTenant(t, db)

	for i := 0; i < 3; i++ {
		p, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "A", int64((i+1)*1000), time.Now(), "")
		require.NoError(t, repo.Create(context.Background(), p))
	}
	paid, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "P", 9999, time.Now(), "")
	require.NoError(t, paid.MarkAsPaid("u", time.Now()))
	require.NoError(t, repo.Create(context.Background(), paid))

	pendente := domain.StatusPendente
	res, err := repo.List(context.Background(), domain.ListFilters{TenantID: tenantID, Status: &pendente})
	require.NoError(t, err)
	assert.EqualValues(t, 3, res.Total)
	assert.Len(t, res.Items, 3)
}

func TestList_Pagination(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantID := seedTenant(t, db)
	for i := 0; i < 5; i++ {
		p, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "A", 1000, time.Date(2026, 4, i+1, 0, 0, 0, 0, time.UTC), "")
		require.NoError(t, repo.Create(context.Background(), p))
	}
	res, err := repo.List(context.Background(), domain.ListFilters{TenantID: tenantID, Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.EqualValues(t, 5, res.Total)
	assert.Len(t, res.Items, 2)
}

func TestListAll_NoTenantFilter_ReturnsAll(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantA := seedTenant(t, db)
	tenantB := seedTenant(t, db)
	pA, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantA, "A", 1000, time.Now(), "")
	pB, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantB, "B", 1000, time.Now(), "")
	require.NoError(t, repo.Create(context.Background(), pA))
	require.NoError(t, repo.Create(context.Background(), pB))

	res, err := repo.ListAll(context.Background(), domain.ListFilters{})
	require.NoError(t, err)
	assert.EqualValues(t, 2, res.Total)
}

func TestListAll_FilterByTenant(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantA := seedTenant(t, db)
	tenantB := seedTenant(t, db)
	pA, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantA, "A", 1000, time.Now(), "")
	pB, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantB, "B", 1000, time.Now(), "")
	require.NoError(t, repo.Create(context.Background(), pA))
	require.NoError(t, repo.Create(context.Background(), pB))

	res, err := repo.ListAll(context.Background(), domain.ListFilters{TenantID: tenantA})
	require.NoError(t, err)
	assert.EqualValues(t, 1, res.Total)
	assert.Equal(t, pA.ID, res.Items[0].ID)
}

func TestListOverdueCandidates(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantID := seedTenant(t, db)

	old, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "Velho", 1000,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), "")
	require.NoError(t, repo.Create(context.Background(), old))

	fresh, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "Novo", 1000,
		time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), "")
	require.NoError(t, repo.Create(context.Background(), fresh))

	paid, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "Pago", 1000,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), "")
	require.NoError(t, paid.MarkAsPaid("u", time.Now()))
	require.NoError(t, repo.Create(context.Background(), paid))

	got, err := repo.ListOverdueCandidates(context.Background(), time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, old.ID, got[0].ID)
}

func TestGormPaymentRepository_GlobalSummary(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)

	t1 := seedTenantWithName(t, db, "Tenant Um")
	t2 := seedTenantWithName(t, db, "Tenant Dois")

	paidAt := now
	// t1: pago no ano (200000), pendente 50000, atrasado 30000
	require.NoError(t, repo.Create(ctx, &domain.Payment{
		ID: uuid.NewString(), TenantID: t1, Tipo: domain.TypeAvulso, Descricao: "x",
		ValorCents: 200000, Status: domain.StatusPago,
		DataVencimento: now, DataPagamento: &paidAt,
		CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, repo.Create(ctx, &domain.Payment{
		ID: uuid.NewString(), TenantID: t1, Tipo: domain.TypeAvulso, Descricao: "y",
		ValorCents: 50000, Status: domain.StatusPendente,
		DataVencimento: now, CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, repo.Create(ctx, &domain.Payment{
		ID: uuid.NewString(), TenantID: t1, Tipo: domain.TypeAvulso, Descricao: "z",
		ValorCents: 30000, Status: domain.StatusAtrasado,
		DataVencimento: now, CreatedAt: now, UpdatedAt: now,
	}))
	// t2: atrasado 80000 (maior, deve aparecer em primeiro no top)
	require.NoError(t, repo.Create(ctx, &domain.Payment{
		ID: uuid.NewString(), TenantID: t2, Tipo: domain.TypeAvulso, Descricao: "a",
		ValorCents: 80000, Status: domain.StatusAtrasado,
		DataVencimento: now, CreatedAt: now, UpdatedAt: now,
	}))

	got, err := repo.GlobalSummary(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, int64(200000), got.TotalPagoAnoCents)
	assert.Equal(t, int64(50000), got.TotalPendenteCents)
	assert.Equal(t, int64(110000), got.TotalAtrasadoCents) // 30000 + 80000
	require.Len(t, got.TopOverdue, 2)
	assert.Equal(t, t2, got.TopOverdue[0].TenantID) // ordenado desc
	assert.Equal(t, int64(80000), got.TopOverdue[0].ValorCents)
	assert.Equal(t, "Tenant Dois", got.TopOverdue[0].TenantName)
	assert.Equal(t, t1, got.TopOverdue[1].TenantID)
	assert.Equal(t, int64(30000), got.TopOverdue[1].ValorCents)
	assert.Equal(t, "Tenant Um", got.TopOverdue[1].TenantName)
}

func TestSummary(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	tenantID := seedTenant(t, db)
	today := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	// pago no ano corrente
	p1, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "x", 20000, today.AddDate(0, -1, 0), "")
	require.NoError(t, p1.MarkAsPaid("u", today.AddDate(0, -1, 0)))
	require.NoError(t, repo.Create(context.Background(), p1))

	// pago em ano anterior - nao deve contar
	pOld, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "old", 99999, today.AddDate(-1, 0, 0), "")
	require.NoError(t, pOld.MarkAsPaid("u", today.AddDate(-1, 0, 0)))
	require.NoError(t, repo.Create(context.Background(), pOld))

	// pendente
	p2, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "y", 10000, today.AddDate(0, 0, 10), "")
	require.NoError(t, repo.Create(context.Background(), p2))

	// atrasado
	p3, _ := domain.NewAvulsoPayment(uuid.NewString(), tenantID, "z", 5000, today.AddDate(0, 0, -5), "")
	p3.Status = domain.StatusAtrasado
	require.NoError(t, repo.Create(context.Background(), p3))

	sum, err := repo.Summary(context.Background(), tenantID, today)
	require.NoError(t, err)
	assert.Equal(t, int64(20000), sum.TotalPagoAnoCents)
	assert.Equal(t, int64(10000), sum.TotalPendenteCents)
	assert.Equal(t, int64(5000), sum.TotalAtrasadoCents)
	assert.True(t, sum.HasAtrasado)
	assert.True(t, sum.HasPendente)
}
