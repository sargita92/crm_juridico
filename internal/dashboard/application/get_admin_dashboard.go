package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
)

type GetAdminDashboard struct {
	provider AdminStatsProvider
	infra    InfraProvider
	clock    Clock
}

func NewGetAdminDashboard(p AdminStatsProvider, i InfraProvider, c Clock) *GetAdminDashboard {
	return &GetAdminDashboard{provider: p, infra: i, clock: c}
}

func (uc *GetAdminDashboard) Execute(ctx context.Context) (*domain.AdminStats, error) {
	ctx, span := tracer.Start(ctx, "GetAdminDashboard")
	defer span.End()
	now := uc.clock.Now()

	tenants, err := uc.provider.TenantsBlock(ctx, now)
	if err != nil {
		return nil, err
	}
	usage, err := uc.provider.UsageBlock(ctx)
	if err != nil {
		return nil, err
	}
	health, err := uc.provider.HealthBlock(ctx, now)
	if err != nil {
		return nil, err
	}
	spec, err := uc.provider.EspecialistasBlock(ctx)
	if err != nil {
		return nil, err
	}
	fin, err := uc.provider.FinanceiroBlock(ctx, now)
	if err != nil {
		return nil, err
	}
	infra, err := uc.infra.Snapshot(ctx)
	if err != nil {
		return nil, err
	}

	return &domain.AdminStats{
		Bloco1_Tenants:       *tenants,
		Bloco2_Uso:           *usage,
		Bloco3_Health:        *health,
		Bloco4_Infra:         *infra,
		Bloco5_Especialistas: *spec,
		Bloco6_Financeiro:    *fin,
	}, nil
}
