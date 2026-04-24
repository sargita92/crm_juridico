package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/audit/domain"
)

// GormAuditLogRepository implementa domain.Repository sobre Gorm/MySQL.
type GormAuditLogRepository struct {
	db *gorm.DB
}

// NewGormAuditLogRepository constroi o repositorio. Retorna o tipo concreto
// para que o composition root possa atribuir a `domain.Repository` (interface).
func NewGormAuditLogRepository(db *gorm.DB) *GormAuditLogRepository {
	return &GormAuditLogRepository{db: db}
}

// Compile-time check de aderencia ao contrato.
var _ domain.Repository = (*GormAuditLogRepository)(nil)

// Create persiste o audit log. Erros de banco sao propagados para o caller —
// o publisher (Step 3) e responsavel por nao quebrar o caso de uso original.
func (r *GormAuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	model, err := toModel(log)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(model).Error
}

// FindByID retorna o log pelo ID; traduz `gorm.ErrRecordNotFound` para
// `domain.ErrAuditLogNotFound`.
func (r *GormAuditLogRepository) FindByID(ctx context.Context, id string) (*domain.AuditLog, error) {
	var model auditLogModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrAuditLogNotFound
		}
		return nil, err
	}
	return toDomain(&model)
}

// List devolve a pagina filtrada e o total de registros que casam com o
// filtro (sem aplicar paginacao na contagem).
//
// Comportamento:
//   - Normaliza o filtro (defaults de pagina/limit + validacao From <= To).
//   - Aplica WHERE somente para campos non-nil.
//   - ORDER BY created_at DESC, id DESC para desempate estavel entre logs
//     no mesmo milissegundo (importante na paginacao de rajadas).
//   - Conta o total respeitando o WHERE em uma query separada.
func (r *GormAuditLogRepository) List(ctx context.Context, filter domain.Filter) ([]*domain.AuditLog, int64, error) {
	if err := filter.Normalize(); err != nil {
		return nil, 0, err
	}

	query := r.db.WithContext(ctx).Model(&auditLogModel{})
	query = applyFilter(query, filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []auditLogModel
	if err := query.
		Order("created_at DESC").
		Order("id DESC").
		Limit(filter.PageSize).
		Offset(filter.Offset()).
		Find(&models).Error; err != nil {
		return nil, 0, err
	}

	logs := make([]*domain.AuditLog, 0, len(models))
	for i := range models {
		log, err := toDomain(&models[i])
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, log)
	}
	return logs, total, nil
}

// applyFilter monta as clausulas WHERE a partir do Filter normalizado.
// Retorna a query encadeada — caller decide se conta ou pagina.
func applyFilter(query *gorm.DB, filter domain.Filter) *gorm.DB {
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Action != nil {
		query = query.Where("action = ?", string(*filter.Action))
	}
	if filter.From != nil {
		query = query.Where("created_at >= ?", filter.From.UTC())
	}
	if filter.To != nil {
		query = query.Where("created_at <= ?", filter.To.UTC())
	}
	return query
}
