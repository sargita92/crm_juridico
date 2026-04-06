package infrastructure

import (
	"context"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
)

type UserNameAdapter struct {
	userRepo authdomain.UserRepository
}

func NewUserNameAdapter(userRepo authdomain.UserRepository) *UserNameAdapter {
	return &UserNameAdapter{userRepo: userRepo}
}

func (a *UserNameAdapter) FindNameByID(ctx context.Context, userID string) (string, error) {
	user, err := a.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return user.Name, nil
}
