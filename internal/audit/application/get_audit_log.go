package application

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/audit/domain"
)

// GetAuditLogUseCase recupera um audit log pelo ID.
//
// Contrato (decisao do design F12, secao 3.3):
//   - ID vazio (apos trim) retorna domain.ErrAuditLogNotFound — handler
//     converte em 404 generico (Tela 2 estado "Log nao encontrado",
//     OWASP A01: nao distinguir "id invalido" de "log nao existe").
//   - Erros do repo sao propagados (incluindo context.Canceled).
//   - Span: `audit.usecase.get` com atributo audit.id.
type GetAuditLogUseCase struct {
	repo domain.Repository
	log  *zap.Logger
}

// NewGetAuditLogUseCase constroi o caso de uso.
//
// `logger` pode ser nil — sera substituido por zap.NewNop().
func NewGetAuditLogUseCase(repo domain.Repository, logger *zap.Logger) *GetAuditLogUseCase {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &GetAuditLogUseCase{repo: repo, log: logger}
}

// Execute valida o id e devolve o log.
func (uc *GetAuditLogUseCase) Execute(ctx context.Context, id string) (*domain.AuditLog, error) {
	ctx, span := tracer.Start(ctx, "audit.usecase.get")
	defer span.End()

	if strings.TrimSpace(id) == "" {
		span.RecordError(domain.ErrAuditLogNotFound)
		return nil, domain.ErrAuditLogNotFound
	}
	span.SetAttributes(attribute.String("audit.id", id))

	log, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	uc.log.Info("audit log fetched",
		zap.String("audit_log_id", log.ID),
		zap.String("action", string(log.Action)),
	)
	return log, nil
}
