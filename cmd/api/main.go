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

	"github.com/sasrgita/crm-juridico/internal/ai"
	aiapp "github.com/sasrgita/crm-juridico/internal/ai/application"
	"github.com/sasrgita/crm-juridico/internal/auth"
	authapp "github.com/sasrgita/crm-juridico/internal/auth/application"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/automation"
	"github.com/sasrgita/crm-juridico/internal/document"
	"github.com/sasrgita/crm-juridico/internal/funnel"
	funnelinfra "github.com/sasrgita/crm-juridico/internal/funnel/infrastructure"
	landinghttp "github.com/sasrgita/crm-juridico/internal/landing/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/mcp"
	"github.com/sasrgita/crm-juridico/internal/notification"
	notifdomain "github.com/sasrgita/crm-juridico/internal/notification/domain"
	notifhttp "github.com/sasrgita/crm-juridico/internal/notification/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/permission"
	perminfra "github.com/sasrgita/crm-juridico/internal/permission/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/product"
	productinfra "github.com/sasrgita/crm-juridico/internal/product/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/config"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/logger"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	"github.com/sasrgita/crm-juridico/internal/specialist"
	"github.com/sasrgita/crm-juridico/internal/tenant"
	"github.com/sasrgita/crm-juridico/internal/whatsapp"
	whatsappinfra "github.com/sasrgita/crm-juridico/internal/whatsapp/infrastructure"
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

	if cfg.Env == "production" && cfg.AI.PlaygroundEnabled {
		log.Warn("AI playground is ENABLED in production — disable AI_PLAYGROUND_ENABLED")
	}

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

	// Shared ToolRegistry — created here so both the specialist UI and AI module use the same
	// instance without introducing a circular module dependency.
	toolRegistry := aiapp.NewToolRegistry()

	// Modules
	tenantMod := tenant.NewModule(db)
	specialistMod := specialist.NewModule(db, tenantMod.TenantRepo(), toolRegistry)
	documentMod := document.NewModule(db, specialistMod.SpecialistRepo())
	mcpMod := mcp.NewModule(db, specialistMod.SpecialistRepo())

	// WhatsApp provider
	whatsmeowProvider := whatsappinfra.NewWhatsmeowProvider("storage/whatsmeow", log)
	defer whatsmeowProvider.Shutdown()

	sharedEventBus := events.NewMemoryEventBus()
	whatsappMod := whatsapp.NewModule(db, whatsmeowProvider, sharedEventBus, log)

	// Product module (must be created before funnel module for adapter wiring)
	productMod := product.NewModule(db, log)

	// Cross-module adapters
	contactAdapter := funnelinfra.NewWhatsAppContactAdapter(whatsappMod.ContactRepo())
	messageAdapter := funnelinfra.NewWhatsAppMessageAdapter(whatsappMod.MessageRepo())
	userNameAdapter := funnelinfra.NewUserNameAdapter(authinfra.NewGormUserRepository(db))
	productDetectorAdapter := funnelinfra.NewProductDetectorAdapter(productMod.DetectUseCase())
	productProviderAdapter := funnelinfra.NewProductProviderAdapter(productMod.ProductRepo())
	funnelProductRouterAdapter := funnelinfra.NewFunnelProductRouterAdapter(productMod.FunnelProductRepo())

	productListerAdapter := funnelinfra.NewProductListerAdapter(productMod.ProductRepo(), productMod.TenantProductRepo())
	// Funnel module is built with a nil picker; the real LoadBalancePicker is
	// wired below via funnelMod.SetResponsiblePicker after the permission
	// module exists (see Task 14 wiring).
	funnelMod := funnel.NewModule(db, contactAdapter, messageAdapter, userNameAdapter, productDetectorAdapter, productProviderAdapter, funnelProductRouterAdapter, productListerAdapter, sharedEventBus, nil, log)
	whatsappMod.SetLeadCreator(funnelMod.LeadCreator())

	// Cross-module: product handler needs funnel lister for link form
	funnelListerAdapter := productinfra.NewFunnelListerAdapter(funnelMod.ListFunnelsUC())
	productMod.Handler().SetFunnelLister(funnelListerAdapter)

	// Cross-module: product handler needs tenant lister for association form
	tenantListerAdapter := productinfra.NewTenantListerAdapter(tenantMod.TenantRepo())
	productMod.Handler().SetTenantLister(tenantListerAdapter)

	// Notification module (must be before automation for NotifyService)
	notificationMod := notification.NewModule(db, sharedEventBus, log)

	// Automation module — must be created before AI module so we can pass the engine.
	// (Moved up; the trigger wiring on funnelMod happens after both are created.)
	automationMod := automation.NewModule(db, automation.ModuleDeps{
		MoveLeadUC:      funnelMod.MoveLeadUC(),
		LeadRepo:        funnelMod.LeadRepo(),
		ColumnRepo:      funnelMod.ColumnRepo(),
		NoteRepo:        funnelMod.NoteRepo(),
		NotifyService:   notificationMod.NotifyService(),
		DB:              db,
		ListFunnelsUC:   funnelMod.ListFunnelsUC(),
		ContactProvider: contactAdapter,
		SpecialistRepo:  specialistMod.SpecialistRepo(),
		SpecTenantRepo:  specialistMod.SpecTenantRepo(),
	}, log)

	// AI module
	aiMod := ai.NewModule(db, cfg.AI, log, ai.ModuleDeps{
		SpecialistRepo:       specialistMod.SpecialistRepo(),
		StepRepo:             specialistMod.StepRepo(),
		GuardrailRepo:        specialistMod.GuardrailRepo(),
		SpecTenantRepo:       specialistMod.SpecTenantRepo(),
		DocRepo:              documentMod.DocRepo(),
		SpecDocRepo:          documentMod.SpecDocRepo(),
		ProductRepo:          productMod.ProductRepo(),
		PhoneNumberRepo:      productMod.PhoneNumberRepo(),
		DetectProductUC:      productMod.DetectUseCase(),
		MessageRepo:          whatsappMod.MessageRepo(),
		ConversationRepo:     whatsappMod.ConversationRepo(),
		SendMessageUC:        whatsappMod.SendMessageUC(),
		ReceiveMessageUC:     whatsappMod.ReceiveMessageUC(),
		LeadRepo:             funnelMod.LeadRepo(),
		MoveLeadUC:           funnelMod.MoveLeadUC(),
		FunnelRepo:           funnelMod.FunnelRepo(),
		ColumnRepo:           funnelMod.ColumnRepo(),
		NoteRepo:             funnelMod.NoteRepo(),
		TenantProductRepo:    productMod.TenantProductRepo(),
		AutomationEngine:     automationMod.Engine(),
		SpecialistToolFinder: specialistMod.SpecialistToolRepo(),
		ScoringConfigFinder:  specialistMod.ScoringConfigRepo(),
		ToolRegistry:         toolRegistry,
	})
	whatsappMod.SetAIHandler(aiMod)

	// Set automation trigger on funnel module
	funnelMod.SetAutomationTrigger(automationMod.Engine())

	// Auth module (login, tenant selection, invites, user management)
	authMod := auth.NewModule(db, tenantMod.TenantRepo(), cfg.JWT.Secret, cfg.JWT.Expiration, cfg.Server.SecureCookie)

	// Permission module (depends on authMod for load balance use case and funnelMod for funnels/columns)
	userRepo := authinfra.NewGormUserRepository(db)
	permissionMod := permission.NewModule(
		db, log,
		authMod.LoadBalanceUseCase(),
		funnelMod.ListFunnelsUC(),
		funnelMod.ColumnRepo(),
		authMod.ManageUsersUseCase(),
		userRepo,
	)

	// Wire cross-module permission use cases into auth's PageHandler (resolves circular dep at construction).
	authMod.AttachPermissionDeps(permissionMod.ListGroupsUseCase(), permissionMod.ManagePermissionsUseCase(), permissionMod.Resolver(), log)

	// Wire the load-balance overlap checker into auth (enforces the
	// one-active-LB-per-column invariant at ManageLoadBalanceUseCase.SetByGroup).
	// Done after both modules exist because the adapter depends on repositories
	// owned by each side.
	lbOverlapChecker := perminfra.NewGroupColumnOverlapAdapter(permissionMod.GroupFunnelRepo(), authMod.LoadBalanceRepo())
	authMod.SetLoadBalanceOverlapChecker(lbOverlapChecker)

	// Wire the LoadBalancePicker into the funnel module. The picker depends on
	// repositories owned by BOTH auth and permission modules, and the funnel
	// module depends on it transitively via CreateLeadUseCase — the cycle is
	// broken by constructing funnel with a nil picker first, then injecting the
	// real one here via SetResponsiblePicker.
	picker := authinfra.NewLoadBalancePicker(
		permissionMod.GroupFunnelRepo(),
		authMod.LoadBalanceRepo(),
		permissionMod.UserGroupRepo(),
		authMod.UserTenantRepo(),
		funnelMod.LeadLoadCounter(),
		log,
	)
	funnelMod.SetResponsiblePicker(picker)

	modules := []module.Module{tenantMod, specialistMod, documentMod, mcpMod, whatsappMod, funnelMod, productMod, aiMod, permissionMod, notificationMod, automationMod}

	// Token provider is used for the auth middleware and admin login route
	tokenProvider := authinfra.NewJWTProvider(cfg.JWT.Secret, cfg.JWT.Expiration)
	loginUC := authMod.LoginUseCase()

	// Middlewares
	authMw := middleware.Auth(tokenProvider)
	tenantMw := middleware.RequireTenant()
	adminMw := middleware.RequireAdmin()
	requirePermMw := middleware.RequirePermission(permissionMod.Resolver())

	mw := module.Middlewares{
		Auth:              authMw,
		Tenant:            tenantMw,
		Admin:             adminMw,
		RequirePermission: requirePermMw,
	}

	router, tmpl := setupRouter(log, authMod, modules, loginUC, mw, cfg.Server.SecureCookie, cfg.AI.PlaygroundEnabled)
	notificationMod.SetRenderer(notifhttp.NewToastRenderer(tmpl))

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

func setupRouter(log *zap.Logger, authMod *auth.Module, modules []module.Module, loginUC *authapp.LoginUseCase, mw module.Middlewares, secureCookie bool, aiPlaygroundEnabled bool) (*gin.Engine, *template.Template) {
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
		"aiPlaygroundEnabled": func() bool { return aiPlaygroundEnabled },
		"add":                 func(a, b int) int { return a + b },
		"sub":                 func(a, b int) int { return a - b },
		"typeIcon": func(t string) string {
			return notifhttp.TypeIcon(notifdomain.NotificationType(t))
		},
		"typeLabel": func(t string) string {
			return notifhttp.TypeLabel(notifdomain.NotificationType(t))
		},
		"relativeTime": notifhttp.RelativeTime,
		"formatFileSize": func(size int64) string {
			const (
				kb = 1024
				mb = kb * 1024
			)
			switch {
			case size >= mb:
				return fmt.Sprintf("%.1f MB", float64(size)/float64(mb))
			case size >= kb:
				return fmt.Sprintf("%.1f KB", float64(size)/float64(kb))
			default:
				return fmt.Sprintf("%d B", size)
			}
		},
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("web/templates/**/*.html"))
	router.SetHTMLTemplate(tmpl)
	router.Static("/static", "web/static")

	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Prometheus())
	router.Use(middleware.Logger(log))

	// Landing page
	landingHandler := landinghttp.NewHandler()
	landingHandler.RegisterRoutes(router)

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

	// Auth routes (login, logout, select-tenant, dashboard, invites, user management)
	authMod.RegisterRoutes(router, mw)

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
	adminGroup.Use(mw.Auth, mw.Admin)
	adminGroup.GET("/dashboard", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin/dashboard.html", gin.H{})
	})

	for _, mod := range modules {
		mod.RegisterRoutes(router, mw)
	}

	return router, tmpl
}
