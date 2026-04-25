package infrastructure

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type JWTProvider struct {
	secret     []byte
	expiration time.Duration
}

func NewJWTProvider(secret string, expiration time.Duration) *JWTProvider {
	return &JWTProvider{
		secret:     []byte(secret),
		expiration: expiration,
	}
}

type jwtClaims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id,omitempty"`
	jwt.RegisteredClaims
}

func (p *JWTProvider) Generate(claims domain.TokenClaims) (string, error) {
	now := time.Now()
	c := jwtClaims{
		UserID:   claims.UserID,
		Email:    claims.Email,
		Role:     string(claims.Role),
		TenantID: claims.TenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(p.expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(p.secret)
}

func (p *JWTProvider) Validate(tokenString string) (*domain.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return p.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return &domain.TokenClaims{
		UserID:   claims.UserID,
		Email:    claims.Email,
		Role:     domain.UserRole(claims.Role),
		TenantID: claims.TenantID,
	}, nil
}
