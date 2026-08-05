package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type guardrailModel struct {
	ID        string `gorm:"primaryKey;column:id;type:char(36)"`
	Name      string `gorm:"column:name;type:varchar(120);not null"`
	Type      string `gorm:"column:type;not null"`
	Rule      string `gorm:"column:rule;type:text;not null"`
	Message   string `gorm:"column:message;type:text"`
	Active    bool   `gorm:"column:active;not null;default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (guardrailModel) TableName() string { return "guardrails" }

// specialistGuardrailModel is the join row linking a guardrail to a specialist.
type specialistGuardrailModel struct {
	SpecialistID string    `gorm:"primaryKey;column:specialist_id;type:char(36)"`
	GuardrailID  string    `gorm:"primaryKey;column:guardrail_id;type:char(36)"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (specialistGuardrailModel) TableName() string { return "specialist_guardrails" }

func guardrailToModel(g *domain.Guardrail) *guardrailModel {
	return &guardrailModel{
		ID: g.ID, Name: g.Name, Type: string(g.Type),
		Rule: g.Rule, Message: g.Message, Active: g.Active,
		CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt,
	}
}

func guardrailToDomain(m *guardrailModel) *domain.Guardrail {
	return &domain.Guardrail{
		ID: m.ID, Name: m.Name, Type: domain.GuardrailType(m.Type),
		Rule: m.Rule, Message: m.Message, Active: m.Active,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

type GormGuardrailRepository struct {
	db *gorm.DB
}

func NewGormGuardrailRepository(db *gorm.DB) *GormGuardrailRepository {
	return &GormGuardrailRepository{db: db}
}

func (r *GormGuardrailRepository) Create(ctx context.Context, guardrail *domain.Guardrail) error {
	return r.db.WithContext(ctx).Create(guardrailToModel(guardrail)).Error
}

func (r *GormGuardrailRepository) FindByID(ctx context.Context, id string) (*domain.Guardrail, error) {
	var model guardrailModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrGuardrailNotFound
		}
		return nil, err
	}
	return guardrailToDomain(&model), nil
}

func (r *GormGuardrailRepository) Update(ctx context.Context, guardrail *domain.Guardrail) error {
	// Scope the update to the mutable columns so a Save with a zero CreatedAt does
	// not clobber the library item's creation stamp.
	result := r.db.WithContext(ctx).Model(&guardrailModel{ID: guardrail.ID}).
		Updates(map[string]any{
			"name":       guardrail.Name,
			"type":       string(guardrail.Type),
			"rule":       guardrail.Rule,
			"message":    guardrail.Message,
			"active":     guardrail.Active,
			"updated_at": guardrail.UpdatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrGuardrailNotFound
	}
	return nil
}

func (r *GormGuardrailRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&guardrailModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrGuardrailNotFound
	}
	return nil
}

func (r *GormGuardrailRepository) FindAll(ctx context.Context) ([]domain.Guardrail, error) {
	var models []guardrailModel
	if err := r.db.WithContext(ctx).Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	guardrails := make([]domain.Guardrail, len(models))
	for i := range models {
		guardrails[i] = *guardrailToDomain(&models[i])
	}
	return guardrails, nil
}

func (r *GormGuardrailRepository) FindBySpecialistID(ctx context.Context, specialistID string) ([]domain.Guardrail, error) {
	var models []guardrailModel
	err := r.db.WithContext(ctx).
		Joins("JOIN specialist_guardrails sg ON sg.guardrail_id = guardrails.id").
		Where("sg.specialist_id = ?", specialistID).
		Order("guardrails.created_at ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	guardrails := make([]domain.Guardrail, len(models))
	for i := range models {
		guardrails[i] = *guardrailToDomain(&models[i])
	}
	return guardrails, nil
}

func (r *GormGuardrailRepository) Attach(ctx context.Context, specialistID, guardrailID string) error {
	link := specialistGuardrailModel{
		SpecialistID: specialistID,
		GuardrailID:  guardrailID,
		CreatedAt:    time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(&link).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.ErrGuardrailAlreadyAttached
		}
		return err
	}
	return nil
}

func (r *GormGuardrailRepository) Detach(ctx context.Context, specialistID, guardrailID string) error {
	result := r.db.WithContext(ctx).
		Where("specialist_id = ? AND guardrail_id = ?", specialistID, guardrailID).
		Delete(&specialistGuardrailModel{})
	return result.Error
}

func (r *GormGuardrailRepository) CountSpecialistsByGuardrailID(ctx context.Context, guardrailID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&specialistGuardrailModel{}).
		Where("guardrail_id = ?", guardrailID).
		Count(&count).Error
	return int(count), err
}
