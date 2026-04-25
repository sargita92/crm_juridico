package infrastructure

import (
	"encoding/json"
	"time"

	"github.com/sasrgita/crm-juridico/internal/audit/domain"
)

// auditLogModel e o mapeamento Gorm para a tabela `audit_logs`.
//
// Logs sao imutaveis: nao ha helpers de Update/Delete. Metadata e persistida
// como JSON via marshalling manual (string column) seguindo o padrao do
// repositorio de conversation_states (`internal/ai/infrastructure`).
type auditLogModel struct {
	ID         string    `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID   *string   `gorm:"column:tenant_id;type:char(36)"`
	UserID     *string   `gorm:"column:user_id;type:char(36)"`
	ActorEmail string    `gorm:"column:actor_email;type:varchar(255);not null"`
	Action     string    `gorm:"column:action;type:varchar(64);not null"`
	Entity     string    `gorm:"column:entity;type:varchar(64);not null;default:''"`
	EntityID   *string   `gorm:"column:entity_id;type:char(36)"`
	IP         string    `gorm:"column:ip;type:varchar(45);not null"`
	UserAgent  string    `gorm:"column:user_agent;type:varchar(255);not null;default:''"`
	Metadata   *string   `gorm:"column:metadata;type:json"`
	CreatedAt  time.Time `gorm:"column:created_at;not null"`
}

// TableName retorna o nome da tabela MySQL.
func (auditLogModel) TableName() string { return "audit_logs" }

// toModel converte a entidade de dominio para o modelo Gorm.
//
// Metadata vazia/nil e persistida como NULL (em vez de "{}") para reduzir
// pegada na coluna JSON e tornar o filtro `IS NULL` natural.
func toModel(log *domain.AuditLog) (*auditLogModel, error) {
	m := &auditLogModel{
		ID:         log.ID,
		TenantID:   log.TenantID,
		UserID:     log.UserID,
		ActorEmail: log.ActorEmail,
		Action:     string(log.Action),
		Entity:     log.Entity,
		EntityID:   log.EntityID,
		IP:         log.IP,
		UserAgent:  log.UserAgent,
		CreatedAt:  log.CreatedAt.UTC(),
	}
	if len(log.Metadata) > 0 {
		raw, err := json.Marshal(log.Metadata)
		if err != nil {
			return nil, err
		}
		s := string(raw)
		m.Metadata = &s
	}
	return m, nil
}

// toDomain converte o modelo Gorm para a entidade de dominio.
//
// Metadata NULL ou "null" volta como `domain.Metadata{}` (mapa vazio nao-nil)
// para que callers possam fazer assercoes sem precisar checar `nil`.
func toDomain(m *auditLogModel) (*domain.AuditLog, error) {
	metadata := domain.Metadata{}
	if m.Metadata != nil && *m.Metadata != "" && *m.Metadata != "null" {
		if err := json.Unmarshal([]byte(*m.Metadata), &metadata); err != nil {
			return nil, err
		}
	}
	return &domain.AuditLog{
		ID:         m.ID,
		TenantID:   m.TenantID,
		UserID:     m.UserID,
		ActorEmail: m.ActorEmail,
		Action:     domain.Action(m.Action),
		Entity:     m.Entity,
		EntityID:   m.EntityID,
		IP:         m.IP,
		UserAgent:  m.UserAgent,
		Metadata:   metadata,
		CreatedAt:  m.CreatedAt.UTC(),
	}, nil
}
