package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/sasrgita/crm-juridico/internal/automation/domain"
)

// GormRateLimitRepository is the GORM-backed implementation of domain.RateLimitRepository.
type GormRateLimitRepository struct {
	db *gorm.DB
}

func NewGormRateLimitRepository(db *gorm.DB) *GormRateLimitRepository {
	return &GormRateLimitRepository{db: db}
}

// FindOrCreate returns today's counter for the given tenant/specialist pair.
// If no record exists it is created atomically via FirstOrCreate.
func (r *GormRateLimitRepository) FindOrCreate(ctx context.Context, tenantID, specialistID string) (*domain.RateLimitCounter, error) {
	today := startOfDay(time.Now())

	attrs := rateLimitCounterModel{
		ID:           uuid.NewString(),
		TenantID:     tenantID,
		PeriodStart:  today,
		MessageCount: 0,
		CreatedAt:    time.Now(),
	}
	if specialistID != "" {
		v := specialistID
		attrs.SpecialistID = &v
	}

	// Conditions used to look up the existing row.
	conds := rateLimitCounterModel{
		TenantID:    tenantID,
		PeriodStart: today,
	}
	if specialistID != "" {
		v := specialistID
		conds.SpecialistID = &v
	}

	var m rateLimitCounterModel
	result := r.db.WithContext(ctx).
		Where(conds).
		Attrs(attrs).
		FirstOrCreate(&m)
	if result.Error != nil {
		return nil, result.Error
	}
	return rateLimitToDomain(&m), nil
}

// Increment atomically increments message_count for the given counter ID.
func (r *GormRateLimitRepository) Increment(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Exec("UPDATE rate_limit_counters SET message_count = message_count + 1 WHERE id = ?", id).
		Error
}

// startOfDay returns midnight UTC for the given time.
func startOfDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
