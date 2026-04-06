package infrastructure

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

type documentModel struct {
	ID        string `gorm:"primaryKey;column:id;type:char(36)"`
	Name      string `gorm:"column:name;type:varchar(255);not null"`
	Type      string `gorm:"column:type;type:varchar(10);not null"`
	FilePath  string `gorm:"column:file_path;type:varchar(500);not null"`
	FileSize  int64  `gorm:"column:file_size;type:bigint;not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (documentModel) TableName() string { return "documents" }

func docToModel(d *domain.Document) *documentModel {
	return &documentModel{
		ID:        d.ID,
		Name:      d.Name,
		Type:      d.Type,
		FilePath:  d.FilePath,
		FileSize:  d.FileSize,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

func docToDomain(m *documentModel) *domain.Document {
	return &domain.Document{
		ID:        m.ID,
		Name:      m.Name,
		Type:      m.Type,
		FilePath:  m.FilePath,
		FileSize:  m.FileSize,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type GormDocumentRepository struct {
	db *gorm.DB
}

func NewGormDocumentRepository(db *gorm.DB) *GormDocumentRepository {
	return &GormDocumentRepository{db: db}
}

func (r *GormDocumentRepository) Create(ctx context.Context, doc *domain.Document) error {
	model := docToModel(doc)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormDocumentRepository) FindByID(ctx context.Context, id string) (*domain.Document, error) {
	var model documentModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrDocumentNotFound
		}
		return nil, err
	}
	return docToDomain(&model), nil
}

func (r *GormDocumentRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&documentModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrDocumentNotFound
	}
	return nil
}

func (r *GormDocumentRepository) FindWithFilter(ctx context.Context, filter domain.DocumentFilter) (*domain.DocumentList, error) {
	query := r.db.WithContext(ctx).Model(&documentModel{})

	if filter.Search != "" {
		escaped := escapeLike(filter.Search)
		search := "%" + escaped + "%"
		query = query.Where("name LIKE ?", search)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}

	offset := (page - 1) * limit

	var models []documentModel
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}

	documents := make([]domain.Document, len(models))
	for i := range models {
		documents[i] = *docToDomain(&models[i])
	}

	return &domain.DocumentList{
		Documents: documents,
		Total:     total,
		Page:      page,
		Limit:     limit,
	}, nil
}

func (r *GormDocumentRepository) FindByIDs(ctx context.Context, ids []string) ([]domain.Document, error) {
	if len(ids) == 0 {
		return []domain.Document{}, nil
	}

	var models []documentModel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&models).Error; err != nil {
		return nil, err
	}

	documents := make([]domain.Document, len(models))
	for i := range models {
		documents[i] = *docToDomain(&models[i])
	}
	return documents, nil
}
