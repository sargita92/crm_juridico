package infrastructure

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

type GormFileRepository struct {
	db *gorm.DB
}

func NewGormFileRepository(db *gorm.DB) *GormFileRepository {
	return &GormFileRepository{db: db}
}

func (r *GormFileRepository) Create(ctx context.Context, f *domain.File) error {
	return r.db.WithContext(ctx).Create(toModel(f)).Error
}

func (r *GormFileRepository) FindByID(ctx context.Context, tenantID, id string) (*domain.File, error) {
	var m fileModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFileNotFound
		}
		return nil, err
	}
	return toDomain(&m), nil
}

func (r *GormFileRepository) List(ctx context.Context, q domain.ListQuery) (*domain.ListResult, error) {
	if q.TenantID == "" {
		return nil, domain.ErrTenantIDRequired
	}
	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = domain.DefaultPageSize
	}
	if pageSize > domain.MaxPageSize {
		pageSize = domain.MaxPageSize
	}

	base := r.db.WithContext(ctx).Model(&fileModel{}).Where("files.tenant_id = ?", q.TenantID)
	if q.LeadID != nil {
		base = base.Where("files.lead_id = ?", *q.LeadID)
	}
	if q.MediaType != nil {
		base = base.Where("files.media_type = ?", string(*q.MediaType))
	}
	if s := strings.TrimSpace(q.Search); s != "" {
		base = base.Where("files.name LIKE ? ESCAPE '\\\\'", "%"+escapeLike(s)+"%")
	}
	if q.From != nil {
		base = base.Where("files.created_at >= ?", *q.From)
	}
	if q.To != nil {
		base = base.Where("files.created_at <= ?", *q.To)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}

	type listRow struct {
		ID             string    `gorm:"column:id"`
		TenantID       string    `gorm:"column:tenant_id"`
		LeadID         *string   `gorm:"column:lead_id"`
		ConversationID string    `gorm:"column:conversation_id"`
		ContactID      string    `gorm:"column:contact_id"`
		MessageID      *string   `gorm:"column:message_id"`
		Name           string    `gorm:"column:name"`
		MediaType      string    `gorm:"column:media_type"`
		MimeType       string    `gorm:"column:mime_type"`
		SizeBytes      int64     `gorm:"column:size_bytes"`
		StorageKey     string    `gorm:"column:storage_key"`
		Direction      string    `gorm:"column:direction"`
		CreatedAt      time.Time `gorm:"column:created_at"`
		UpdatedAt      time.Time `gorm:"column:updated_at"`
		ContactName    string    `gorm:"column:contact_name"`
	}
	var rows []listRow
	err := base.Session(&gorm.Session{}).
		Select(`files.id, files.tenant_id, files.lead_id, files.conversation_id,
			files.contact_id, files.message_id, files.name, files.media_type,
			files.mime_type, files.size_bytes, files.storage_key, files.direction,
			files.created_at, files.updated_at,
			contacts.name AS contact_name`).
		Joins("LEFT JOIN contacts ON contacts.id = files.contact_id").
		Order("files.created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	items := make([]domain.FileWithContext, len(rows))
	for i, row := range rows {
		items[i] = domain.FileWithContext{
			File: domain.File{
				ID:             row.ID,
				TenantID:       row.TenantID,
				LeadID:         row.LeadID,
				ConversationID: row.ConversationID,
				ContactID:      row.ContactID,
				MessageID:      row.MessageID,
				Name:           row.Name,
				MediaType:      domain.MediaType(row.MediaType),
				MimeType:       row.MimeType,
				SizeBytes:      row.SizeBytes,
				StorageKey:     row.StorageKey,
				Direction:      domain.Direction(row.Direction),
				CreatedAt:      row.CreatedAt,
				UpdatedAt:      row.UpdatedAt,
			},
			ContactName: row.ContactName,
		}
	}

	return &domain.ListResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (r *GormFileRepository) CountByLead(ctx context.Context, tenantID, leadID string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&fileModel{}).
		Where("tenant_id = ? AND lead_id = ?", tenantID, leadID).
		Count(&total).Error
	return total, err
}

func (r *GormFileRepository) ListRecentByLead(ctx context.Context, tenantID, leadID string, limit int) ([]domain.File, error) {
	if limit <= 0 {
		limit = 6
	}
	var models []fileModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND lead_id = ?", tenantID, leadID).
		Order("created_at DESC").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.File, len(models))
	for i := range models {
		out[i] = *toDomain(&models[i])
	}
	return out, nil
}

// escapeLike escapes % and _ in user-supplied search so they are matched
// literally, preventing search-pattern injection that would otherwise let a
// user match everything with `%` or `_`.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}
