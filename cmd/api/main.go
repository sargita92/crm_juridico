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

	authapp "github.com/sasrgita/crm-juridico/internal/auth/application"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	authhttp "github.com/sasrgita/crm-juridico/internal/auth/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/config"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/logger"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	tenantapp "github.com/sasrgita/crm-juridico/internal/tenant/application"
	tenantinfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
	tenanthttp "github.com/sasrgita/crm-juridico/internal/tenant/interfaces/http"
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

	// --- Wire-up ---

	// Tenant
	tenantRepo := tenantinfra.NewGormTenantRepository(db)

	// Auth
	userRepo := authinfra.NewGormUserRepository(db)
	userTenantRepo := authinfra.NewGormUserTenantRepository(db)
	passwordHasher := authinfra.NewBcryptHasher()
	tokenProvider := authinfra.NewJWTProvider(cfg.JWT.Secret, cfg.JWT.Expiration)

	loginUC := authapp.NewLoginUseCase(userRepo, userTenantRepo, tenantRepo, passwordHasher, tokenProvider)
	selectTenantUC := authapp.NewSelectTenantUseCase(userTenantRepo, tenantRepo, tokenProvider)
	listTenantsUC := authapp.NewListUserTenantsUseCase(userTenantRepo, tenantRepo)

	authHandler := authhttp.NewHandler(loginUC, selectTenantUC, listTenantsUC, cfg.Server.SecureCookie)

	// Tenant CRUD (admin)
	createTenantUC := tenantapp.NewCreateTenantUseCase(tenantRepo)
	listTenantsAdminUC := tenantapp.NewListTenantsUseCase(tenantRepo)
	getTenantUC := tenantapp.NewGetTenantUseCase(tenantRepo)
	updateTenantUC := tenantapp.NewUpdateTenantUseCase(tenantRepo)
	deactivateTenantUC := tenantapp.NewDeactivateTenantUseCase(tenantRepo)
	blockTenantUC := tenantapp.NewBlockTenantUseCase(tenantRepo)
	unblockTenantUC := tenantapp.NewUnblockTenantUseCase(tenantRepo)

	tenantHandler := tenanthttp.NewHandler(
		createTenantUC, listTenantsAdminUC, getTenantUC,
		updateTenantUC, deactivateTenantUC,
		blockTenantUC, unblockTenantUC,
	)

	// Middlewares
	authMw := middleware.Auth(tokenProvider)
	tenantMw := middleware.RequireTenant()
	adminMw := middleware.RequireAdmin()

	router := setupRouter(log, authHandler, tenantHandler, loginUC, authMw, tenantMw, adminMw, cfg.Server.SecureCookie)

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

func renderAdminLoginError(c *gin.Context) {
	tmpl := "admin/login.html"
	if c.GetHeader("HX-Request") == "true" {
		tmpl = "admin/login_card.html"
	}
	c.HTML(http.StatusOK, tmpl, gin.H{"Error": "Email ou senha inválidos"})
}

func setupRouter(log *zap.Logger, authHandler *authhttp.Handler, tenantHandler *tenanthttp.Handler, loginUC *authapp.LoginUseCase, authMw, tenantMw, adminMw gin.HandlerFunc, secureCookie bool) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	funcMap := template.FuncMap{
		"dict": func(values ...interface{}) map[string]interface{} {
			m := make(map[string]interface{})
			for i := 0; i+1 < len(values); i += 2 {
				key, _ := values[i].(string)
				m[key] = values[i+1]
			}
			return m
		},
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("web/templates/**/*.html"))
	router.SetHTMLTemplate(tmpl)
	router.Static("/static", "web/static")

	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Prometheus())
	router.Use(middleware.Logger(log))

	// Health routes
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

	// Auth routes
	authHandler.RegisterRoutes(router, authMw, tenantMw)

	// Admin public routes
	router.GET("/admin/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin/login.html", gin.H{"Error": ""})
	})
	router.POST("/admin/login", func(c *gin.Context) {
		email := c.PostForm("email")
		password := c.PostForm("password")

		if email == "" || password == "" {
			renderAdminLoginError(c)
			return
		}

		output, err := loginUC.Execute(c.Request.Context(), authapp.LoginInput{
			Email:    email,
			Password: password,
		})
		if err != nil {
			renderAdminLoginError(c)
			return
		}

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("token", output.Token, 86400, "/", "", secureCookie, true)
		c.Header("HX-Redirect", "/admin/dashboard")
		c.Status(http.StatusOK)
	})

	router.GET("/admin", func(c *gin.Context) {
		if t, err := c.Cookie("token"); err != nil || t == "" {
			c.Redirect(http.StatusFound, "/admin/login")
			return
		}
		c.Redirect(http.StatusFound, "/admin/dashboard")
	})

	// Admin authenticated routes
	adminGroup := router.Group("/admin")
	adminGroup.Use(authMw, adminMw)
	adminGroup.GET("/dashboard", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin/dashboard.html", gin.H{})
	})

	tenantHandler.RegisterRoutes(router, authMw, adminMw)

	return router
}
