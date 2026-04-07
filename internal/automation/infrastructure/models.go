package infrastructure

import (
	"encoding/json"
	"time"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
)

// --- automationModel ---

type automationModel struct {
	ID        string    `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID  string    `gorm:"column:tenant_id;type:char(36);not null"`
	FunnelID  string    `gorm:"column:funnel_id;type:char(36);not null"`
	ColumnID  *string   `gorm:"column:column_id;type:char(36)"`
	Type      string    `gorm:"column:type;type:varchar(30);not null"`
	Config    string    `gorm:"column:config;type:json;not null"`
	Active    bool      `gorm:"column:active;type:tinyint(1);not null;default:1"`
	Priority  int       `gorm:"column:priority;type:int;not null;default:0"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (automationModel) TableName() string { return "automations" }

func automationToModel(a *domain.Automation) *automationModel {
	cfg, _ := json.Marshal(a.Config)
	m := &automationModel{
		ID:        a.ID,
		TenantID:  a.TenantID,
		FunnelID:  a.FunnelID,
		Type:      string(a.Type),
		Config:    string(cfg),
		Active:    a.Active,
		Priority:  a.Priority,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
	if a.ColumnID != "" {
		v := a.ColumnID
		m.ColumnID = &v
	}
	return m
}

func automationToDomain(m *automationModel) *domain.Automation {
	var cfg map[string]interface{}
	if m.Config != "" {
		_ = json.Unmarshal([]byte(m.Config), &cfg)
	}
	if cfg == nil {
		cfg = map[string]interface{}{}
	}
	a := &domain.Automation{
		ID:        m.ID,
		TenantID:  m.TenantID,
		FunnelID:  m.FunnelID,
		Type:      domain.AutomationType(m.Type),
		Config:    cfg,
		Active:    m.Active,
		Priority:  m.Priority,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if m.ColumnID != nil {
		a.ColumnID = *m.ColumnID
	}
	return a
}

// --- executionLogModel ---

type executionLogModel struct {
	ID           string    `gorm:"primaryKey;column:id;type:char(36)"`
	AutomationID string    `gorm:"column:automation_id;type:char(36);not null"`
	LeadID       string    `gorm:"column:lead_id;type:char(36);not null"`
	TenantID     string    `gorm:"column:tenant_id;type:char(36);not null"`
	Status       string    `gorm:"column:status;type:varchar(20);not null"`
	ErrorMessage *string   `gorm:"column:error_message;type:text"`
	ExecutedAt   time.Time `gorm:"column:executed_at"`
}

func (executionLogModel) TableName() string { return "execution_logs" }

func executionLogToModel(l *domain.ExecutionLog) *executionLogModel {
	m := &executionLogModel{
		ID:           l.ID,
		AutomationID: l.AutomationID,
		LeadID:       l.LeadID,
		TenantID:     l.TenantID,
		Status:       string(l.Status),
		ExecutedAt:   l.ExecutedAt,
	}
	if l.ErrorMessage != "" {
		v := l.ErrorMessage
		m.ErrorMessage = &v
	}
	return m
}

func executionLogToDomain(m *executionLogModel) *domain.ExecutionLog {
	l := &domain.ExecutionLog{
		ID:           m.ID,
		AutomationID: m.AutomationID,
		LeadID:       m.LeadID,
		TenantID:     m.TenantID,
		Status:       domain.ExecutionStatus(m.Status),
		ExecutedAt:   m.ExecutedAt,
	}
	if m.ErrorMessage != nil {
		l.ErrorMessage = *m.ErrorMessage
	}
	return l
}

// --- rateLimitCounterModel ---

type rateLimitCounterModel struct {
	ID           string    `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID     string    `gorm:"column:tenant_id;type:char(36);not null"`
	SpecialistID *string   `gorm:"column:specialist_id;type:char(36)"`
	PeriodStart  time.Time `gorm:"column:period_start"`
	MessageCount int       `gorm:"column:message_count;type:int;not null;default:0"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (rateLimitCounterModel) TableName() string { return "rate_limit_counters" }

func rateLimitToModel(r *domain.RateLimitCounter) *rateLimitCounterModel {
	m := &rateLimitCounterModel{
		ID:           r.ID,
		TenantID:     r.TenantID,
		PeriodStart:  r.PeriodStart,
		MessageCount: r.MessageCount,
		CreatedAt:    r.CreatedAt,
	}
	if r.SpecialistID != "" {
		v := r.SpecialistID
		m.SpecialistID = &v
	}
	return m
}

func rateLimitToDomain(m *rateLimitCounterModel) *domain.RateLimitCounter {
	r := &domain.RateLimitCounter{
		ID:           m.ID,
		TenantID:     m.TenantID,
		PeriodStart:  m.PeriodStart,
		MessageCount: m.MessageCount,
		CreatedAt:    m.CreatedAt,
	}
	if m.SpecialistID != nil {
		r.SpecialistID = *m.SpecialistID
	}
	return r
}
