package infrastructure

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

type specialistModel struct {
	ID                            string `gorm:"primaryKey;column:id;type:char(36)"`
	Name                          string `gorm:"column:name;type:varchar(255);not null"`
	Description                   string `gorm:"column:description;type:varchar(500);not null;default:''"`
	Prompt                        string `gorm:"column:prompt;type:text;not null"`
	Status                        string `gorm:"column:status;type:enum('active','inactive');not null;default:active"`
	CrossSellEnabled              bool   `gorm:"column:cross_sell_enabled;type:tinyint(1);not null;default:0"`
	CrossSellMode                 string `gorm:"column:cross_sell_mode;type:varchar(16);not null;default:'announce'"`
	CrossSellAnnouncementTemplate string `gorm:"column:cross_sell_announcement_template;type:text"`
	AllowAICrossSellSuggestion    bool   `gorm:"column:allow_ai_cross_sell_suggestion;type:tinyint(1);not null;default:0"`
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
}

func (specialistModel) TableName() string { return "specialists" }

func toModel(s *domain.Specialist) *specialistModel {
	return &specialistModel{
		ID:                            s.ID,
		Name:                          s.Name,
		Description:                   s.Description,
		Prompt:                        s.Prompt,
		Status:                        string(s.Status),
		CrossSellEnabled:              s.CrossSellEnabled,
		CrossSellMode:                 string(s.CrossSellMode),
		CrossSellAnnouncementTemplate: s.CrossSellAnnouncementTemplate,
		AllowAICrossSellSuggestion:    s.AllowAICrossSellSuggestion,
		CreatedAt:                     s.CreatedAt,
		UpdatedAt:                     s.UpdatedAt,
	}
}

func toDomain(m *specialistModel) *domain.Specialist {
	return &domain.Specialist{
		ID:                            m.ID,
		Name:                          m.Name,
		Description:                   m.Description,
		Prompt:                        m.Prompt,
		Status:                        domain.SpecialistStatus(m.Status),
		CrossSellEnabled:              m.CrossSellEnabled,
		CrossSellMode:                 domain.CrossSellMode(m.CrossSellMode),
		CrossSellAnnouncementTemplate: m.CrossSellAnnouncementTemplate,
		AllowAICrossSellSuggestion:    m.AllowAICrossSellSuggestion,
		CreatedAt:                     m.CreatedAt,
		UpdatedAt:                     m.UpdatedAt,
	}
}

type GormSpecialistRepository struct {
	db *gorm.DB
}

func NewGormSpecialistRepository(db *gorm.DB) *GormSpecialistRepository {
	return &GormSpecialistRepository{db: db}
}

func (r *GormSpecialistRepository) Create(ctx context.Context, specialist *domain.Specialist) error {
	model := toModel(specialist)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormSpecialistRepository) FindByID(ctx context.Context, id string) (*domain.Specialist, error) {
	var model specialistModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSpecialistNotFound
		}
		return nil, err
	}
	return toDomain(&model), nil
}

func (r *GormSpecialistRepository) Update(ctx context.Context, specialist *domain.Specialist) error {
	model := toModel(specialist)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrSpecialistNotFound
	}
	return nil
}

func (r *GormSpecialistRepository) FindWithFilter(ctx context.Context, filter domain.SpecialistFilter) (*domain.SpecialistList, error) {
	query := r.db.WithContext(ctx).Model(&specialistModel{})

	if filter.Search != "" {
		escaped := escapeLike(filter.Search)
		search := "%" + escaped + "%"
		query = query.Where("name LIKE ?", search)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", string(filter.Status))
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

	var models []specialistModel
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}

	specialists := make([]domain.Specialist, len(models))
	for i := range models {
		specialists[i] = *toDomain(&models[i])
	}

	return &domain.SpecialistList{
		Specialists: specialists,
		Total:       total,
		Page:        page,
		Limit:       limit,
	}, nil
}
