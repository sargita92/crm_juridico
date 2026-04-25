package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// actorFromContext extrai actor_email e actor_user_id do TokenClaims do
// contexto. Retorna ("", nil) quando nao ha claims (ex.: testes que nao
// chamam middleware.SetClaimsForTest). E responsabilidade do RegisterUC do
// audit validar que ActorEmail nao e vazio — esta funcao apenas reflete o
// que o context oferece.
//
// Duplicado intencionalmente em internal/tenant/application/audit_helpers.go
// para manter os modulos `auth` e `tenant` desacoplados — extrair para
// shared traria uma dependencia transversal pequena demais para justificar
// o pacote novo.
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

// snapshotAdminUser produz um mapa shallow com os campos auditaveis de um
// usuario admin. Usado pelo UpdateAdminUser como entrada do BuildDiff.
// PasswordHash e excluido por seguranca (chave proibida do BuildDiff
// inclui `password_hash`); UpdatedAt e excluido por ser transiente.
func snapshotAdminUser(u *domain.User) map[string]any {
	return map[string]any{
		"name":   u.Name,
		"email":  u.Email,
		"role":   string(u.Role),
		"status": string(u.Status),
	}
}
