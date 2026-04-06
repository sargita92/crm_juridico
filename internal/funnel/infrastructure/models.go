package infrastructure

import (
	"time"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type funnelModel struct {
	ID          string `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID    string `gorm:"column:tenant_id;type:char(36);not null"`
	Name        string `gorm:"column:name;type:varchar(255);not null"`
	Description string `gorm:"column:description;type:text"`
	Active      bool   `gorm:"column:active;type:tinyint(1);not null;default:1"`
	IsDefault   bool   `gorm:"column:is_default;type:tinyint(1);not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (funnelModel) TableName() string { return "funnels" }

type columnModel struct {
	ID         string `gorm:"primaryKey;column:id;type:char(36)"`
	FunnelID   string `gorm:"column:funnel_id;type:char(36);not null"`
	Name       string `gorm:"column:name;type:varchar(100);not null"`
	OrderIndex int    `gorm:"column:order_index;type:int;not null;default:0"`
	Type       string `gorm:"column:type;type:varchar(20);not null"`
	Color      string `gorm:"column:color;type:varchar(7);not null;default:'#3b82f6'"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (columnModel) TableName() string { return "funnel_columns" }

type leadModel struct {
	ID              string `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID        string `gorm:"column:tenant_id;type:char(36);not null"`
	FunnelID        string `gorm:"column:funnel_id;type:char(36);not null"`
	ColumnID        string `gorm:"column:column_id;type:char(36);not null"`
	ContactID       string `gorm:"column:contact_id;type:char(36);not null"`
	ConversationID  string `gorm:"column:conversation_id;type:char(36);not null"`
	ProductID       string `gorm:"column:product_id;type:char(36)"`
	Score           int    `gorm:"column:score;type:int;not null;default:0"`
	Status          string `gorm:"column:status;type:varchar(20);not null;default:'open'"`
	ColumnEnteredAt time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (leadModel) TableName() string { return "leads" }

type leadMovementModel struct {
	ID           string    `gorm:"primaryKey;column:id;type:char(36)"`
	LeadID       string    `gorm:"column:lead_id;type:char(36);not null"`
	FromColumnID string    `gorm:"column:from_column_id;type:char(36)"`
	ToColumnID   string    `gorm:"column:to_column_id;type:char(36);not null"`
	MovedAt      time.Time `gorm:"column:moved_at;not null"`
}

func (leadMovementModel) TableName() string { return "lead_movements" }

// --- Funnel mappers ---

func funnelToModel(f *domain.Funnel) *funnelModel {
	return &funnelModel{
		ID:          f.ID,
		TenantID:    f.TenantID,
		Name:        f.Name,
		Description: f.Description,
		Active:      f.Active,
		IsDefault:   f.IsDefault,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}
}

func funnelToDomain(m *funnelModel) *domain.Funnel {
	return &domain.Funnel{
		ID:          m.ID,
		TenantID:    m.TenantID,
		Name:        m.Name,
		Description: m.Description,
		Active:      m.Active,
		IsDefault:   m.IsDefault,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// --- Column mappers ---

func columnToModel(c *domain.Column) *columnModel {
	return &columnModel{
		ID:         c.ID,
		FunnelID:   c.FunnelID,
		Name:       c.Name,
		OrderIndex: c.OrderIndex,
		Type:       string(c.Type),
		Color:      c.Color,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

func columnToDomain(m *columnModel) *domain.Column {
	return &domain.Column{
		ID:         m.ID,
		FunnelID:   m.FunnelID,
		Name:       m.Name,
		OrderIndex: m.OrderIndex,
		Type:       domain.ColumnType(m.Type),
		Color:      m.Color,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

// --- Lead mappers ---

func leadToModel(l *domain.Lead) *leadModel {
	return &leadModel{
		ID:              l.ID,
		TenantID:        l.TenantID,
		FunnelID:        l.FunnelID,
		ColumnID:        l.ColumnID,
		ContactID:       l.ContactID,
		ConversationID:  l.ConversationID,
		ProductID:       l.ProductID,
		Score:           l.Score,
		Status:          string(l.Status),
		ColumnEnteredAt: l.ColumnEnteredAt,
		CreatedAt:       l.CreatedAt,
		UpdatedAt:       l.UpdatedAt,
	}
}

func leadToDomain(m *leadModel) *domain.Lead {
	return &domain.Lead{
		ID:              m.ID,
		TenantID:        m.TenantID,
		FunnelID:        m.FunnelID,
		ColumnID:        m.ColumnID,
		ContactID:       m.ContactID,
		ConversationID:  m.ConversationID,
		ProductID:       m.ProductID,
		Score:           m.Score,
		Status:          domain.LeadStatus(m.Status),
		ColumnEnteredAt: m.ColumnEnteredAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

// --- LeadMovement mappers ---

func movementToModel(m *domain.LeadMovement) *leadMovementModel {
	return &leadMovementModel{
		ID:           m.ID,
		LeadID:       m.LeadID,
		FromColumnID: m.FromColumnID,
		ToColumnID:   m.ToColumnID,
		MovedAt:      m.MovedAt,
	}
}

func movementToDomain(m *leadMovementModel) *domain.LeadMovement {
	return &domain.LeadMovement{
		ID:           m.ID,
		LeadID:       m.LeadID,
		FromColumnID: m.FromColumnID,
		ToColumnID:   m.ToColumnID,
		MovedAt:      m.MovedAt,
	}
}

// --- LeadNote model ---

type leadNoteModel struct {
	ID        string    `gorm:"primaryKey;column:id;type:char(36)"`
	LeadID    string    `gorm:"column:lead_id;type:char(36);not null"`
	TenantID  string    `gorm:"column:tenant_id;type:char(36);not null"`
	Content   string    `gorm:"column:content;type:text;not null"`
	CreatedBy string    `gorm:"column:created_by;type:char(36);not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

func (leadNoteModel) TableName() string { return "lead_notes" }

func noteToModel(n *domain.LeadNote) *leadNoteModel {
	return &leadNoteModel{
		ID:        n.ID,
		LeadID:    n.LeadID,
		TenantID:  n.TenantID,
		Content:   n.Content,
		CreatedBy: n.CreatedBy,
		CreatedAt: n.CreatedAt,
	}
}

func noteToDomain(m *leadNoteModel) *domain.LeadNote {
	return &domain.LeadNote{
		ID:        m.ID,
		LeadID:    m.LeadID,
		TenantID:  m.TenantID,
		Content:   m.Content,
		CreatedBy: m.CreatedBy,
		CreatedAt: m.CreatedAt,
	}
}
