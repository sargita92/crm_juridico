package domain

import "errors"

// Erros de dominio do pacote audit/domain.
//
// Sao sentinelas (sem wrapping) para que callers possam usar errors.Is.
// Mensagens em ingles, em linha com o padrao dos demais dominios do projeto
// (ver internal/tenant/domain/tenant.go e internal/auth/domain).
var (
	// ErrAuditLogNotFound retornado quando FindByID nao encontra o registro.
	ErrAuditLogNotFound = errors.New("audit log not found")

	// ErrInvalidAction retornado quando a Action nao pertence ao enum.
	ErrInvalidAction = errors.New("invalid audit action")

	// ErrActorEmailRequired retornado quando ActorEmail esta vazio.
	ErrActorEmailRequired = errors.New("actor email is required")

	// ErrEntityRequired retornado quando Entity esta vazia.
	ErrEntityRequired = errors.New("entity is required")

	// ErrIPRequired retornado quando IP esta vazio.
	ErrIPRequired = errors.New("ip address is required")

	// ErrInvalidPeriod retornado quando o filtro tem From > To.
	ErrInvalidPeriod = errors.New("period 'from' must be before or equal to 'to'")
)
