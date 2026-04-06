package module

import "github.com/gin-gonic/gin"

type Middlewares struct {
	Auth   gin.HandlerFunc
	Tenant gin.HandlerFunc
	Admin  gin.HandlerFunc
}

type Module interface {
	Name() string
	RegisterRoutes(router *gin.Engine, mw Middlewares)
}
