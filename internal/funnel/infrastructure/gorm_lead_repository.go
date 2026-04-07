package infrastructure

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

type GormLeadRepository struct {
	db *gorm.DB
}

func NewGormLeadRepository(db *gorm.DB) *GormLeadRepository {
	return &GormLeadRepository{db: db}
}

func (r *GormLeadRepository) Create(ctx context.Context, lead *domain.Lead) error {
	model := leadToModel(lead)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormLeadRepository) FindByID(ctx context.Context, id string) (*domain.Lead, error) {
	var model leadModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrLeadNotFound
		}
		return nil, err
	}
	return leadToDomain(&model), nil
}

func (r *GormLeadRepository) Update(ctx context.Context, lead *domain.Lead) error {
	model := leadToModel(lead)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrLeadNotFound
	}
	return nil
}

func (r *GormLeadRepository) FindByConversationID(ctx context.Context, conversationID string) (*domain.Lead, error) {
	var model leadModel
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrLeadNotFound
		}
		return nil, err
	}
	return leadToDomain(&model), nil
}

func (r *GormLeadRepository) FindByContactAndTenant(ctx context.Context, tenantID, contactID string) (*domain.Lead, error) {
	var model leadModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND contact_id = ?", tenantID, contactID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrLeadNotFound
		}
		return nil, err
	}
	return leadToDomain(&model), nil
}

func (r *GormLeadRepository) FindByFunnelID(ctx context.Context, funnelID string, filter domain.LeadFilter) (*domain.LeadList, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}

	// Count query
	countQuery := r.db.WithContext(ctx).
		Table("leads").
		Joins("JOIN contacts ON contacts.id = leads.contact_id").
		Where("leads.funnel_id = ?", funnelID)

	if filter.Search != "" {
		escaped := escapeLike(filter.Search)
		search := "%" + escaped + "%"
		countQuery = countQuery.Where("contacts.name LIKE ? OR contacts.phone LIKE ?", search, search)
	}
	if filter.ColumnID != "" {
		countQuery = countQuery.Where("leads.column_id = ?", filter.ColumnID)
	}
	if filter.ProductID != "" {
		countQuery = countQuery.Where("leads.product_id = ?", filter.ProductID)
	}

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	// Data query
	type leadRow struct {
		ID              string `gorm:"column:id"`
		TenantID        string `gorm:"column:tenant_id"`
		FunnelID        string `gorm:"column:funnel_id"`
		ColumnID        string `gorm:"column:column_id"`
		ContactID       string `gorm:"column:contact_id"`
		ConversationID  string `gorm:"column:conversation_id"`
		ProductID       string `gorm:"column:product_id"`
		Score           int    `gorm:"column:score"`
		Status          string `gorm:"column:status"`
		ColumnEnteredAt string `gorm:"column:column_entered_at"`
		CreatedAt       string `gorm:"column:lead_created_at"`
		UpdatedAt       string `gorm:"column:lead_updated_at"`
		ContactName     string `gorm:"column:contact_name"`
		ContactPhone    string `gorm:"column:contact_phone"`
		ColumnName      string `gorm:"column:column_name"`
		ColumnColor     string `gorm:"column:column_color"`
		ColumnOrder     int    `gorm:"column:column_order"`
	}

	dataQuery := r.db.WithContext(ctx).
		Table("leads").
		Select(`leads.id, leads.tenant_id, leads.funnel_id, leads.column_id,
			leads.contact_id, leads.conversation_id, leads.product_id, leads.score, leads.status,
			leads.column_entered_at, leads.created_at as lead_created_at, leads.updated_at as lead_updated_at,
			contacts.name as contact_name, contacts.phone as contact_phone,
			funnel_columns.name as column_name, funnel_columns.color as column_color,
			funnel_columns.order_index as column_order`).
		Joins("JOIN contacts ON contacts.id = leads.contact_id").
		Joins("JOIN funnel_columns ON funnel_columns.id = leads.column_id").
		Where("leads.funnel_id = ?", funnelID)

	if filter.Search != "" {
		escaped := escapeLike(filter.Search)
		search := "%" + escaped + "%"
		dataQuery = dataQuery.Where("contacts.name LIKE ? OR contacts.phone LIKE ?", search, search)
	}
	if filter.ColumnID != "" {
		dataQuery = dataQuery.Where("leads.column_id = ?", filter.ColumnID)
	}
	if filter.ProductID != "" {
		dataQuery = dataQuery.Where("leads.product_id = ?", filter.ProductID)
	}

	offset := (page - 1) * limit
	var rows []leadRow
	if err := dataQuery.
		Order("funnel_columns.order_index ASC, leads.created_at ASC").
		Offset(offset).Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	leads := make([]domain.LeadWithContact, len(rows))
	for i, row := range rows {
		leads[i] = domain.LeadWithContact{
			Lead: domain.Lead{
				ID:             row.ID,
				TenantID:       row.TenantID,
				FunnelID:       row.FunnelID,
				ColumnID:       row.ColumnID,
				ContactID:      row.ContactID,
				ConversationID: row.ConversationID,
				ProductID:      row.ProductID,
				Score:          row.Score,
				Status:         domain.LeadStatus(row.Status),
			},
			ContactName:  row.ContactName,
			ContactPhone: row.ContactPhone,
			ColumnName:   row.ColumnName,
			ColumnColor:  row.ColumnColor,
		}
	}

	return &domain.LeadList{
		Leads: leads,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *GormLeadRepository) CountByColumnID(ctx context.Context, columnID string) (int, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&leadModel{}).
		Where("column_id = ?", columnID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}
