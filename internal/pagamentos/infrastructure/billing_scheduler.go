package infrastructure

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
)

// BillingScheduler roda diariamente os UCs do cron de pagamentos:
// GenerateRecurringPayments e RefreshOverdueStatuses. Expõe RunOnce para
// testes e invocação manual.
type BillingScheduler struct {
	spec     string
	generate *application.GenerateRecurringPayments
	refresh  *application.RefreshOverdueStatuses
	log      *zap.Logger
	cron     *cron.Cron
}

func NewBillingScheduler(spec string, gen *application.GenerateRecurringPayments, ref *application.RefreshOverdueStatuses, log *zap.Logger, loc *time.Location) *BillingScheduler {
	if spec == "" {
		spec = "0 3 * * *"
	}
	if loc == nil {
		loc = time.UTC
	}
	c := cron.New(cron.WithLocation(loc))
	return &BillingScheduler{spec: spec, generate: gen, refresh: ref, log: log, cron: c}
}

func (s *BillingScheduler) Start() error {
	_, err := s.cron.AddFunc(s.spec, s.runOnce)
	if err != nil {
		return err
	}
	s.cron.Start()
	return nil
}

func (s *BillingScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}

// RunOnce executa o ciclo completo (generate + refresh) uma vez, útil para
// testes e para invocação manual.
func (s *BillingScheduler) RunOnce(ctx context.Context) (int, int, error) {
	return s.runWithCtx(ctx)
}

func (s *BillingScheduler) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	_, _, _ = s.runWithCtx(ctx)
}

func (s *BillingScheduler) runWithCtx(ctx context.Context) (int, int, error) {
	requestID := "cron-billing-" + uuid.NewString()
	log := s.log.With(zap.String("request_id", requestID))
	start := time.Now()

	nGen, err := s.generate.Execute(ctx)
	if err != nil {
		log.Error("billing cron generate failed", zap.Error(err))
		CronRunsTotal.WithLabelValues("error").Inc()
		CronDurationSeconds.Observe(time.Since(start).Seconds())
		return nGen, 0, err
	}
	if nGen > 0 {
		RecorrentesGeradosTotal.Add(float64(nGen))
	}

	nRef, err := s.refresh.Execute(ctx)
	if err != nil {
		log.Error("billing cron refresh failed", zap.Error(err))
		CronRunsTotal.WithLabelValues("error").Inc()
		CronDurationSeconds.Observe(time.Since(start).Seconds())
		return nGen, nRef, err
	}
	if nRef > 0 {
		AtualizadosAtrasadoTotal.Add(float64(nRef))
	}

	CronRunsTotal.WithLabelValues("success").Inc()
	CronDurationSeconds.Observe(time.Since(start).Seconds())
	log.Info("billing cron run",
		zap.Int("generated", nGen),
		zap.Int("overdue", nRef),
		zap.Duration("duration", time.Since(start)),
	)
	return nGen, nRef, nil
}
