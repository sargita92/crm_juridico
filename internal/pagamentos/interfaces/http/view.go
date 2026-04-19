package http

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

type paymentView struct {
	ID             string
	Tipo           string
	Descricao      string
	Valor          string // "R$ 50,00"
	Status         string
	StatusBadge    string
	Competencia    string // "MM/YYYY" ou ""
	DataVencimento string // "DD/MM/YYYY"
	DataPagamento  string
	PodeMarcarPago bool
	PodeCancelar   bool
	TenantID       string
}

func toPaymentView(p domain.Payment) paymentView {
	pv := paymentView{
		ID:             p.ID,
		Tipo:           string(p.Tipo),
		Descricao:      p.Descricao,
		Valor:          formatCents(p.ValorCents),
		Status:         string(p.Status),
		StatusBadge:    statusBadgeClass(p.Status),
		DataVencimento: p.DataVencimento.Format("02/01/2006"),
		TenantID:       p.TenantID,
	}
	if p.Competencia != nil {
		pv.Competencia = p.Competencia.Format("01/2006")
	}
	if p.DataPagamento != nil {
		pv.DataPagamento = p.DataPagamento.Format("02/01/2006 15:04")
	}
	pv.PodeMarcarPago = p.Status == domain.StatusPendente || p.Status == domain.StatusAtrasado
	pv.PodeCancelar = p.Status != domain.StatusPago && p.Status != domain.StatusCancelado
	return pv
}

func formatCents(c int64) string {
	return fmt.Sprintf("R$ %.2f", float64(c)/100.0)
}

func statusBadgeClass(s domain.PaymentStatus) string {
	switch s {
	case domain.StatusPago:
		return "badge badge-success"
	case domain.StatusPendente:
		return "badge badge-warning"
	case domain.StatusAtrasado:
		return "badge badge-danger"
	case domain.StatusCancelado:
		return "badge badge-muted"
	}
	return "badge"
}

type summaryView struct {
	Badge         string
	BadgeText     string
	TotalPagoAno  string
	TotalPendente string
	TotalAtrasado string
}

func toSummaryView(s *application.FinancialSummaryOutput) summaryView {
	text := map[application.FinancialBadge]string{
		application.BadgeEmDia:       "Em dia",
		application.BadgePendente:    "Pendente",
		application.BadgeAtrasado:    "Atrasado",
		application.BadgeSemCobranca: "Sem cobrança",
	}
	return summaryView{
		Badge:         string(s.Badge),
		BadgeText:     text[s.Badge],
		TotalPagoAno:  formatCents(s.TotalPagoAnoCents),
		TotalPendente: formatCents(s.TotalPendenteCents),
		TotalAtrasado: formatCents(s.TotalAtrasadoCents),
	}
}

type queryReader interface {
	Query(string) string
}

func parseFiltersFromQuery(c queryReader) (status *domain.PaymentStatus, di, df *time.Time, page int) {
	if v := c.Query("status"); v != "" {
		s := domain.PaymentStatus(v)
		status = &s
	}
	if v := c.Query("data_inicial"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			di = &t
		}
	}
	if v := c.Query("data_final"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			df = &t
		}
	}
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			page = n
		}
	}
	if page < 1 {
		page = 1
	}
	return
}

func parseValorCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, domain.ErrValorInvalido
	}
	s = strings.ReplaceAll(s, ",", ".")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if f <= 0 {
		return 0, domain.ErrValorInvalido
	}
	return int64(f*100 + 0.5), nil
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
