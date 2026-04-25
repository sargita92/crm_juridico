package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// actorFromContext extrai actor_email e actor_user_id do TokenClaims do
// contexto. Retorna ("", nil) quando nao ha claims (ex.: testes que nao
// chamam middleware.SetClaimsForTest). E responsabilidade do
// RegisterAuditLogUseCase validar que ActorEmail nao e vazio — esta
// funcao apenas reflete o que o context oferece.
//
// Duplicado em internal/{tenant,auth}/application/audit_helpers.go para
// manter os modulos desacoplados; extrair para shared traria
// dependencia transversal pequena demais para justificar pacote novo.
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
