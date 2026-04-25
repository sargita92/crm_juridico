package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

type userModel struct {
	ID           string `gorm:"primaryKey;column:id;type:char(36)"`
	Name         string `gorm:"column:name;type:varchar(255);not null"`
	Email        string `gorm:"column:email;type:varchar(255);not null;uniqueIndex:idx_users_email"`
	PasswordHash string `gorm:"column:password_hash;type:varchar(255);not null"`
	Role         string `gorm:"column:role;type:enum('admin','user');not null;default:user"`
	Status       string `gorm:"column:status;type:enum('active','inactive');not null;default:active"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (userModel) TableName() string { return "users" }

func userToModel(u *domain.User) *userModel {
	return &userModel{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         string(u.Role),
		Status:       string(u.Status),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func userToDomain(m *userModel) *domain.User {
	return &domain.User{
		ID:           m.ID,
		Name:         m.Name,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		Role:         domain.UserRole(m.Role),
		Status:       domain.UserStatus(m.Status),
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(ctx context.Context, user *domain.User) error {
	model := userToModel(user)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.ErrUserEmailExists
		}
		return err
	}
	return nil
}

// Update persiste alteracoes em campos mutaveis (Name, Email, Status).
// PasswordHash nao e alterado por este metodo — fluxo separado.
//
// Erros 1062 (UNIQUE constraint) em email viram ErrUserEmailExists,
// alinhado a Create. RowsAffected == 0 retorna ErrUserNotFound.
func (r *GormUserRepository) Update(ctx context.Context, user *domain.User) error {
	model := userToModel(user)
	model.UpdatedAt = time.Now()
	res := r.db.WithContext(ctx).Model(&userModel{}).
		Where("id = ?", user.ID).
		Updates(map[string]any{
			"name":       model.Name,
			"email":      model.Email,
			"status":     model.Status,
			"updated_at": model.UpdatedAt,
		})
	if res.Error != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(res.Error, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.ErrUserEmailExists
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *GormUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var model userModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return userToDomain(&model), nil
}

func (r *GormUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var model userModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return userToDomain(&model), nil
}

func (r *GormUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&userModel{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
