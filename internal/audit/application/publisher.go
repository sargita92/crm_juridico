package application

import (
	"context"

	"go.uber.org/zap"
)

// Publisher e o ponto de injecao usado pelas features que produzem eventos
// auditaveis (auth, tenant, permission). Implementacoes nao devem propagar
// erro — a politica do projeto e "falha de auditoria nao quebra a operacao".
//
// Para casos onde o caller precisa do erro (testes de integracao da F12,
// por exemplo), use `RegisterAuditLogUseCase` diretamente.
type Publisher interface {
	Publish(ctx context.Context, in RegisterAuditLogInput) error
}

// defaultPublisher chama o RegisterAuditLogUseCase e converte qualquer erro
// em log WARN, retornando nil para o caller. Implementa a decisao do design
// F12 secao 3.1 + 4.1: bloquear bloqueio de tenant porque um INSERT na
// tabela de logs falhou e pior do que perder um registro de auditoria.
type defaultPublisher struct {
	uc  *RegisterAuditLogUseCase
	log *zap.Logger
}

// NewPublisher devolve o publisher padrao do projeto.
//
// `logger` pode ser nil (substituido por Nop). O UC ja loga internamente
// via `*RegisterAuditLogUseCase.log` — este logger e usado para registrar
// que o publisher engoliu o erro (sinal extra para correlacao com a
// metrica `audit_logs_registered_total{status="error"}`).
func NewPublisher(uc *RegisterAuditLogUseCase, logger *zap.Logger) Publisher {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &defaultPublisher{uc: uc, log: logger}
}

// Publish invoca o caso de uso e absorve qualquer erro com WARN.
//
// Sempre retorna nil (a assinatura mantem `error` para futuras politicas
// de back-pressure ou metricas adicionais sem quebrar callers).
func (p *defaultPublisher) Publish(ctx context.Context, in RegisterAuditLogInput) error {
	if err := p.uc.Execute(ctx, in); err != nil {
		p.log.Warn("audit publisher swallowed error",
			zap.Error(err),
			zap.String("action", string(in.Action)),
			zap.String("actor_email", in.ActorEmail),
			zap.Stringp("tenant_id", in.TenantID),
		)
	}
	return nil
}

// NoopPublisher e usado em testes de outras features para nao acoplar a
// suite ao banco/UC reais. Exportado de proposito — outros pacotes podem
// importar e usar como `audit.NoopPublisher{}`.
type NoopPublisher struct{}

// Publish e no-op: nao chama UC, nao registra metricas, retorna nil.
func (NoopPublisher) Publish(_ context.Context, _ RegisterAuditLogInput) error {
	return nil
}
