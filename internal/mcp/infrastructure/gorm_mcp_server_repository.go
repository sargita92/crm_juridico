package infrastructure

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

type mcpServerModel struct {
	ID        string `gorm:"primaryKey;column:id;type:char(36)"`
	Name      string `gorm:"column:name;type:varchar(255);not null"`
	URL       string `gorm:"column:url;type:varchar(500);not null"`
	Config    string `gorm:"column:config;type:text"`
	Headers   string `gorm:"column:headers;type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (mcpServerModel) TableName() string { return "mcp_servers" }

func mcpToModel(m *domain.McpServer) *mcpServerModel {
	return &mcpServerModel{
		ID: m.ID, Name: m.Name, URL: m.URL,
		Config: m.Config, Headers: m.Headers,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func mcpToDomain(m *mcpServerModel) *domain.McpServer {
	return &domain.McpServer{
		ID: m.ID, Name: m.Name, URL: m.URL,
		Config: m.Config, Headers: m.Headers,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

type GormMcpServerRepository struct {
	db *gorm.DB
}

func NewGormMcpServerRepository(db *gorm.DB) *GormMcpServerRepository {
	return &GormMcpServerRepository{db: db}
}

func (r *GormMcpServerRepository) Create(ctx context.Context, mcp *domain.McpServer) error {
	return r.db.WithContext(ctx).Create(mcpToModel(mcp)).Error
}

func (r *GormMcpServerRepository) FindByID(ctx context.Context, id string) (*domain.McpServer, error) {
	var model mcpServerModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrMcpNotFound
		}
		return nil, err
	}
	return mcpToDomain(&model), nil
}

func (r *GormMcpServerRepository) Update(ctx context.Context, mcp *domain.McpServer) error {
	result := r.db.WithContext(ctx).Save(mcpToModel(mcp))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrMcpNotFound
	}
	return nil
}

func (r *GormMcpServerRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&mcpServerModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrMcpNotFound
	}
	return nil
}

func (r *GormMcpServerRepository) FindWithFilter(ctx context.Context, filter domain.McpFilter) (*domain.McpList, error) {
	query := r.db.WithContext(ctx).Model(&mcpServerModel{})

	if filter.Search != "" {
		escaped := escapeLike(filter.Search)
		query = query.Where("name LIKE ?", "%"+escaped+"%")
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

	var models []mcpServerModel
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}

	servers := make([]domain.McpServer, len(models))
	for i := range models {
		servers[i] = *mcpToDomain(&models[i])
	}

	return &domain.McpList{McpServers: servers, Total: total, Page: page, Limit: limit}, nil
}

func (r *GormMcpServerRepository) FindByIDs(ctx context.Context, ids []string) ([]domain.McpServer, error) {
	if len(ids) == 0 {
		return []domain.McpServer{}, nil
	}
	var models []mcpServerModel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&models).Error; err != nil {
		return nil, err
	}
	servers := make([]domain.McpServer, len(models))
	for i := range models {
		servers[i] = *mcpToDomain(&models[i])
	}
	return servers, nil
}
