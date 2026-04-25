// Package http: handlers full-page (sem HX-Request).
//
// Quando a request chega sem o header `HX-Request: true` precisamos
// renderizar o layout completo (sidebar + header + filtros + tabela).
// As funcoes aqui delegam a montagem do view-model para o handler.go e
// chamam o template apropriado.
//
// Mantemos os "Page" e os "Fragment" no mesmo Handler struct para que o
// composition root tenha apenas uma instancia para passar ao
// RegisterRoutes.
package http

import (
	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/audit/domain"
)

// ListPage renderiza /admin/logs em layout completo OU delega para o
// fragmento HTMX dependendo do header HX-Request.
//
// Esta funcao e o ponto de entrada do GET /admin/logs registrado nas
// rotas do modulo (routes.go). O fluxo single-handler/single-route segue
// o padrao do projeto (ver tenant/interfaces/http/handler.go RenderList).
func (h *Handler) ListPage(c *gin.Context) {
	if c.GetHeader("HX-Request") == "true" {
		h.ListFragment(c)
		return
	}

	data, status := h.buildListData(c)

	// Listers podem ser nil em testes sem dropdowns. Em prod o
	// composition root injeta os adapters; quando ausentes a UI mostra
	// dropdowns vazios sem quebrar o template.
	if h.tenantLister != nil {
		if tenants, err := h.tenantLister.ListTenants(c.Request.Context()); err == nil {
			data["TenantOptions"] = tenants
		} else {
			data["TenantOptions"] = []domain.TenantSummary{}
		}
	} else {
		data["TenantOptions"] = []domain.TenantSummary{}
	}

	if h.adminUserLister != nil {
		if users, err := h.adminUserLister.ListAdminUsers(c.Request.Context()); err == nil {
			data["UserOptions"] = users
		} else {
			data["UserOptions"] = []domain.AdminUserSummary{}
		}
	} else {
		data["UserOptions"] = []domain.AdminUserSummary{}
	}

	c.HTML(status, "admin/audit/list.html", data)
}

// DetailPage renderiza GET /admin/logs/:id no layout completo, OU delega
// para o fragmento HTMX. Em caso de log inexistente (ou id mal formado)
// devolve 404 com a pagina generica de not_found, identica ao caso
// "nao admin" do AdminOr404 (S5-C08).
func (h *Handler) DetailPage(c *gin.Context) {
	if c.GetHeader("HX-Request") == "true" {
		h.DetailFragment(c)
		return
	}

	log, err := h.fetchLog(c)
	if err != nil {
		h.renderNotFound(c)
		return
	}
	c.HTML(200, "admin/audit/detail.html", h.detailViewModel(c, log))
}
