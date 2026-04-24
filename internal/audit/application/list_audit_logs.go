package application

import (
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/audit/domain"
	auditinfra "github.com/sasrgita/crm-juridico/internal/audit/infrastructure"
)

// ListAuditLogsUseCase devolve a pagina filtrada de audit logs e o total.
//
// Contrato (decisao do design F12, secao 3.2):
//   - Normaliza o filtro antes de chamar o repo (defesa em profundidade:
//     o repo tambem normaliza, mas chamar aqui garante que erros de filtro
//     - acao invalida, periodo invertido - nao toquem no banco).
//   - Mede a duracao total no histograma `crm_audit_logs_list_duration_seconds`.
//   - Emite span `audit.usecase.list` com atributos de filtro.
//   - Propaga erros do repo (incluindo context.Canceled) para o caller.
type ListAuditLogsUseCase struct {
	repo domain.Repository
	log  *zap.Logger
}

// NewListAuditLogsUseCase constroi o caso de uso.
//
// `logger` pode ser nil — sera substituido por zap.NewNop() para evitar
// panic em chamadas defensivas, em linha com NewRegisterAuditLogUseCase.
func NewListAuditLogsUseCase(repo domain.Repository, logger *zap.Logger) *ListAuditLogsUseCase {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ListAuditLogsUseCase{repo: repo, log: logger}
}

// Execute normaliza o filtro, consulta o repo e devolve a pagina + total.
//
// Em qualquer caminho de erro (Normalize ou repo), o histograma de duracao
// e observado mesmo assim — o tempo gasto medindo a falha tambem e parte
// da latencia percebida pelo handler.
func (uc *ListAuditLogsUseCase) Execute(
	ctx context.Context,
	filter domain.Filter,
) ([]*domain.AuditLog, int64, error) {
	ctx, span := tracer.Start(ctx, "audit.usecase.list")
	defer span.End()

	start := time.Now()
	defer func() {
		auditinfra.AuditLogsListDuration.
			WithLabelValues().
			Observe(time.Since(start).Seconds())
	}()

	if err := filter.Normalize(); err != nil {
		span.RecordError(err)
		return nil, 0, err
	}

	span.SetAttributes(
		attribute.Bool("audit.filter.has_tenant_id", filter.TenantID != nil),
		attribute.Bool("audit.filter.has_user_id", filter.UserID != nil),
		attribute.String("audit.filter.action", actionLabel(filter.Action)),
		attribute.Int("audit.page.number", filter.Page),
		attribute.Int("audit.page.size", filter.PageSize),
	)

	logs, total, err := uc.repo.List(ctx, filter)
	if err != nil {
		span.RecordError(err)
		return nil, 0, err
	}

	uc.log.Info("audit logs listed",
		zap.Int("items", len(logs)),
		zap.Int64("total", total),
		zap.String("page", strconv.Itoa(filter.Page)),
		zap.String("page_size", strconv.Itoa(filter.PageSize)),
	)
	return logs, total, nil
}

// actionLabel devolve uma representacao estavel do filtro de acao para
// atributo de span. "" quando filter.Action e nil — evita publicar
// "<nil>" no Jaeger/Tempo.
func actionLabel(a *domain.Action) string {
	if a == nil {
		return ""
	}
	return string(*a)
}
