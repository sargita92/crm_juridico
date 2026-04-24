package application

import (
	"context"

	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/audit/domain"
	auditinfra "github.com/sasrgita/crm-juridico/internal/audit/infrastructure"
)

// RegisterAuditLogInput agrupa os parametros do caso de uso.
//
// Espelha 1:1 os campos de domain.NewAuditLogInput — a camada de aplicacao
// nao adiciona invariantes proprios; delega validacao + sanitizacao ao
// construtor de dominio (defesa em profundidade unica e centralizada).
type RegisterAuditLogInput struct {
	TenantID   *string
	UserID     *string
	ActorEmail string
	Action     domain.Action
	Entity     string
	EntityID   *string
	IP         string
	UserAgent  string
	Metadata   domain.Metadata
}

// RegisterAuditLogUseCase persiste um evento auditavel.
//
// Contrato (decisao do design F12, secao 3.1):
//   - Valida + sanitiza via domain.NewAuditLog.
//   - Persiste via domain.Repository.Create.
//   - Em sucesso: incrementa AuditLogsRegisteredTotal{action,status="success"}.
//   - Em falha: incrementa AuditLogsRegisteredTotal{action,status="error"},
//     loga WARN com action+actor_email+erro e devolve o erro para o caller.
//
// **Importante**: este UC nao decide se aborta a operacao de negocio. Quem
// engole o erro (politica "auditoria nao quebra fluxo") e o Publisher
// (ver publisher.go). UC retorna o erro real para callers que queiram
// propagar (testes de integracao, retries, etc.).
type RegisterAuditLogUseCase struct {
	repo domain.Repository
	log  *zap.Logger
}

// NewRegisterAuditLogUseCase constroi o caso de uso.
//
// `logger` pode ser nil — sera substituido por zap.NewNop() para evitar
// panic em chamadas defensivas.
func NewRegisterAuditLogUseCase(repo domain.Repository, logger *zap.Logger) *RegisterAuditLogUseCase {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RegisterAuditLogUseCase{repo: repo, log: logger}
}

// Execute valida o input, monta a entidade de dominio e persiste.
//
// Em qualquer caminho de erro (validacao ou repo), o counter de erro e
// incrementado com o action recebido (mesmo que seja invalido — labels nao
// sao escolhidos por whitelist aqui; o gargalo de cardinalidade ja existe
// em domain.Action.IsValid e se manifesta como "error" agrupado).
func (uc *RegisterAuditLogUseCase) Execute(ctx context.Context, in RegisterAuditLogInput) error {
	log, err := domain.NewAuditLog(domain.NewAuditLogInput{
		TenantID:   in.TenantID,
		UserID:     in.UserID,
		ActorEmail: in.ActorEmail,
		Action:     in.Action,
		Entity:     in.Entity,
		EntityID:   in.EntityID,
		IP:         in.IP,
		UserAgent:  in.UserAgent,
		Metadata:   in.Metadata,
	})
	if err != nil {
		auditinfra.AuditLogsRegisteredTotal.
			WithLabelValues(string(in.Action), "error").Inc()
		uc.log.Warn("audit log validation failed",
			zap.Error(err),
			zap.String("action", string(in.Action)),
			zap.String("actor_email", in.ActorEmail),
		)
		return err
	}

	if err := uc.repo.Create(ctx, log); err != nil {
		auditinfra.AuditLogsRegisteredTotal.
			WithLabelValues(string(in.Action), "error").Inc()
		uc.log.Warn("audit log persist failed",
			zap.Error(err),
			zap.String("action", string(in.Action)),
			zap.String("actor_email", in.ActorEmail),
		)
		return err
	}

	auditinfra.AuditLogsRegisteredTotal.
		WithLabelValues(string(in.Action), "success").Inc()
	uc.log.Info("audit log registered",
		zap.String("audit_log_id", log.ID),
		zap.String("action", string(log.Action)),
		zap.Stringp("tenant_id", log.TenantID),
	)
	return nil
}
