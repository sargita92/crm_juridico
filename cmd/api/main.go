package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/ai"
	aiapp "github.com/sasrgita/crm-juridico/internal/ai/application"
	"github.com/sasrgita/crm-juridico/internal/audit"
	auditapp "github.com/sasrgita/crm-juridico/internal/audit/application"
	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	auditinfra "github.com/sasrgita/crm-juridico/internal/audit/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/auth"
	authapp "github.com/sasrgita/crm-juridico/internal/auth/application"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/automation"
	"github.com/sasrgita/crm-juridico/internal/dashboard"
	dashboardinfra "github.com/sasrgita/crm-juridico/internal/dashboard/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/document"
	"github.com/sasrgita/crm-juridico/internal/files"
	"github.com/sasrgita/crm-juridico/internal/funnel"
	funnelinfra "github.com/sasrgita/crm-juridico/internal/funnel/infrastructure"
	landinghttp "github.com/sasrgita/crm-juridico/internal/landing/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/mcp"
	"github.com/sasrgita/crm-juridico/internal/notification"
	notifdomain "github.com/sasrgita/crm-juridico/internal/notification/domain"
	notifhttp "github.com/sasrgita/crm-juridico/internal/notification/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/pagamentos"
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
	"github.com/sasrgita/crm-juridico/internal/shared/profiling"
	"github.com/sasrgita/crm-juridico/internal/shared/ui"
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
	defer func() { _ = log.Sync() }()

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

	// Campos de contexto anexados a cada log de slow query / erro de banco,
	// permitindo correlacionar a query lenta com o request que a originou (F26).
	dbCtxFields := func(ctx context.Context) []zap.Field {
		var f []zap.Field
		if rid := middleware.GetRequestID(ctx); rid != "" {
			f = append(f, zap.String("request_id", rid))
		}
		if tid := middleware.GetTenantID(ctx); tid != "" {
			f = append(f, zap.String("tenant_id", tid))
		}
		return f
	}

	db, err := database.New(cfg.Database, log, dbCtxFields)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer database.Close(db, log)

	// Expõe métricas do pool de conexões (sql.DBStats) em /metrics. wait_count e
	// wait_duration são a evidência direta de exaustão de pool (ver F26).
	if sqlDB, sErr := db.DB(); sErr != nil {
		log.Error("failed to get sql.DB for pool metrics", zap.Error(sErr))
	} else if rErr := observability.RegisterDBStats(prometheus.DefaultRegisterer, sqlDB, cfg.Database.Name); rErr != nil {
		log.Error("failed to register db pool metrics", zap.Error(rErr))
	}

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
	whatsappMod.SetLeadNotesService(funnelMod.LeadNotesService(whatsappMod.ConversationRepo()))

	// Files module — captures WhatsApp media into a per-tenant store and
	// exposes listing/preview/download. Wire the storer into whatsapp so
	// inbound media is persisted automatically.
	filesMod, err := files.NewModule(db, log, funnelMod.LeadLookupAdapter(), cfg.Files.StorageRoot, cfg.Files.MaxBytes)
	if err != nil {
		log.Fatal("failed to initialize files module", zap.Error(err))
	}
	whatsappMod.SetFileStorer(filesMod.FileStorer())

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

	pagLoc, locErr := time.LoadLocation(cfg.Billing.Timezone)
	if locErr != nil {
		log.Warn("billing timezone invalido, usando UTC", zap.String("tz", cfg.Billing.Timezone), zap.Error(locErr))
		pagLoc = time.UTC
	}
	pagamentosMod := pagamentos.NewModule(db, log, pagamentos.Config{
		CronSpec:  cfg.Billing.Cron,
		GraceDays: cfg.Billing.GraceDays,
		Location:  pagLoc,
	})
	pagamentosMod.SetPermissionChecker(permissionMod.Resolver())
	if err := pagamentosMod.StartScheduler(); err != nil {
		log.Fatal("start billing scheduler", zap.Error(err))
	}
	defer pagamentosMod.StopScheduler()

	dashboardMod := dashboard.NewModule(
		db,
		authMod.UserTenantRepo(),
		pagamentosMod.PaymentRepo(),
		log,
		dashboard.Config{
			ServiceChecks: []dashboardinfra.ServiceCheck{
				func(ctx context.Context) (string, bool) {
					sqlDB, err := db.DB()
					if err != nil {
						return "mysql", false
					}
					return "mysql", sqlDB.PingContext(ctx) == nil
				},
				// WhatsApp check pode ser adicionado quando whatsappMod expor IsConnected().
			},
		},
	)

	// Audit module (F12) — sits at the end of wiring so all other modules can
	// inject the publisher into their use cases. Step 8 wires /admin/logs
	// via auditMod.RegisterRoutes (apos setupRouter retornar o engine).
	auditMod := audit.NewModule(db, log)
	authMod.SetAuditPublisher(auditMod.Publisher)
	tenantMod.SetAuditPublisher(auditMod.Publisher)
	// F12 Step 7: permissao alterada de usuario admin produz audit log.
	permissionMod.SetAuditPublisher(auditMod.Publisher)

	// F12 Step 8: adapters para os dropdowns de filtro da Tela 1.
	// TenantLister usa o repo ja existente; AdminUserLister usa o gorm
	// db direto (decisao do design — view de leitura especifica do
	// painel admin de auditoria, fora do dominio de auth).
	auditMod.AttachFilters(
		auditinfra.NewTenantListerAdapter(tenantMod.TenantRepo()),
		auditinfra.NewAdminUserListerAdapter(db),
	)

	modules := []module.Module{tenantMod, specialistMod, documentMod, mcpMod, whatsappMod, funnelMod, productMod, filesMod, aiMod, permissionMod, notificationMod, automationMod, pagamentosMod, dashboardMod}

	// Token provider is used for the auth middleware and admin login route
	tokenProvider := authinfra.NewJWTProvider(cfg.JWT.Secret, cfg.JWT.Expiration)
	loginUC := authMod.LoginUseCase()
	auditPublisher := auditMod.Publisher

	// Middlewares
	authMw := middleware.Auth(tokenProvider)
	tenantMw := middleware.RequireTenant()
	adminMw := middleware.RequireAdmin()
	requirePermMw := middleware.RequirePermission(permissionMod.Resolver())
	// Sidebar UX flag: cookie ux_show_pagamentos lido por JS na sidebar.
	sidebarMw := middleware.SidebarFlags(pagamentosMod.ShowsPortalForTenant)
	// Office name UX: cookie crm_office_name lido pela sidebar/topbar do tenant.
	tenantRepo := tenantMod.TenantRepo()
	officeNameMw := middleware.OfficeName(func(ctx context.Context, tenantID string) (string, error) {
		t, err := tenantRepo.FindByID(ctx, tenantID)
		if err != nil {
			return "", err
		}
		return t.Name, nil
	})

	mw := module.Middlewares{
		Auth:              authMw,
		Tenant:            tenantMw,
		Admin:             adminMw,
		RequirePermission: requirePermMw,
	}

	// Compose Tenant middleware to also stamp sidebar flags + office name after auth/tenant resolution.
	mw.Tenant = func(c *gin.Context) {
		tenantMw(c)
		if c.IsAborted() {
			return
		}
		sidebarMw(c)
		if c.IsAborted() {
			return
		}
		officeNameMw(c)
	}

	router, tmpl := setupRouter(log, authMod, modules, loginUC, auditPublisher, tokenProvider, mw, cfg.Server.SecureCookie, cfg.AI.PlaygroundEnabled, time.Duration(cfg.Server.SlowRequestThresholdMs)*time.Millisecond)
	notificationMod.SetRenderer(notifhttp.NewToastRenderer(tmpl))

	// pprof — desabilitado por padrão; quando ligado (PPROF_ENABLED), fica atrás
	// de auth+admin. Usado para investigar gargalos fora do banco (F26).
	profiling.RegisterPprof(router, cfg.Server.PprofEnabled, authMw, adminMw)
	if cfg.Server.PprofEnabled {
		log.Warn("pprof endpoints ENABLED at /debug/pprof (admin-only)")
	}

	// F12 Step 8: rotas /admin/logs e /admin/logs/:id.
	// Registradas fora do for-modules porque o audit.Module nao
	// implementa module.Module (assinatura RegisterRoutes diferente —
	// recebe tokenProvider para o middleware AdminPageAuth especifico).
	auditMod.RegisterRoutes(router, tokenProvider)

	// Reconnect WhatsApp sessions paired in a previous run so a restart (deploy,
	// or Air hot-reload in dev) doesn't leave tenants disconnected until a manual
	// reconnect. Runs in the background to avoid delaying startup; tenants without
	// a paired session are skipped (no QR pairing is triggered).
	go func() {
		tenants, err := tenantMod.TenantRepo().FindAll(context.Background())
		if err != nil {
			log.Error("whatsapp startup reconnect: failed to list tenants", zap.Error(err))
			return
		}
		ids := make([]string, 0, len(tenants))
		for _, t := range tenants {
			ids = append(ids, t.ID)
		}
		whatsmeowProvider.ReconnectExisting(context.Background(), ids)
	}()

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

// publishAuthEvent enriquece o input com IP/UA do contexto e publica no
// audit publisher, se houver. Nil-safe (publisher pode nao estar wired em
// testes antigos).
//
// Erros sao engolidos pelo proprio publisher (decisao do design F12 secao
// 4.1: auditoria nao quebra a operacao).
func publishAuthEvent(ctx context.Context, p auditapp.Publisher, in auditapp.RegisterAuditLogInput) {
	if p == nil {
		return
	}
	in.IP = middleware.IPFromContext(ctx)
	in.UserAgent = middleware.UserAgentFromContext(ctx)
	_ = p.Publish(ctx, in)
}

// publishLoginFailure mapeia o erro do LoginUseCase para um motivo legivel
// no metadata do log de auditoria. ErrUserNotFound e tratado a parte para
// que o operador veja em /admin/logs a diferenca entre senha errada e email
// que nao existe (S1-C12 / S1-C13 dos cenarios de QA).
func publishLoginFailure(ctx context.Context, p auditapp.Publisher, email string, err error) {
	if p == nil {
		return
	}
	reason := classifyLoginErr(err)
	publishAuthEvent(ctx, p, auditapp.RegisterAuditLogInput{
		Action:     auditdomain.ActionLoginFailure,
		ActorEmail: email,
		Entity:     "session",
		Metadata:   auditdomain.Metadata{"reason": reason},
	})
}

// classifyLoginErr converte erros tipados do LoginUseCase em strings curtas
// usadas como `metadata.reason` no audit log. Nomes em PT-BR sem acento
// para alinhar com o id do enum no banco.
func classifyLoginErr(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, authdomain.ErrUserNotFound):
		return "usuario_nao_encontrado"
	case errors.Is(err, authdomain.ErrInvalidCredentials):
		return "credenciais_invalidates"
	default:
		return "erro_inesperado"
	}
}

// ptrString retorna um *string apontando para s. Pequeno helper de
// conveniencia para os campos opcionais de RegisterAuditLogInput. Nao trata
// vazio especialmente — o caller decide quando passar nil vs ponteiro de "".
func ptrString(s string) *string {
	return &s
}

func renderAdminLoginError(c *gin.Context) {
	tmpl := "admin/login.html"
	if c.GetHeader("HX-Request") == "true" {
		tmpl = "admin/login_card.html"
	}
	c.HTML(http.StatusOK, tmpl, gin.H{"Error": "Email ou senha inválidos"})
}

func setupRouter(log *zap.Logger, authMod *auth.Module, modules []module.Module, loginUC *authapp.LoginUseCase, auditPublisher auditapp.Publisher, tokenProvider authdomain.TokenProvider, mw module.Middlewares, secureCookie bool, aiPlaygroundEnabled bool, slowReqThreshold time.Duration) (*gin.Engine, *template.Template) {
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
		"columnTypeLabel":         ui.ColumnTypeLabel,
		"permissionActionLabel":   ui.PermissionActionLabel,
		"permissionResourceLabel": ui.PermissionResourceLabel,
		"relativeTime":            notifhttp.RelativeTime,
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
		"formatValor": func(c *int64) string {
			if c == nil {
				return ""
			}
			return fmt.Sprintf("%.2f", float64(*c)/100.0)
		},
		"uint8Or": func(p *uint8, fallback uint8) uint8 {
			if p == nil {
				return fallback
			}
			return *p
		},
		"dateOr": func(p *time.Time) string {
			if p == nil {
				return ""
			}
			return p.Format("2006-01-02")
		},
		// deref dereferencia *string com fallback "" para nil. Usado em
		// templates de F12 (audit) que recebem TenantID/UserID/EntityID
		// nullable da entidade.
		"deref": func(p *string) string {
			if p == nil {
				return ""
			}
			return *p
		},
		// prettyJSON converte qualquer valor para uma string JSON
		// formatada e escapada por html/template. Usado no detalhe do
		// audit log (F12 S4-C17) — substitui exibir o JSON cru de
		// `metadata` por algo legivel sem usar template.HTML (a regra de
		// seguranca proibe executar HTML cru de dado de log).
		"prettyJSON": func(v interface{}) string {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return fmt.Sprintf("%v", v)
			}
			return string(b)
		},
	}
	tmpl := template.New("").Funcs(funcMap)
	for _, pattern := range []string{
		"web/templates/*.html",
		"web/templates/*/*.html",
		"web/templates/*/*/*.html",
	} {
		matches, _ := filepath.Glob(pattern)
		if len(matches) == 0 {
			continue
		}
		tmpl = template.Must(tmpl.ParseGlob(pattern))
	}
	router.SetHTMLTemplate(tmpl)
	router.Static("/static", "web/static")

	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	// ResponseTime adiciona o header X-Response-Time e loga "slow http request"
	// acima do limiar — para flagrar a latência por request (F26).
	router.Use(middleware.ResponseTime(log, slowReqThreshold))
	// RequestMeta extrai IP/User-Agent e injeta no context — usado pelo
	// publisher de auditoria (F12) e por qualquer feature futura que precise
	// desses metadados sem reler headers.
	router.Use(middleware.RequestMeta())
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
		ctx := c.Request.Context()
		email := c.PostForm("email")
		password := c.PostForm("password")

		if email == "" || password == "" {
			renderAdminLoginError(c)
			return
		}

		output, err := loginUC.Execute(ctx, authapp.LoginInput{
			Email:    email,
			Password: password,
		})
		if err != nil {
			publishLoginFailure(ctx, auditPublisher, email, err)
			renderAdminLoginError(c)
			return
		}

		// F12: publish auth.login.success — best effort, swallowed on error.
		publishAuthEvent(ctx, auditPublisher, auditapp.RegisterAuditLogInput{
			Action:     auditdomain.ActionLoginSuccess,
			ActorEmail: email,
			UserID:     ptrString(output.UserID),
			Entity:     "session",
			EntityID:   ptrString(output.UserID),
		})

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("token", output.Token, 86400, "/", "", secureCookie, true)
		c.Header("HX-Redirect", "/admin/dashboard")
		c.Status(http.StatusOK)
	})

	// /admin/logout: limpa o cookie e publica auth.logout no audit log.
	// Existe em paralelo ao /logout do auth handler (que serve usuarios de
	// tenant) para que o evento auditavel seja registrado com a action
	// correta para a camada admin sem precisar mexer no handler do auth.
	router.POST("/admin/logout", func(c *gin.Context) {
		ctx := c.Request.Context()
		// Tenta extrair claims do token atual para enriquecer o log.
		if tokenStr, err := c.Cookie("token"); err == nil && tokenStr != "" {
			if claims, err := tokenProvider.Validate(tokenStr); err == nil && claims != nil {
				publishAuthEvent(ctx, auditPublisher, auditapp.RegisterAuditLogInput{
					Action:     auditdomain.ActionLogout,
					ActorEmail: claims.Email,
					UserID:     ptrString(claims.UserID),
					Entity:     "session",
					EntityID:   ptrString(claims.UserID),
				})
			}
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("token", "", -1, "/", "", secureCookie, true)
		c.Header("HX-Redirect", "/admin/login")
		c.Status(http.StatusOK)
	})

	router.GET("/admin", func(c *gin.Context) {
		if t, err := c.Cookie("token"); err != nil || t == "" {
			c.Redirect(http.StatusFound, "/admin/login")
			return
		}
		c.Redirect(http.StatusFound, "/admin/dashboard")
	})

	for _, mod := range modules {
		mod.RegisterRoutes(router, mw)
	}

	return router, tmpl
}
