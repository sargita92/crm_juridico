package domain

type TokenClaims struct {
	UserID   string
	Role     UserRole
	TenantID string
}

type TokenProvider interface {
	Generate(claims TokenClaims) (string, error)
	Validate(token string) (*TokenClaims, error)
}
