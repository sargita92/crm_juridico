package infrastructure

import (
	"encoding/json"
	"time"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// --- PermissionGroup model ---

type permissionGroupModel struct {
	ID          string `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID    string `gorm:"column:tenant_id;type:char(36);not null"`
	Name        string `gorm:"column:name;type:varchar(100);not null"`
	Description string `gorm:"column:description;type:varchar(500);not null;default:''"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (permissionGroupModel) TableName() string { return "permission_groups" }

func permissionGroupToModel(g *domain.PermissionGroup) *permissionGroupModel {
	return &permissionGroupModel{
		ID:          g.ID,
		TenantID:    g.TenantID,
		Name:        g.Name,
		Description: g.Description,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}

func permissionGroupToDomain(m *permissionGroupModel) *domain.PermissionGroup {
	return &domain.PermissionGroup{
		ID:          m.ID,
		TenantID:    m.TenantID,
		Name:        m.Name,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// --- UserGroup model ---

type userGroupModel struct {
	ID        string `gorm:"primaryKey;column:id;type:char(36)"`
	UserID    string `gorm:"column:user_id;type:char(36);not null"`
	GroupID   string `gorm:"column:group_id;type:char(36);not null"`
	TenantID  string `gorm:"column:tenant_id;type:char(36);not null"`
	CreatedAt time.Time
}

func (userGroupModel) TableName() string { return "user_groups" }

func userGroupToModel(ug *domain.UserGroup) *userGroupModel {
	return &userGroupModel{
		ID:        ug.ID,
		UserID:    ug.UserID,
		GroupID:   ug.GroupID,
		TenantID:  ug.TenantID,
		CreatedAt: ug.CreatedAt,
	}
}

func userGroupToDomain(m *userGroupModel) *domain.UserGroup {
	return &domain.UserGroup{
		ID:        m.ID,
		UserID:    m.UserID,
		GroupID:   m.GroupID,
		TenantID:  m.TenantID,
		CreatedAt: m.CreatedAt,
	}
}

// --- Permission model ---

type permissionModel struct {
	ID        string  `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID  string  `gorm:"column:tenant_id;type:char(36);not null"`
	GroupID   *string `gorm:"column:group_id;type:char(36)"`
	UserID    *string `gorm:"column:user_id;type:char(36)"`
	Resource  string  `gorm:"column:resource;type:varchar(50);not null"`
	Action    string  `gorm:"column:action;type:varchar(50);not null"`
	CreatedAt time.Time
}

func (permissionModel) TableName() string { return "permissions" }

func permissionToModel(p *domain.Permission) *permissionModel {
	m := &permissionModel{
		ID:        p.ID,
		TenantID:  p.TenantID,
		Resource:  p.Resource,
		Action:    p.Action,
		CreatedAt: p.CreatedAt,
	}
	if p.GroupID != "" {
		v := p.GroupID
		m.GroupID = &v
	}
	if p.UserID != "" {
		v := p.UserID
		m.UserID = &v
	}
	return m
}

func permissionToDomain(m *permissionModel) *domain.Permission {
	p := &domain.Permission{
		ID:        m.ID,
		TenantID:  m.TenantID,
		Resource:  m.Resource,
		Action:    m.Action,
		CreatedAt: m.CreatedAt,
	}
	if m.GroupID != nil {
		p.GroupID = *m.GroupID
	}
	if m.UserID != nil {
		p.UserID = *m.UserID
	}
	return p
}

// --- ViewProfile model ---

type viewProfileModel struct {
	ID             string `gorm:"primaryKey;column:id;type:char(36)"`
	GroupID        string `gorm:"column:group_id;type:char(36);not null"`
	FunnelID       string `gorm:"column:funnel_id;type:char(36);not null"`
	VisibleColumns string `gorm:"column:visible_columns;type:json;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (viewProfileModel) TableName() string { return "view_profiles" }

func viewProfileToModel(vp *domain.ViewProfile) *viewProfileModel {
	cols, _ := json.Marshal(vp.VisibleColumns)
	return &viewProfileModel{
		ID:             vp.ID,
		GroupID:        vp.GroupID,
		FunnelID:       vp.FunnelID,
		VisibleColumns: string(cols),
		CreatedAt:      vp.CreatedAt,
		UpdatedAt:      vp.UpdatedAt,
	}
}

func viewProfileToDomain(m *viewProfileModel) *domain.ViewProfile {
	var cols []string
	if m.VisibleColumns != "" {
		_ = json.Unmarshal([]byte(m.VisibleColumns), &cols)
	}
	if cols == nil {
		cols = []string{}
	}
	return &domain.ViewProfile{
		ID:             m.ID,
		GroupID:        m.GroupID,
		FunnelID:       m.FunnelID,
		VisibleColumns: cols,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// --- GroupFunnel model ---

type groupFunnelModel struct {
	ID        string `gorm:"primaryKey;column:id;type:char(36)"`
	GroupID   string `gorm:"column:group_id;type:char(36);not null"`
	FunnelID  string `gorm:"column:funnel_id;type:char(36);not null"`
	ColumnIDs string `gorm:"column:column_ids;type:json;not null"`
	CreatedAt time.Time
}

func (groupFunnelModel) TableName() string { return "group_funnels" }

func groupFunnelToModel(gf *domain.GroupFunnel) *groupFunnelModel {
	ids, _ := json.Marshal(gf.ColumnIDs)
	return &groupFunnelModel{
		ID:        gf.ID,
		GroupID:   gf.GroupID,
		FunnelID:  gf.FunnelID,
		ColumnIDs: string(ids),
		CreatedAt: gf.CreatedAt,
	}
}

func groupFunnelToDomain(m *groupFunnelModel) *domain.GroupFunnel {
	var ids []string
	if m.ColumnIDs != "" {
		_ = json.Unmarshal([]byte(m.ColumnIDs), &ids)
	}
	if ids == nil {
		ids = []string{}
	}
	return &domain.GroupFunnel{
		ID:        m.ID,
		GroupID:   m.GroupID,
		FunnelID:  m.FunnelID,
		ColumnIDs: ids,
		CreatedAt: m.CreatedAt,
	}
}
