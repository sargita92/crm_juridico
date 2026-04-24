package domain

import "context"

// Repository e o contrato de persistencia dos audit logs.
//
// Logs sao imutaveis: nao expomos Update nem Delete. Implementacoes Gorm
// vao para internal/audit/infrastructure (Step 2).
type Repository interface {
	// Create persiste um novo audit log.
	// Implementacoes nao devem propagar erro para o caso de uso da operacao
	// auditada — isso e responsabilidade do publisher (Step 3).
	Create(ctx context.Context, log *AuditLog) error

	// List devolve a pagina filtrada e o total de registros (para paginacao).
	// O Filter ja deve estar normalizado (caller invoca Filter.Normalize()).
	// Resultados sao ordenados por CreatedAt DESC (mais recentes primeiro).
	List(ctx context.Context, filter Filter) (logs []*AuditLog, total int64, err error)

	// FindByID retorna o log pelo ID. Devolve ErrAuditLogNotFound quando
	// o registro nao existe.
	FindByID(ctx context.Context, id string) (*AuditLog, error)
}
