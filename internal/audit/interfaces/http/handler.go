// Package http: handler de fragmentos HTMX para a feature F12.
//
// Embora os fragments e a pagina full compartilhem o mesmo `Handler`
// struct (mais simples para wiring), separamos os metodos em dois
// arquivos para facilitar leitura: este arquivo cobre as variantes
// `*Fragment` que sao chamadas quando a request vem com `HX-Request: true`.
package http

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/audit/application"
	"github.com/sasrgita/crm-juridico/internal/audit/domain"
)

// Handler reune as dependencias compartilhadas entre ListPage/DetailPage
// (pagina completa) e ListFragment/DetailFragment (fragmento HTMX).
//
// `tenantLister` e `adminUserLister` sao opcionais: quando nil os
// dropdowns de filtro sao renderizados vazios — util em testes que nao
// querem montar um repo de tenants/users.
type Handler struct {
	listUC          *application.ListAuditLogsUseCase
	getUC           *application.GetAuditLogUseCase
	tenantLister    domain.TenantLister
	adminUserLister domain.AdminUserLister
}

// NewHandler constroi o Handler. Ambos os listers podem ser nil em testes
// ou quando o composition root nao quer expor dropdowns ainda.
func NewHandler(
	listUC *application.ListAuditLogsUseCase,
	getUC *application.GetAuditLogUseCase,
	tenantLister domain.TenantLister,
	adminUserLister domain.AdminUserLister,
) *Handler {
	return &Handler{
		listUC:          listUC,
		getUC:           getUC,
		tenantLister:    tenantLister,
		adminUserLister: adminUserLister,
	}
}

// ListFragment responde apenas o `<tbody>` + paginacao para HTMX swap.
// E chamado pelo mesmo path `GET /admin/logs` quando o header
// `HX-Request: true` esta presente (ver ListPage). Mantido como metodo
// separado pra facilitar testes que nao precisam montar layout completo.
func (h *Handler) ListFragment(c *gin.Context) {
	data, status := h.buildListData(c)
	c.HTML(status, "admin/audit/list_table.html", data)
}

// DetailFragment responde o card de detalhe do log para HTMX swap.
//
// Em caso de id inexistente devolve `audit/not_found.html` com 404 — o
// mesmo template usado pelo middleware AdminOr404, garantindo a
// indistinguibilidade exigida pelo cenario S5-C08.
func (h *Handler) DetailFragment(c *gin.Context) {
	log, err := h.fetchLog(c)
	if err != nil {
		h.renderNotFound(c)
		return
	}
	c.HTML(http.StatusOK, "admin/audit/detail_fragment.html", h.detailViewModel(c, log))
}

// fetchLog busca o log pelo id na URL; trata id vazio como not found para
// nunca vazar a diferenca entre "id mal formado" e "id inexistente"
// (S4-C14).
func (h *Handler) fetchLog(c *gin.Context) (*domain.AuditLog, error) {
	id := c.Param("id")
	log, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrAuditLogNotFound) {
			return nil, err
		}
		return nil, err
	}
	return log, nil
}

// renderNotFound centraliza o 404 generico. Body identico ao do
// AdminOr404 — parte da defesa OWASP A01 (S5-C08).
func (h *Handler) renderNotFound(c *gin.Context) {
	c.HTML(http.StatusNotFound, "admin/audit/not_found.html", gin.H{})
}

// buildListData parse-eia a querystring, normaliza pagina/page_size e
// chama o ListUC. Erros de filtro (acao invalida, periodo invertido)
// resultam em status 400 + tabela vazia + mensagem amigavel — decisao
// 2026-04-24 do usuario (rejeitar com 400, mas com pagina renderizada).
//
// Retorna o map de dados pronto para o template + status code que o
// caller (Page ou Fragment) deve usar no `c.HTML`.
func (h *Handler) buildListData(c *gin.Context) (gin.H, int) {
	filter, parseErr := parseFilterFromQuery(c)
	data := gin.H{
		"Filters":    formFilterValues(c),
		"Items":      []*domain.AuditLog{},
		"Total":      int64(0),
		"Page":       filter.Page,
		"PageSize":   filter.PageSize,
		"PageSizes":  []int{10, 25, 50, 100},
		"Actions":    domain.AllActions(),
		"ActiveNav":  "logs",
		"ReturnQS":   c.Request.URL.RawQuery,
	}

	if parseErr != nil {
		// Erro de parse (date malformed / action fora do enum / page_size
		// mal posicionado): seguimos renderizando a tabela vazia com a
		// mensagem para o usuario corrigir os filtros.
		data["Error"] = humanizeFilterErr(parseErr)
		return data, http.StatusBadRequest
	}

	logs, total, err := h.listUC.Execute(c.Request.Context(), filter)
	if err != nil {
		// Erros de UC (invalid period, invalid action) tambem caem como 400
		// em S3-C17/S3-C18.
		if errors.Is(err, domain.ErrInvalidPeriod) || errors.Is(err, domain.ErrInvalidAction) {
			data["Error"] = humanizeFilterErr(err)
			return data, http.StatusBadRequest
		}
		data["Error"] = "Nao foi possivel carregar os logs."
		return data, http.StatusInternalServerError
	}

	data["Items"] = logs
	data["Total"] = total
	data["StartItem"] = (filter.Page-1)*filter.PageSize + 1
	end := int64(filter.Page * filter.PageSize)
	if end > total {
		end = total
	}
	data["EndItem"] = end
	data["HasPrev"] = filter.Page > 1
	data["HasNext"] = int64(filter.Page*filter.PageSize) < total
	data["PrevPage"] = filter.Page - 1
	data["NextPage"] = filter.Page + 1
	if total == 0 {
		data["IsEmpty"] = true
	}
	return data, http.StatusOK
}

// detailViewModel monta o map para o template de detalhe.
//
// Inclui o `ReturnURL` baseado em `?return=` (preserva filtros do
// usuario, S4-C08). Quando vazio, cai em `/admin/logs`.
func (h *Handler) detailViewModel(c *gin.Context, log *domain.AuditLog) gin.H {
	returnURL := "/admin/logs"
	if r := c.Query("return"); r != "" {
		returnURL = "/admin/logs?" + r
	}
	return gin.H{
		"Log":         log,
		"Action":      string(log.Action),
		"ActionLabel": log.Action.Humanized(),
		"ReturnURL":   returnURL,
		"ActiveNav":   "logs",
	}
}

// parseFilterFromQuery converte a query string em domain.Filter.
//
// Decisoes alinhadas ao usuario (2026-04-24):
//   - page_size invalido (>100 ou !=10/25/50/100) e clampado em
//     domain.MaxPageSize via Filter.Normalize — nao 400.
//   - action fora do enum -> 400 (decisao 2).
//   - tenant_id/user_id mal formado -> 400 (S3-C21).
//   - from/to mal formados -> 400 (S3-C22).
func parseFilterFromQuery(c *gin.Context) (domain.Filter, error) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	filter := domain.Filter{
		Page:     page,
		PageSize: pageSize,
	}

	if v := c.Query("tenant_id"); v != "" {
		filter.TenantID = strPtr(v)
	}
	if v := c.Query("user_id"); v != "" {
		filter.UserID = strPtr(v)
	}
	if v := c.Query("action"); v != "" {
		a := domain.Action(v)
		if !a.IsValid() {
			return filter, domain.ErrInvalidAction
		}
		filter.Action = &a
	}
	if v := c.Query("from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return filter, errInvalidDate
		}
		filter.From = &t
	}
	if v := c.Query("to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return filter, errInvalidDate
		}
		// Inclui o dia inteiro: ate 23:59:59.999.
		eod := t.Add(24*time.Hour - time.Millisecond)
		filter.To = &eod
	}
	return filter, nil
}

var errInvalidDate = errors.New("invalid date format")

// humanizeFilterErr converte erros de validacao em texto amigavel para o
// usuario admin. Mantemos curtos e nao-tecnicos (sem mostrar nome de
// erro Go).
func humanizeFilterErr(err error) string {
	switch {
	case errors.Is(err, domain.ErrInvalidAction):
		return "Acao selecionada e invalida."
	case errors.Is(err, domain.ErrInvalidPeriod):
		return "Periodo invalido: a data inicial deve ser anterior ou igual a final."
	case errors.Is(err, errInvalidDate):
		return "Data invalida. Use o formato AAAA-MM-DD."
	default:
		return "Filtros invalidos."
	}
}

// formFilterValues devolve um map de strings prontas para alimentar os
// inputs do form de filtros (decoupled de domain.Filter para nao expor
// ponteiros no template).
func formFilterValues(c *gin.Context) gin.H {
	return gin.H{
		"TenantID": c.Query("tenant_id"),
		"UserID":   c.Query("user_id"),
		"Action":   c.Query("action"),
		"From":     c.Query("from"),
		"To":       c.Query("to"),
		"PageSize": c.Query("page_size"),
	}
}

func strPtr(s string) *string { return &s }
