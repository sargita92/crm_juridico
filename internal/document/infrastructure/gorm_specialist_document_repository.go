package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

type specialistDocumentModel struct {
	SpecialistID string    `gorm:"primaryKey;column:specialist_id;type:char(36)"`
	DocumentID   string    `gorm:"primaryKey;column:document_id;type:char(36)"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (specialistDocumentModel) TableName() string { return "specialist_documents" }

type GormSpecialistDocumentRepository struct {
	db *gorm.DB
}

func NewGormSpecialistDocumentRepository(db *gorm.DB) *GormSpecialistDocumentRepository {
	return &GormSpecialistDocumentRepository{db: db}
}

func (r *GormSpecialistDocumentRepository) Associate(ctx context.Context, specialistID, documentID string) error {
	model := specialistDocumentModel{
		SpecialistID: specialistID,
		DocumentID:   documentID,
		CreatedAt:    time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.ErrDocumentAlreadyAssociated
		}
		return err
	}
	return nil
}

func (r *GormSpecialistDocumentRepository) Dissociate(ctx context.Context, specialistID, documentID string) error {
	result := r.db.WithContext(ctx).
		Where("specialist_id = ? AND document_id = ?", specialistID, documentID).
		Delete(&specialistDocumentModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrDocumentNotAssociated
	}
	return nil
}

func (r *GormSpecialistDocumentRepository) DissociateAllByDocumentID(ctx context.Context, documentID string) error {
	return r.db.WithContext(ctx).
		Where("document_id = ?", documentID).
		Delete(&specialistDocumentModel{}).Error
}

func (r *GormSpecialistDocumentRepository) FindDocumentIDsBySpecialistID(ctx context.Context, specialistID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Model(&specialistDocumentModel{}).
		Where("specialist_id = ?", specialistID).
		Pluck("document_id", &ids).Error
	return ids, err
}

func (r *GormSpecialistDocumentRepository) FindSpecialistIDsByDocumentID(ctx context.Context, documentID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Model(&specialistDocumentModel{}).
		Where("document_id = ?", documentID).
		Pluck("specialist_id", &ids).Error
	return ids, err
}

func (r *GormSpecialistDocumentRepository) Exists(ctx context.Context, specialistID, documentID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&specialistDocumentModel{}).
		Where("specialist_id = ? AND document_id = ?", specialistID, documentID).
		Count(&count).Error
	return count > 0, err
}

func (r *GormSpecialistDocumentRepository) CountByDocumentID(ctx context.Context, documentID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&specialistDocumentModel{}).
		Where("document_id = ?", documentID).
		Count(&count).Error
	return int(count), err
}
