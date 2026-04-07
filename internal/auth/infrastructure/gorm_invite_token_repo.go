package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

type inviteTokenModel struct {
	ID        string       `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID  string       `gorm:"column:tenant_id;type:char(36);not null"`
	Token     string       `gorm:"column:token;type:varchar(64);not null;uniqueIndex:uk_invite_token"`
	CreatedBy string       `gorm:"column:created_by;type:char(36);not null"`
	GroupIDs  string       `gorm:"column:group_ids;type:json;not null"`
	ExpiresAt time.Time    `gorm:"column:expires_at;not null"`
	UsedAt    sql.NullTime `gorm:"column:used_at"`
	UsedBy    string       `gorm:"column:used_by;type:char(36)"`
	CreatedAt time.Time    `gorm:"column:created_at;autoCreateTime"`
}

func (inviteTokenModel) TableName() string { return "invite_tokens" }

func inviteTokenToModel(t *domain.InviteToken) (*inviteTokenModel, error) {
	groupIDsJSON, err := json.Marshal(t.GroupIDs)
	if err != nil {
		return nil, err
	}

	m := &inviteTokenModel{
		ID:        t.ID,
		TenantID:  t.TenantID,
		Token:     t.Token,
		CreatedBy: t.CreatedBy,
		GroupIDs:  string(groupIDsJSON),
		ExpiresAt: t.ExpiresAt,
		UsedBy:    t.UsedBy,
		CreatedAt: t.CreatedAt,
	}
	if t.UsedAt != nil {
		m.UsedAt = sql.NullTime{Time: *t.UsedAt, Valid: true}
	}
	return m, nil
}

func inviteTokenToDomain(m *inviteTokenModel) (*domain.InviteToken, error) {
	var groupIDs []string
	if err := json.Unmarshal([]byte(m.GroupIDs), &groupIDs); err != nil {
		return nil, err
	}

	t := &domain.InviteToken{
		ID:        m.ID,
		TenantID:  m.TenantID,
		Token:     m.Token,
		CreatedBy: m.CreatedBy,
		GroupIDs:  groupIDs,
		ExpiresAt: m.ExpiresAt,
		UsedBy:    m.UsedBy,
		CreatedAt: m.CreatedAt,
	}
	if m.UsedAt.Valid {
		t.UsedAt = &m.UsedAt.Time
	}
	return t, nil
}

type GormInviteTokenRepository struct {
	db *gorm.DB
}

func NewGormInviteTokenRepository(db *gorm.DB) *GormInviteTokenRepository {
	return &GormInviteTokenRepository{db: db}
}

func (r *GormInviteTokenRepository) Create(ctx context.Context, token *domain.InviteToken) error {
	model, err := inviteTokenToModel(token)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormInviteTokenRepository) FindByToken(ctx context.Context, token string) (*domain.InviteToken, error) {
	var model inviteTokenModel
	err := r.db.WithContext(ctx).
		Where("token = ?", token).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrInviteTokenNotFound
		}
		return nil, err
	}
	return inviteTokenToDomain(&model)
}

func (r *GormInviteTokenRepository) FindByTenantID(ctx context.Context, tenantID string) ([]*domain.InviteToken, error) {
	var models []inviteTokenModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	result := make([]*domain.InviteToken, 0, len(models))
	for i := range models {
		t, err := inviteTokenToDomain(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

func (r *GormInviteTokenRepository) Update(ctx context.Context, token *domain.InviteToken) error {
	model, err := inviteTokenToModel(token)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *GormInviteTokenRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&inviteTokenModel{}).Error
}
