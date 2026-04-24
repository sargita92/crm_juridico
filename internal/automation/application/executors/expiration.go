package executors

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/sasrgita/crm-juridico/internal/automation/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// ExpirationExecutor handles leads that have been in a column longer than the configured duration.
// It either archives (moves to lost column) or soft-deletes them.
type ExpirationExecutor struct {
	leadFinder    domain.LeadFinder
	leadMover     domain.LeadMover
	leadDeleter   domain.LeadDeleter
	lostColFinder domain.LostColumnFinder
}

func NewExpirationExecutor(
	f domain.LeadFinder,
	m domain.LeadMover,
	d domain.LeadDeleter,
	l domain.LostColumnFinder,
) *ExpirationExecutor {
	return &ExpirationExecutor{
		leadFinder:    f,
		leadMover:     m,
		leadDeleter:   d,
		lostColFinder: l,
	}
}

func (e *ExpirationExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) (err error) {
	ctx, span := observability.StartSpan(ctx, "automation.executor.expiration",
		attribute.String("tenant.id", tenantID),
		attribute.String("lead.id", leadID),
		attribute.String("automation.id", a.ID),
	)
	defer span.End()

	start := time.Now()
	defer func() {
		outcome := "success"
		if err != nil {
			outcome = "error"
		}
		infrastructure.ExecutionDuration.WithLabelValues("expiration", outcome).Observe(time.Since(start).Seconds())
	}()

	action := a.ConfigString("action")
	lead, err := e.leadFinder.FindByID(ctx, leadID)
	if err != nil {
		return err
	}
	switch action {
	case "archive":
		lostColID, err := e.lostColFinder.FindLostColumn(ctx, lead.FunnelID)
		if err != nil {
			return err
		}
		return e.leadMover.MoveLead(ctx, tenantID, leadID, lostColID, "")
	case "delete":
		return e.leadDeleter.SoftDelete(ctx, leadID)
	default:
		return domain.ErrInvalidConfig
	}
}

// FindExpiredLeads returns the IDs of leads in the automation's column that have exceeded
// the configured duration_hours threshold.
func (e *ExpirationExecutor) FindExpiredLeads(ctx context.Context, a *domain.Automation) ([]string, error) {
	hours := a.ConfigFloat("duration_hours")
	if hours <= 0 {
		return nil, nil
	}
	maxAge := time.Duration(hours) * time.Hour
	return e.leadFinder.FindExpiredInColumn(ctx, a.ColumnID, maxAge)
}
