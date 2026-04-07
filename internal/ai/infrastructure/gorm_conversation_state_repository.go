package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

type conversationStateModel struct {
	ID               string     `gorm:"primaryKey;column:id;type:char(36)"`
	ConversationID   string     `gorm:"column:conversation_id;type:char(36);uniqueIndex;not null"`
	SpecialistID     string     `gorm:"column:specialist_id;type:char(36);not null"`
	CurrentStepIndex int        `gorm:"column:current_step_index;not null;default:0"`
	CollectedData    string     `gorm:"column:collected_data;type:text;not null;default:'{}'"`
	AccumulatedScore int        `gorm:"column:accumulated_score;not null;default:0"`
	HandoffActive    bool       `gorm:"column:handoff_active;not null;default:false"`
	HandoffAt        *time.Time `gorm:"column:handoff_at"`
	ResumedAt        *time.Time `gorm:"column:resumed_at"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (conversationStateModel) TableName() string { return "conversation_states" }

func conversationStateToModel(s *domain.ConversationState) (*conversationStateModel, error) {
	data, err := json.Marshal(s.CollectedData)
	if err != nil {
		return nil, err
	}
	return &conversationStateModel{
		ID:               s.ID,
		ConversationID:   s.ConversationID,
		SpecialistID:     s.SpecialistID,
		CurrentStepIndex: s.CurrentStepIndex,
		CollectedData:    string(data),
		AccumulatedScore: s.AccumulatedScore,
		HandoffActive:    s.HandoffActive,
		HandoffAt:        s.HandoffAt,
		ResumedAt:        s.ResumedAt,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}, nil
}

func conversationStateToDomain(m *conversationStateModel) (*domain.ConversationState, error) {
	var collectedData map[string]string
	if m.CollectedData == "" || m.CollectedData == "null" {
		collectedData = make(map[string]string)
	} else {
		if err := json.Unmarshal([]byte(m.CollectedData), &collectedData); err != nil {
			return nil, err
		}
	}
	return &domain.ConversationState{
		ID:               m.ID,
		ConversationID:   m.ConversationID,
		SpecialistID:     m.SpecialistID,
		CurrentStepIndex: m.CurrentStepIndex,
		CollectedData:    collectedData,
		AccumulatedScore: m.AccumulatedScore,
		HandoffActive:    m.HandoffActive,
		HandoffAt:        m.HandoffAt,
		ResumedAt:        m.ResumedAt,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}, nil
}

// GormConversationStateRepository implements domain.ConversationStateRepository using GORM.
type GormConversationStateRepository struct {
	db *gorm.DB
}

// NewGormConversationStateRepository creates a new GormConversationStateRepository.
func NewGormConversationStateRepository(db *gorm.DB) *GormConversationStateRepository {
	return &GormConversationStateRepository{db: db}
}

// Create inserts a new ConversationState record.
func (r *GormConversationStateRepository) Create(ctx context.Context, state *domain.ConversationState) error {
	model, err := conversationStateToModel(state)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// FindByConversationID retrieves a ConversationState by conversation ID.
func (r *GormConversationStateRepository) FindByConversationID(ctx context.Context, conversationID string) (*domain.ConversationState, error) {
	var model conversationStateModel
	if err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrConversationStateNotFound
		}
		return nil, err
	}
	return conversationStateToDomain(&model)
}

// Update saves changes to an existing ConversationState record.
func (r *GormConversationStateRepository) Update(ctx context.Context, state *domain.ConversationState) error {
	model, err := conversationStateToModel(state)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(model).Error
}
