package product

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/product/application"
	"github.com/sasrgita/crm-juridico/internal/product/infrastructure"
	producthttp "github.com/sasrgita/crm-juridico/internal/product/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

type Module struct {
	handler     *producthttp.Handler
	detectUC    *application.DetectProductUseCase
	productRepo *infrastructure.GormProductRepository
	fpRepo      *infrastructure.GormFunnelProductRepository
	tpRepo      *infrastructure.GormTenantProductRepository
}

func NewModule(db *gorm.DB, log *zap.Logger) *Module {
	productRepo := infrastructure.NewGormProductRepository(db)
	fpRepo := infrastructure.NewGormFunnelProductRepository(db)
	tpRepo := infrastructure.NewGormTenantProductRepository(db)

	createProductUC := application.NewCreateProductUseCase(productRepo)
	updateProductUC := application.NewUpdateProductUseCase(productRepo)
	listProductsUC := application.NewListProductsUseCase(productRepo, fpRepo)
	listTenantProdUC := application.NewListTenantProductsUseCase(productRepo, tpRepo, fpRepo)
	toggleProductUC := application.NewToggleProductUseCase(productRepo)
	deleteProductUC := application.NewDeleteProductUseCase(productRepo)
	detectProductUC := application.NewDetectProductUseCase(productRepo, tpRepo)
	manageFPUC := application.NewManageFunnelProductsUseCase(fpRepo)
	manageTPUC := application.NewManageTenantProductsUseCase(tpRepo)

	handler := producthttp.NewHandler(
		createProductUC, updateProductUC, listProductsUC,
		listTenantProdUC, toggleProductUC, deleteProductUC,
		manageFPUC, manageTPUC, productRepo, fpRepo, log,
	)

	return &Module{
		handler:     handler,
		detectUC:    detectProductUC,
		productRepo: productRepo,
		fpRepo:      fpRepo,
		tpRepo:      tpRepo,
	}
}

func (m *Module) Name() string { return "product" }

func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant, mw.Admin)
}

func (m *Module) DetectUseCase() *application.DetectProductUseCase {
	return m.detectUC
}

func (m *Module) ProductRepo() *infrastructure.GormProductRepository {
	return m.productRepo
}

func (m *Module) FunnelProductRepo() *infrastructure.GormFunnelProductRepository {
	return m.fpRepo
}

func (m *Module) TenantProductRepo() *infrastructure.GormTenantProductRepository {
	return m.tpRepo
}

func (m *Module) Handler() *producthttp.Handler {
	return m.handler
}
