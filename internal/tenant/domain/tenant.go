package domain

import (
	"errors"
	"time"
)

type TenantType string

const (
	TenantTypePF TenantType = "PF"
	TenantTypePJ TenantType = "PJ"
)

type TenantStatus string

const (
	TenantStatusActive   TenantStatus = "active"
	TenantStatusBlocked  TenantStatus = "blocked"
	TenantStatusInactive TenantStatus = "inactive"
)

var (
	ErrTenantNotFound        = errors.New("tenant not found")
	ErrTenantDocumentExists  = errors.New("tenant with this document already exists")
	ErrTenantBlocked         = errors.New("tenant is blocked")
	ErrTenantInactive        = errors.New("tenant is inactive")
	ErrInvalidTenantType     = errors.New("invalid tenant type: must be PF or PJ")
	ErrTenantNameRequired    = errors.New("tenant name is required")
	ErrTenantDocRequired     = errors.New("tenant document is required")
	ErrBlockReasonRequired   = errors.New("block reason is required")
	ErrUnblockReasonRequired = errors.New("unblock reason is required")
	ErrTenantAlreadyBlocked  = errors.New("tenant is already blocked")
	ErrTenantNotBlocked      = errors.New("tenant is not blocked")
	ErrReasonTooLong         = errors.New("reason must be at most 500 characters")
)

type Tenant struct {
	ID                 string
	Name               string
	Type               TenantType
	Document           string
	Status             TenantStatus
	BlockReason        string
	Plano              string
	ValorCobrancaCents *int64
	DiaVencimento      *uint8
	DataInicioCobranca *time.Time
	ExibirPagamentos   bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NewTenant(id, name string, tenantType TenantType, document string) (*Tenant, error) {
	if name == "" {
		return nil, ErrTenantNameRequired
	}
	if document == "" {
		return nil, ErrTenantDocRequired
	}
	if tenantType != TenantTypePF && tenantType != TenantTypePJ {
		return nil, ErrInvalidTenantType
	}

	now := time.Now()
	return &Tenant{
		ID:               id,
		Name:             name,
		Type:             tenantType,
		Document:         document,
		Status:           TenantStatusActive,
		Plano:            "mensal",
		ExibirPagamentos: true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// SetBillingConfig atualiza os campos de cobrança do tenant. A validação de
// combinações válidas (ex: plano mensal exige valor/dia/data) é feita na
// aplicação antes de chamar esse setter.
func (t *Tenant) SetBillingConfig(plano string, valorCents *int64, dia *uint8, start *time.Time, exibir bool) {
	t.Plano = plano
	t.ValorCobrancaCents = valorCents
	t.DiaVencimento = dia
	t.DataInicioCobranca = start
	t.ExibirPagamentos = exibir
	t.UpdatedAt = time.Now()
}

func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

func (t *Tenant) IsBlocked() bool {
	return t.Status == TenantStatusBlocked
}

func (t *Tenant) IsInactive() bool {
	return t.Status == TenantStatusInactive
}

func (t *Tenant) Block(reason string) error {
	if reason == "" {
		return ErrBlockReasonRequired
	}
	if len(reason) > 500 {
		return ErrReasonTooLong
	}
	if t.Status == TenantStatusBlocked {
		return ErrTenantAlreadyBlocked
	}
	if t.Status == TenantStatusInactive {
		return ErrTenantInactive
	}
	t.Status = TenantStatusBlocked
	t.BlockReason = reason
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Tenant) Unblock(reason string) error {
	if reason == "" {
		return ErrUnblockReasonRequired
	}
	if len(reason) > 500 {
		return ErrReasonTooLong
	}
	if t.Status != TenantStatusBlocked {
		return ErrTenantNotBlocked
	}
	t.Status = TenantStatusActive
	t.BlockReason = ""
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Tenant) Deactivate() error {
	if t.Status == TenantStatusInactive {
		return ErrTenantInactive
	}
	t.Status = TenantStatusInactive
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Tenant) Update(name string, tenantType TenantType, document string) error {
	if name == "" {
		return ErrTenantNameRequired
	}
	if document == "" {
		return ErrTenantDocRequired
	}
	if tenantType != TenantTypePF && tenantType != TenantTypePJ {
		return ErrInvalidTenantType
	}
	t.Name = name
	t.Type = tenantType
	t.Document = document
	t.UpdatedAt = time.Now()
	return nil
}
