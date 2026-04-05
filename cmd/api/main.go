package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/shared/config"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/logger"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.Log.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	tp, err := observability.InitTracer("crm-juridico")
	if err != nil {
		log.Fatal("failed to initialize tracer", zap.Error(err))
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Error("failed to shutdown tracer", zap.Error(err))
		}
	}()

	db, err := database.New(cfg.Database, log)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer database.Close(db, log)

	if err := database.RunMigrations(db, log); err != nil {
		log.Fatal("failed to run migrations", zap.Error(err))
	}

	router := setupRouter(log)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	go func() {
		log.Info("server starting", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed to start", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server forced to shutdown", zap.Error(err))
	}

	log.Info("server exited gracefully")
}

func setupRouter(log *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	tmpl := template.Must(template.ParseGlob("web/templates/**/*.html"))
	router.SetHTMLTemplate(tmpl)
	router.Static("/static", "web/static")

	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Prometheus())
	router.Use(middleware.Logger(log))

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "health/index.html", nil)
	})

	router.GET("/health", func(c *gin.Context) {
		if c.GetHeader("HX-Request") == "true" {
			c.HTML(http.StatusOK, "health/status.html", gin.H{"Status": "up"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "up"})
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return router
}
