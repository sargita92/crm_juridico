package domain

import (
	"errors"
	"net/mail"
	"time"
)

type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserEmailExists  = errors.New("user with this email already exists")
	ErrUserInactive     = errors.New("user is inactive")
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrUserNameRequired = errors.New("user name is required")
	ErrPasswordRequired = errors.New("password is required")
	ErrInvalidCredentials = errors.New("invalid credentials")

	ErrInviteTokenNotFound = errors.New("invite token not found")
	ErrInviteTokenExpired  = errors.New("invite token has expired")
	ErrInviteTokenUsed     = errors.New("invite token has already been used")
	ErrTenantIDRequired    = errors.New("tenant id is required")
	ErrUserIDRequired      = errors.New("user id is required")
	ErrGroupIDRequired     = errors.New("group id is required")
)

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         UserRole
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserTenant struct {
	UserID      string
	TenantID    string
	IsOwner     bool
	WhatsAppID  string
}

func NewUser(id, name, email, passwordHash string, role UserRole) (*User, error) {
	if name == "" {
		return nil, ErrUserNameRequired
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, ErrInvalidEmail
	}
	if passwordHash == "" {
		return nil, ErrPasswordRequired
	}

	now := time.Now()
	return &User{
		ID:           id,
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		Status:       UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}
