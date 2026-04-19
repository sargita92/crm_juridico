package infrastructure

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

type tenantModel struct {
	ID                 string     `gorm:"primaryKey;column:id;type:char(36)"`
	Name               string     `gorm:"column:name;type:varchar(255);not null"`
	Type               string     `gorm:"column:type;type:enum('PF','PJ');not null"`
	Document           string     `gorm:"column:document;type:varchar(20);not null;uniqueIndex:idx_tenants_document"`
	Status             string     `gorm:"column:status;type:enum('active','blocked','inactive');not null;default:active"`
	BlockReason        string     `gorm:"column:block_reason;type:varchar(500)"`
	Plano              string     `gorm:"column:plano;type:varchar(20);not null;default:mensal"`
	ValorCobrancaCents *int64     `gorm:"column:valor_cobranca_cents"`
	DiaVencimento      *uint8     `gorm:"column:dia_vencimento"`
	DataInicioCobranca *time.Time `gorm:"column:data_inicio_cobranca"`
	ExibirPagamentos   bool       `gorm:"column:exibir_pagamentos;not null"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (tenantModel) TableName() string { return "tenants" }

func toModel(t *domain.Tenant) *tenantModel {
	return &tenantModel{
		ID:                 t.ID,
		Name:               t.Name,
		Type:               string(t.Type),
		Document:           t.Document,
		Status:             string(t.Status),
		BlockReason:        t.BlockReason,
		Plano:              t.Plano,
		ValorCobrancaCents: t.ValorCobrancaCents,
		DiaVencimento:      t.DiaVencimento,
		DataInicioCobranca: normalizeDateToLocal(t.DataInicioCobranca),
		ExibirPagamentos:   t.ExibirPagamentos,
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
	}
}

// normalizeDateToLocal rebuilds d at local midnight preserving its Y/M/D so
// the MySQL DATE column roundtrip (driver applies session TZ) does not shift
// the logical date. The input is expected to be at midnight-in-some-TZ from
// a form date parse or a UTC construction.
func normalizeDateToLocal(d *time.Time) *time.Time {
	if d == nil {
		return nil
	}
	out := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
	return &out
}

func toDomain(m *tenantModel) *domain.Tenant {
	return &domain.Tenant{
		ID:                 m.ID,
		Name:               m.Name,
		Type:               domain.TenantType(m.Type),
		Document:           m.Document,
		Status:             domain.TenantStatus(m.Status),
		BlockReason:        m.BlockReason,
		Plano:              m.Plano,
		ValorCobrancaCents: m.ValorCobrancaCents,
		DiaVencimento:      m.DiaVencimento,
		DataInicioCobranca: m.DataInicioCobranca,
		ExibirPagamentos:   m.ExibirPagamentos,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

type GormTenantRepository struct {
	db *gorm.DB
}

func NewGormTenantRepository(db *gorm.DB) *GormTenantRepository {
	return &GormTenantRepository{db: db}
}

func (r *GormTenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	model := toModel(tenant)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.ErrTenantDocumentExists
		}
		return err
	}
	return nil
}

func (r *GormTenantRepository) FindByID(ctx context.Context, id string) (*domain.Tenant, error) {
	var model tenantModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, err
	}
	return toDomain(&model), nil
}

func (r *GormTenantRepository) FindByIDs(ctx context.Context, ids []string) ([]domain.Tenant, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var models []tenantModel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&models).Error; err != nil {
		return nil, err
	}
	tenants := make([]domain.Tenant, len(models))
	for i := range models {
		tenants[i] = *toDomain(&models[i])
	}
	return tenants, nil
}

func (r *GormTenantRepository) FindAll(ctx context.Context) ([]domain.Tenant, error) {
	var models []tenantModel
	if err := r.db.WithContext(ctx).Where("status = ?", string(domain.TenantStatusActive)).Find(&models).Error; err != nil {
		return nil, err
	}
	tenants := make([]domain.Tenant, len(models))
	for i := range models {
		tenants[i] = *toDomain(&models[i])
	}
	return tenants, nil
}

func (r *GormTenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	model := toModel(tenant)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(result.Error, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.ErrTenantDocumentExists
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrTenantNotFound
	}
	return nil
}

func (r *GormTenantRepository) FindWithFilter(ctx context.Context, filter domain.TenantFilter) (*domain.TenantList, error) {
	query := r.db.WithContext(ctx).Model(&tenantModel{})

	if filter.Search != "" {
		escaped := escapeLike(filter.Search)
		search := "%" + escaped + "%"
		query = query.Where("name LIKE ? OR document LIKE ?", search, search)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", string(filter.Status))
	}
	if filter.Type != "" {
		query = query.Where("type = ?", string(filter.Type))
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

	offset := (page - 1) * limit

	var models []tenantModel
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}

	tenants := make([]domain.Tenant, len(models))
	for i := range models {
		tenants[i] = *toDomain(&models[i])
	}

	return &domain.TenantList{
		Tenants: tenants,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}, nil
}

func (r *GormTenantRepository) FindByDocument(ctx context.Context, document string) (*domain.Tenant, error) {
	var model tenantModel
	if err := r.db.WithContext(ctx).Where("document = ?", document).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, err
	}
	return toDomain(&model), nil
}
