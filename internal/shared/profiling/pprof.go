// Package profiling expõe os handlers de runtime profiling (net/http/pprof)
// de forma controlada: só quando habilitado por flag e atrás dos middlewares
// de proteção informados. Usado para investigar gargalos que não estejam no
// banco (CPU, alocação, goroutines) — ver F26.
package profiling

import (
	"net/http/pprof"

	"github.com/gin-gonic/gin"
)

// runtimeProfiles são servidos por pprof.Handler(name): heap, goroutine, etc.
var runtimeProfiles = []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"}

// RegisterPprof monta /debug/pprof/* no router quando enabled é true, aplicando
// os middlewares guard (ex.: auth + admin). Quando enabled é false, nenhuma rota
// é registrada — o endpoint simplesmente não existe.
//
// Profiles caros/bloqueantes (profile, trace) ficam sob o mesmo guard; cuidado
// ao acioná-los em produção pois pausam/coletam por vários segundos.
func RegisterPprof(r gin.IRouter, enabled bool, guard ...gin.HandlerFunc) {
	if !enabled {
		return
	}

	g := r.Group("/debug/pprof", guard...)
	g.GET("/", gin.WrapF(pprof.Index))
	g.GET("/cmdline", gin.WrapF(pprof.Cmdline))
	g.GET("/profile", gin.WrapF(pprof.Profile))
	g.GET("/symbol", gin.WrapF(pprof.Symbol))
	g.POST("/symbol", gin.WrapF(pprof.Symbol))
	g.GET("/trace", gin.WrapF(pprof.Trace))
	for _, name := range runtimeProfiles {
		g.GET("/"+name, gin.WrapH(pprof.Handler(name)))
	}
}
