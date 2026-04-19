package domain

import "errors"

var (
	ErrTenantIDRequired       = errors.New("tenant id is required")
	ErrValorInvalido          = errors.New("valor must be greater than zero")
	ErrDescricaoRequired      = errors.New("descricao is required for avulso")
	ErrDataVencimentoRequired = errors.New("data_vencimento is required")
	ErrInvalidPlano           = errors.New("invalid plano")
	ErrInvalidPaymentType     = errors.New("invalid payment type")
	ErrInvalidStatus          = errors.New("invalid payment status")
	ErrInvalidTransition      = errors.New("invalid status transition")
	ErrMotivoRequired         = errors.New("motivo is required when cancelling")
	ErrPaymentNotFound        = errors.New("payment not found")
	ErrTenantNotFound         = errors.New("tenant not found")
)
