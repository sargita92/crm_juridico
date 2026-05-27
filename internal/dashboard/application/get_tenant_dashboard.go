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

	// ViewUserID, quando setado, faz o owner ver o dashboard de um operador
	// específico (drill-down). Só tem efeito para owner; é ignorado para
	// não-owner (que fica sempre travado no próprio UserID). Deve ser validado
	// pelo handler (pertencimento ao tenant) antes de chegar aqui.
	ViewUserID *string
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
	switch {
	case !in.IsOwner:
		// não-owner: sempre travado no próprio usuário (ignora ViewUserID)
		uid := in.UserID
		userFilter = &uid
	case in.ViewUserID != nil:
		// owner drillando num operador específico
		userFilter = in.ViewUserID
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
		Bloco1_Funil:      *funil,
		Bloco2_WhatsApp:   *whats,
		Bloco3_Responsive: resp,
		Bloco4_TempoFunil: tempo,
		Bloco5_Produtos:   prod,
		ActiveFunnelName:  funilName,
		ScopeIsUser:       userFilter != nil,
	}
	if out.ScopeIsUser {
		// resolve o nome do usuário efetivamente filtrado (próprio, p/ não-owner;
		// operador escolhido, p/ owner em drill-down)
		name, err := uc.users.UserName(ctx, *userFilter)
		if err == nil {
			out.CurrentUserName = name
		}
	}
	return out, nil
}
