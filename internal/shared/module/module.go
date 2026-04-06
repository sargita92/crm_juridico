package module

import "github.com/gin-gonic/gin"

type Module interface {
	Name() string
	RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc)
}
