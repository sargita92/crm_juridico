package domain

// TokenClaims sao os atributos do usuario autenticado embutidos no JWT.
//
// Email e um snapshot do email no memento do login. Existe principalmente
// para a F12 (logs admin) — o `actor_email` precisa estar disponivel em
// qualquer publish de auditoria sem custo extra de SELECT, e o JWT ja viaja
// com a request. Aceita-se o trade-off de ~50 bytes a mais no token e o
// fato de o snapshot poder ficar defasado em relacao ao banco (rotacionar o
// token resolve quando o email muda).
type TokenClaims struct {
	UserID   string
	Email    string
	Role     UserRole
	TenantID string
}

type TokenProvider interface {
	Generate(claims TokenClaims) (string, error)
	Validate(token string) (*TokenClaims, error)
}
