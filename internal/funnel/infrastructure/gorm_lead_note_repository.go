package infrastructure

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"gorm.io/gorm"
)

type GormLeadNoteRepository struct {
	db *gorm.DB
}

func NewGormLeadNoteRepository(db *gorm.DB) *GormLeadNoteRepository {
	return &GormLeadNoteRepository{db: db}
}

func (r *GormLeadNoteRepository) Create(ctx context.Context, note *domain.LeadNote) error {
	model := noteToModel(note)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormLeadNoteRepository) FindByLeadID(ctx context.Context, leadID string) ([]domain.LeadNote, error) {
	var models []leadNoteModel
	err := r.db.WithContext(ctx).
		Where("lead_id = ?", leadID).
		Order("created_at DESC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	notes := make([]domain.LeadNote, len(models))
	for i, m := range models {
		notes[i] = *noteToDomain(&m)
	}
	return notes, nil
}
