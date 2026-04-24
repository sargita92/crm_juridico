package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

// actorFromContext extrai actor_email e actor_user_id do TokenClaims do
// contexto. Retorna ("", nil) quando nao ha claims (ex.: testes que nao
// chamam middleware.SetClaimsForTest). E responsabilidade do RegisterUC do
// audit validar que ActorEmail nao e vazio — esta funcao apenas reflete o
// que o context oferece.
func actorFromContext(ctx context.Context) (string, *string) {
	claims := middleware.GetClaims(ctx)
	if claims == nil {
		return "", nil
	}
	uid := claims.UserID
	var uidPtr *string
	if uid != "" {
		uidPtr = &uid
	}
	return claims.Email, uidPtr
}

// snapshotTenant produz um mapa shallow com os campos auditaveis do tenant.
// Usado pelo update UC como entrada do BuildDiff. Mantem tipos string para
// facilitar diff legivel ("antes": "PF", "depois": "PJ"). Campos transientes
// como UpdatedAt sao naturalmente excluidos pelo BuildDiff (lista
// diffIgnoredKeys).
func snapshotTenant(t *domain.Tenant) map[string]any {
	return map[string]any{
		"Name":               t.Name,
		"Type":               string(t.Type),
		"Document":           t.Document,
		"Status":             string(t.Status),
		"Plano":              t.Plano,
		"ValorCobrancaCents": derefInt64(t.ValorCobrancaCents),
		"DiaVencimento":      derefUint8(t.DiaVencimento),
		"ExibirPagamentos":   t.ExibirPagamentos,
	}
}

func derefInt64(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func derefUint8(p *uint8) any {
	if p == nil {
		return nil
	}
	return *p
}
