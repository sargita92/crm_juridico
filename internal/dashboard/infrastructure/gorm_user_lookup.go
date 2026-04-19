package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
)

type GormUserLookup struct{ db *gorm.DB }

// Compile-time check.
var _ application.UserLookup = (*GormUserLookup)(nil)

func NewGormUserLookup(db *gorm.DB) *GormUserLookup {
	return &GormUserLookup{db: db}
}

// UserName devolve o nome do usuário ou string vazia se não encontrado.
// Erro só é propagado para falhas reais de DB.
func (r *GormUserLookup) UserName(ctx context.Context, userID string) (string, error) {
	var u struct{ Name string }
	err := r.db.WithContext(ctx).Table("users").
		Select("name").
		Where("id = ?", userID).
		Take(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return u.Name, nil
}
