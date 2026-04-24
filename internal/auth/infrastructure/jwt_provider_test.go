package infrastructure

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

func TestJWTProvider_Generate_And_Validate(t *testing.T) {
	provider := NewJWTProvider("test-secret", 24*time.Hour)

	claims := domain.TokenClaims{
		UserID:   "user-123",
		Email:    "joao@example.com",
		Role:     domain.UserRoleUser,
		TenantID: "tenant-456",
	}

	tokenStr, err := provider.Generate(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	result, err := provider.Validate(tokenStr)
	require.NoError(t, err)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "joao@example.com", result.Email)
	assert.Equal(t, domain.UserRoleUser, result.Role)
	assert.Equal(t, "tenant-456", result.TenantID)
}

func TestJWTProvider_Generate_WithoutEmail_RoundTrips(t *testing.T) {
	// Tokens emitidos antes da F12 (sem email) devem continuar validos:
	// o claim e omitempty no JSON, entao deserializa como string vazia.
	provider := NewJWTProvider("test-secret", 24*time.Hour)

	claims := domain.TokenClaims{
		UserID: "user-123",
		Role:   domain.UserRoleUser,
	}

	tokenStr, err := provider.Generate(claims)
	require.NoError(t, err)

	result, err := provider.Validate(tokenStr)
	require.NoError(t, err)
	assert.Empty(t, result.Email)
}

func TestJWTProvider_Generate_WithoutTenantID(t *testing.T) {
	provider := NewJWTProvider("test-secret", 24*time.Hour)

	claims := domain.TokenClaims{
		UserID: "user-123",
		Role:   domain.UserRoleUser,
	}

	tokenStr, err := provider.Generate(claims)
	require.NoError(t, err)

	result, err := provider.Validate(tokenStr)
	require.NoError(t, err)
	assert.Empty(t, result.TenantID)
}

func TestJWTProvider_Validate_ExpiredToken(t *testing.T) {
	provider := NewJWTProvider("test-secret", -1*time.Hour)

	claims := domain.TokenClaims{UserID: "user-123", Role: domain.UserRoleUser}
	tokenStr, err := provider.Generate(claims)
	require.NoError(t, err)

	_, err = provider.Validate(tokenStr)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTProvider_Validate_WrongSignature(t *testing.T) {
	provider1 := NewJWTProvider("secret-1", 24*time.Hour)
	provider2 := NewJWTProvider("secret-2", 24*time.Hour)

	claims := domain.TokenClaims{UserID: "user-123", Role: domain.UserRoleUser}
	tokenStr, err := provider1.Generate(claims)
	require.NoError(t, err)

	_, err = provider2.Validate(tokenStr)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTProvider_Validate_AlgNone_Rejected(t *testing.T) {
	provider := NewJWTProvider("test-secret", 24*time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id": "user-123",
		"role":    "admin",
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = provider.Validate(tokenStr)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTProvider_Validate_TamperedClaims_Rejected(t *testing.T) {
	provider := NewJWTProvider("test-secret", 24*time.Hour)

	claims := domain.TokenClaims{UserID: "user-123", Role: domain.UserRoleUser}
	tokenStr, err := provider.Generate(claims)
	require.NoError(t, err)

	// Tamper with the token by changing a character in the payload
	bytes := []byte(tokenStr)
	if len(bytes) > 30 {
		bytes[25] = 'X'
	}
	tamperedToken := string(bytes)

	_, err = provider.Validate(tamperedToken)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTProvider_Token_HasExpiration(t *testing.T) {
	provider := NewJWTProvider("test-secret", 2*time.Hour)

	claims := domain.TokenClaims{UserID: "user-123", Role: domain.UserRoleUser}
	tokenStr, err := provider.Generate(claims)
	require.NoError(t, err)

	parsed, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(token *jwt.Token) (any, error) {
		return []byte("test-secret"), nil
	})
	require.NoError(t, err)

	jwtc := parsed.Claims.(*jwtClaims)
	assert.NotNil(t, jwtc.ExpiresAt)
	assert.WithinDuration(t, time.Now().Add(2*time.Hour), jwtc.ExpiresAt.Time, 5*time.Second)
}
