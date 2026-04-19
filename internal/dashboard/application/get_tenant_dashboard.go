package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
)

type TenantInput struct {
	TenantID string
	UserID   string
	IsOwner  bool // owner/admin do tenant → sem filtro por user
}

type GetTenantDashboard struct {
	provider TenantStatsProvider
	users    UserLookup
	clock    Clock
}

func NewGetTenantDashboard(p TenantStatsProvider, u UserLookup, c Clock) *GetTenantDashboard {
	return &GetTenantDashboard{provider: p, users: u, clock: c}
}

func (uc *GetTenantDashboard) Execute(ctx context.Context, in TenantInput) (*domain.TenantStats, error) {
	ctx, span := tracer.Start(ctx, "GetTenantDashboard")
	defer span.End()
	if in.TenantID == "" {
		return nil, domain.ErrTenantRequired
	}
	if in.UserID == "" {
		return nil, domain.ErrUserRequired
	}
	span.SetAttributes(attribute.String("tenant_id", in.TenantID), attribute.Bool("is_owner", in.IsOwner))

	var userFilter *string
	if !in.IsOwner {
		uid := in.UserID
		userFilter = &uid
	}
	now := uc.clock.Now()

	funil, funilName, err := uc.provider.FunilBlock(ctx, in.TenantID, userFilter, now)
	if err != nil {
		return nil, err
	}
	whats, err := uc.provider.WhatsAppBlock(ctx, in.TenantID, userFilter)
	if err != nil {
		return nil, err
	}
	resp, err := uc.provider.ResponsaveisBlock(ctx, in.TenantID, userFilter)
	if err != nil {
		return nil, err
	}
	tempo, err := uc.provider.TempoFunilBlock(ctx, in.TenantID, userFilter, now)
	if err != nil {
		return nil, err
	}
	prod, err := uc.provider.ProdutosBlock(ctx, in.TenantID, userFilter)
	if err != nil {
		return nil, err
	}

	out := &domain.TenantStats{
		Bloco1_Funil:        *funil,
		Bloco2_WhatsApp:     *whats,
		Bloco3_Responsaveis: resp,
		Bloco4_TempoFunil:   tempo,
		Bloco5_Produtos:     prod,
		ActiveFunnelName:    funilName,
		ScopeIsUser:         userFilter != nil,
	}
	if out.ScopeIsUser {
		name, err := uc.users.UserName(ctx, in.UserID)
		if err == nil {
			out.CurrentUserName = name
		}
	}
	return out, nil
}
