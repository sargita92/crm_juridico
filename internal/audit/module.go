package audit

import (
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/audit/application"
	"github.com/sasrgita/crm-juridico/internal/audit/domain"
	"github.com/sasrgita/crm-juridico/internal/audit/infrastructure"
)

// Module e o composition root do dominio audit (F12).
//
// Estado atual (Step 3): expoe o RegisterUC, o Publisher default e o
// repositorio para outras features injetarem nas suas use cases. Os
// handlers HTTP, ListUC e GetUC ficam para Steps 4 e 8 — quando esses
// existirem, este struct cresce e o `RegisterRoutes` aparece.
//
// Fluxo de uso (futuro Step 5):
//
//	mod := audit.NewModule(db, log)
//	authModule.SetAuditPublisher(mod.Publisher)
type Module struct {
	RegisterUC *application.RegisterAuditLogUseCase
	Publisher  application.Publisher
	Repo       domain.Repository
}

// NewModule monta as dependencias do contexto audit.
//
// Wire-up minimo: repositorio Gorm + caso de uso de registro + publisher
// default (engole erro com WARN, decisao do design F12 secao 3.1).
//
// `logger` pode ser nil — substituido por Nop dentro do UC/Publisher.
func NewModule(db *gorm.DB, logger *zap.Logger) *Module {
	if logger == nil {
		logger = zap.NewNop()
	}

	repo := infrastructure.NewGormAuditLogRepository(db)
	registerUC := application.NewRegisterAuditLogUseCase(repo, logger)
	publisher := application.NewPublisher(registerUC, logger)

	return &Module{
		RegisterUC: registerUC,
		Publisher:  publisher,
		Repo:       repo,
	}
}
