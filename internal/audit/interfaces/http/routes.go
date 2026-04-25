// Package http expoe os handlers HTTP do dominio audit (F12 — listagem e
// detalhe de audit logs no painel admin).
//
// Os handlers seguem o padrao das demais features admin: GET sem header
// HX-Request renderiza a pagina completa (layout + sidebar); GET com
// HX-Request renderiza apenas o fragmento de tabela/detalhe para swap
// HTMX. A rota e protegida por um pipeline page-friendly:
//
//  1. middleware.AdminPageAuth(tokenProvider): tenta resolver o token sem
//     rejeitar quando ausente — popula claims se conseguir.
//  2. middleware.AdminOr404(): com claims => exige role admin (404 caso
//     contrario, S5-C12); sem claims => redirect 302 para /admin/login
//     (S5-C04, decisao do usuario 2026-04-24 obs.3).
//
// Esta composicao foi escolhida para nao quebrar o middleware Auth padrao
// (que devolve 401 JSON nas outras rotas admin) — assim outros endpoints
// JSON-first continuam intactos.
package http

import (
	"github.com/gin-gonic/gin"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// RegisterRoutes monta as rotas /admin/logs no Gin engine.
//
// `tokenProvider` e necessario para o `AdminPageAuth`. O middleware
// AdminOr404 e independente do provider — apenas le claims do contexto.
func (h *Handler) RegisterRoutes(router *gin.Engine, tokenProvider authdomain.TokenProvider) {
	group := router.Group(
		"/admin/logs",
		middleware.AdminPageAuth(tokenProvider),
		middleware.AdminOr404(),
	)
	{
		group.GET("", h.ListPage)
		group.GET("/:id", h.DetailPage)
	}
}
