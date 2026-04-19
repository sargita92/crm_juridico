package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
	tenantDomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
	tenantInfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
)

type fixedClock struct{ t time.Time }

func (f *fixedClock) Now() time.Time { return f.t }

func seedActiveTenantMensal(t *testing.T, db *gorm.DB, valorCents int64, dia uint8, start time.Time) string {
	t.Helper()
	repo := tenantInfra.NewGormTenantRepository(db)
	tn, err := tenantDomain.NewTenant(uuid.New().String(), "Tenant Mensal", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	require.NoError(t, err)
	tn.SetBillingConfig("mensal", &valorCents, &dia, &start, true)
	require.NoError(t, repo.Create(context.Background(), tn))
	return tn.ID
}

func TestBillingScheduler_RunOnce_EndToEnd(t *testing.T) {
	repo, db := setupPaymentRepo(t)
	billingRepo := NewGormTenantBillingRepository(db)
	cal := domain.NewBrazilHolidayCalendar()

	tenantID := seedActiveTenantMensal(t, db, 50000, 10, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	clk := &fixedClock{t: time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)}
	gen := application.NewGenerateRecurringPayments(repo, billingRepo, cal, application.UUIDGenerator{}, clk)
	ref := application.NewRefreshOverdueStatuses(repo, cal, 1, clk)

	s := NewBillingScheduler("0 3 * * *", gen, ref, zap.NewNop(), time.UTC)

	nGen, nRef, err := s.RunOnce(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, nGen, 1, "deve criar ao menos competencia abril/2026")
	_ = nRef

	// segunda execucao — idempotente
	nGen2, _, err := s.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, nGen2)

	// verifica que o pagamento foi realmente persistido para o tenant
	list, err := repo.List(context.Background(), domain.ListFilters{TenantID: tenantID, Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total)
}

func TestBillingScheduler_StartStop(t *testing.T) {
	// nao executa integracao; apenas valida que Start/Stop nao panica com spec valido
	cal := domain.NewBrazilHolidayCalendar()
	billingRepo := &emptyBillingRepo{}
	payRepo := &noopPaymentRepo{}
	gen := application.NewGenerateRecurringPayments(payRepo, billingRepo, cal, application.UUIDGenerator{}, &fixedClock{t: time.Now()})
	ref := application.NewRefreshOverdueStatuses(payRepo, cal, 1, &fixedClock{t: time.Now()})
	s := NewBillingScheduler("0 3 * * *", gen, ref, zap.NewNop(), time.UTC)
	require.NoError(t, s.Start())
	s.Stop()
}

func TestBillingScheduler_InvalidSpec_ReturnsError(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	billingRepo := &emptyBillingRepo{}
	payRepo := &noopPaymentRepo{}
	gen := application.NewGenerateRecurringPayments(payRepo, billingRepo, cal, application.UUIDGenerator{}, &fixedClock{t: time.Now()})
	ref := application.NewRefreshOverdueStatuses(payRepo, cal, 1, &fixedClock{t: time.Now()})
	s := NewBillingScheduler("lixo", gen, ref, zap.NewNop(), time.UTC)
	err := s.Start()
	assert.Error(t, err)
}

type emptyBillingRepo struct{}

func (emptyBillingRepo) GetByID(context.Context, string) (*domain.TenantBilling, error) {
	return nil, domain.ErrTenantNotFound
}
func (emptyBillingRepo) ListActiveBillable(context.Context) ([]domain.TenantBilling, error) {
	return nil, nil
}

type noopPaymentRepo struct{}

func (noopPaymentRepo) Create(context.Context, *domain.Payment) error { return nil }
func (noopPaymentRepo) Update(context.Context, *domain.Payment) error { return nil }
func (noopPaymentRepo) FindByID(context.Context, string, string) (*domain.Payment, error) {
	return nil, domain.ErrPaymentNotFound
}
func (noopPaymentRepo) FindByIDAdmin(context.Context, string) (*domain.Payment, error) {
	return nil, domain.ErrPaymentNotFound
}
func (noopPaymentRepo) ExistsRecorrente(context.Context, string, time.Time) (bool, error) {
	return false, nil
}
func (noopPaymentRepo) List(context.Context, domain.ListFilters) (*domain.ListResult, error) {
	return &domain.ListResult{}, nil
}
func (noopPaymentRepo) ListAll(context.Context, domain.ListFilters) (*domain.ListResult, error) {
	return &domain.ListResult{}, nil
}
func (noopPaymentRepo) ListOverdueCandidates(context.Context, time.Time) ([]domain.Payment, error) {
	return nil, nil
}
func (noopPaymentRepo) Summary(context.Context, string, time.Time) (*domain.Summary, error) {
	return &domain.Summary{}, nil
}
