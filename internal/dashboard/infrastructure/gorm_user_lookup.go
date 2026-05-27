package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
)

type GormUserLookup struct{ db *gorm.DB }

// Compile-time checks.
var (
	_ application.UserLookup     = (*GormUserLookup)(nil)
	_ application.OperatorLister = (*GormUserLookup)(nil)
)

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

// Operators lista os usuários não-owner do tenant, ordenados por nome — itens do
// seletor de operador do dashboard (F25).
func (r *GormUserLookup) Operators(ctx context.Context, tenantID string) ([]domain.Operator, error) {
	rows := []domain.Operator{}
	err := r.db.WithContext(ctx).
		Table("user_tenants AS ut").
		Select("u.id AS id, u.name AS name").
		Joins("JOIN users u ON u.id = ut.user_id").
		Where("ut.tenant_id = ? AND ut.is_owner = ?", tenantID, false).
		Order("u.name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
