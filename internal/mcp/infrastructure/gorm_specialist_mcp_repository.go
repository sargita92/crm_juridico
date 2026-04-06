package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

type specialistMcpModel struct {
	SpecialistID string    `gorm:"primaryKey;column:specialist_id;type:char(36)"`
	McpID        string    `gorm:"primaryKey;column:mcp_id;type:char(36)"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (specialistMcpModel) TableName() string { return "specialist_mcps" }

type GormSpecialistMcpRepository struct {
	db *gorm.DB
}

func NewGormSpecialistMcpRepository(db *gorm.DB) *GormSpecialistMcpRepository {
	return &GormSpecialistMcpRepository{db: db}
}

func (r *GormSpecialistMcpRepository) Associate(ctx context.Context, specialistID, mcpID string) error {
	model := specialistMcpModel{SpecialistID: specialistID, McpID: mcpID, CreatedAt: time.Now()}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.ErrMcpAlreadyAssociated
		}
		return err
	}
	return nil
}

func (r *GormSpecialistMcpRepository) Dissociate(ctx context.Context, specialistID, mcpID string) error {
	result := r.db.WithContext(ctx).Where("specialist_id = ? AND mcp_id = ?", specialistID, mcpID).Delete(&specialistMcpModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrMcpNotAssociated
	}
	return nil
}

func (r *GormSpecialistMcpRepository) DissociateAllByMcpID(ctx context.Context, mcpID string) error {
	return r.db.WithContext(ctx).Where("mcp_id = ?", mcpID).Delete(&specialistMcpModel{}).Error
}

func (r *GormSpecialistMcpRepository) FindMcpIDsBySpecialistID(ctx context.Context, specialistID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Model(&specialistMcpModel{}).Where("specialist_id = ?", specialistID).Pluck("mcp_id", &ids).Error
	return ids, err
}

func (r *GormSpecialistMcpRepository) FindSpecialistIDsByMcpID(ctx context.Context, mcpID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Model(&specialistMcpModel{}).Where("mcp_id = ?", mcpID).Pluck("specialist_id", &ids).Error
	return ids, err
}

func (r *GormSpecialistMcpRepository) Exists(ctx context.Context, specialistID, mcpID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&specialistMcpModel{}).Where("specialist_id = ? AND mcp_id = ?", specialistID, mcpID).Count(&count).Error
	return count > 0, err
}

func (r *GormSpecialistMcpRepository) CountByMcpID(ctx context.Context, mcpID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&specialistMcpModel{}).Where("mcp_id = ?", mcpID).Count(&count).Error
	return int(count), err
}
