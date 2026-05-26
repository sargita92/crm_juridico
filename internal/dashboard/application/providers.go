package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
)

// TenantStatsProvider agrega as queries do dashboard do tenant.
// Todas recebem tenantID + filtro opcional de responsibleUserID (nil = dados do tenant inteiro).
type TenantStatsProvider interface {
	FunilBlock(ctx context.Context, tenantID string, userID *string, now time.Time) (*domain.FunilBlock, string, error) // retorna também o nome do funil ativo
	WhatsAppBlock(ctx context.Context, tenantID string, userID *string) (*domain.WhatsAppStats, error)
	ResponsaveisBlock(ctx context.Context, tenantID string, userID *string) ([]domain.ResponsiblePerformance, error)
	TempoFunilBlock(ctx context.Context, tenantID string, userID *string, now time.Time) ([]domain.ColumnDwell, error)
	ProdutosBlock(ctx context.Context, tenantID string, userID *string) ([]domain.ProductLeadsCount, error)
}

// AdminStatsProvider agrega as queries do dashboard do admin.
type AdminStatsProvider interface {
	TenantsBlock(ctx context.Context, now time.Time) (*domain.TenantsBlock, error)
	UsageBlock(ctx context.Context) (*domain.UsageBlock, error)
	HealthBlock(ctx context.Context, now time.Time) (*domain.HealthBlock, error)
	EspecialistasBlock(ctx context.Context) (*domain.SpecialistsBlock, error)
	FinanceiroBlock(ctx context.Context, now time.Time) (*domain.FinancialBlock, error)
}

// InfraProvider cobre o Bloco 4 admin (Prometheus/local).
type InfraProvider interface {
	Snapshot(ctx context.Context) (*domain.Infrastructure, error)
}

// UserLookup resolve nome do usuário atual (para título do Bloco 3 quando ScopeIsUser).
type UserLookup interface {
	UserName(ctx context.Context, userID string) (string, error)
}

// OperatorLister lista os operadores (não-owners) de um tenant para popular o
// seletor de usuário do dashboard (F25).
type OperatorLister interface {
	Operators(ctx context.Context, tenantID string) ([]domain.Operator, error)
}

// Clock — injetável para testes determinísticos.
type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }
