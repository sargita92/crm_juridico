package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser_WithValidData_ReturnsUser(t *testing.T) {
	user, err := NewUser("uuid-1", "João Silva", "joao@email.com", "$2a$10$hash", UserRoleUser)

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", user.ID)
	assert.Equal(t, "João Silva", user.Name)
	assert.Equal(t, "joao@email.com", user.Email)
	assert.Equal(t, "$2a$10$hash", user.PasswordHash)
	assert.Equal(t, UserRoleUser, user.Role)
	assert.Equal(t, UserStatusActive, user.Status)
	assert.False(t, user.CreatedAt.IsZero())
}

func TestNewUser_Admin_ReturnsUser(t *testing.T) {
	user, err := NewUser("uuid-1", "Admin", "admin@email.com", "$2a$10$hash", UserRoleAdmin)

	require.NoError(t, err)
	assert.Equal(t, UserRoleAdmin, user.Role)
	assert.True(t, user.IsAdmin())
}

func TestNewUser_EmptyName_ReturnsError(t *testing.T) {
	_, err := NewUser("uuid-1", "", "joao@email.com", "$2a$10$hash", UserRoleUser)
	assert.ErrorIs(t, err, ErrUserNameRequired)
}

func TestNewUser_InvalidEmail_ReturnsError(t *testing.T) {
	_, err := NewUser("uuid-1", "João", "nao-e-email", "$2a$10$hash", UserRoleUser)
	assert.ErrorIs(t, err, ErrInvalidEmail)
}

func TestNewUser_EmptyEmail_ReturnsError(t *testing.T) {
	_, err := NewUser("uuid-1", "João", "", "$2a$10$hash", UserRoleUser)
	assert.ErrorIs(t, err, ErrInvalidEmail)
}

func TestNewUser_EmptyPassword_ReturnsError(t *testing.T) {
	_, err := NewUser("uuid-1", "João", "joao@email.com", "", UserRoleUser)
	assert.ErrorIs(t, err, ErrPasswordRequired)
}

func TestUser_IsActive(t *testing.T) {
	user := &User{Status: UserStatusActive}
	assert.True(t, user.IsActive())

	user.Status = UserStatusInactive
	assert.False(t, user.IsActive())
}

func TestUser_IsAdmin(t *testing.T) {
	user := &User{Role: UserRoleAdmin}
	assert.True(t, user.IsAdmin())

	user.Role = UserRoleUser
	assert.False(t, user.IsAdmin())
}
