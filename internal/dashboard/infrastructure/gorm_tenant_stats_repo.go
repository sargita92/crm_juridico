package infrastructure

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
)

// GormTenantStatsRepo implementa application.TenantStatsProvider via GORM/MySQL.
type GormTenantStatsRepo struct{ db *gorm.DB }

func NewGormTenantStatsRepo(db *gorm.DB) *GormTenantStatsRepo {
	return &GormTenantStatsRepo{db: db}
}

// Compile-time check: GormTenantStatsRepo implements the application provider.
var _ application.TenantStatsProvider = (*GormTenantStatsRepo)(nil)

// FunilBlock — Bloco 1: status totals + colunas + conversão + novos hoje/semana.
// Filtro opcional por responsible_user_id quando userID != nil.
// Retorna também o nome do funil ativo (default do tenant).
func (r *GormTenantStatsRepo) FunilBlock(
	ctx context.Context,
	tenantID string,
	userID *string,
	now time.Time,
) (*domain.FunilBlock, string, error) {
	// 1) pegar funil default do tenant
	var f struct {
		ID   string
		Name string
	}
	if err := r.db.WithContext(ctx).Table("funnels").
		Select("id, name").
		Where("tenant_id = ? AND is_default = ?", tenantID, true).
		Take(&f).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &domain.FunilBlock{}, "", nil
		}
		return nil, "", err
	}
	funnelID, funnelName := f.ID, f.Name

	// 2) contadores por status
	type sRow struct {
		Status string
		Total  int64
	}
	var srows []sRow
	qStatus := r.db.WithContext(ctx).Table("leads").Where("tenant_id = ?", tenantID)
	if userID != nil {
		qStatus = qStatus.Where("responsible_user_id = ?", *userID)
	}
	if err := qStatus.Select("status, COUNT(*) AS total").Group("status").Scan(&srows).Error; err != nil {
		return nil, "", err
	}
	block := &domain.FunilBlock{}
	for _, rr := range srows {
		switch rr.Status {
		case "open":
			block.StatusTotals.Open = rr.Total
		case "won":
			block.StatusTotals.Won = rr.Total
		case "lost":
			block.StatusTotals.Lost = rr.Total
		}
	}
	totalFin := block.StatusTotals.Won + block.StatusTotals.Lost
	if totalFin > 0 {
		block.ConversionPct = float64(block.StatusTotals.Won) * 100 / float64(totalFin)
	}

	// 3) leads por coluna do funil default — JOIN funnel_columns (NÃO 'columns')
	type cRow struct {
		ColumnID   string
		Name       string
		OrderIndex int
		Total      int64
	}
	var crows []cRow
	qCol := r.db.WithContext(ctx).Table("leads AS l").
		Select("c.id AS column_id, c.name AS name, c.order_index AS order_index, COUNT(l.id) AS total").
		Joins("JOIN funnel_columns c ON c.id = l.column_id").
		Where("l.tenant_id = ? AND l.funnel_id = ?", tenantID, funnelID)
	if userID != nil {
		qCol = qCol.Where("l.responsible_user_id = ?", *userID)
	}
	if err := qCol.Group("c.id, c.name, c.order_index").Order("c.order_index").Scan(&crows).Error; err != nil {
		return nil, "", err
	}
	for _, rr := range crows {
		block.ColumnTotals = append(block.ColumnTotals, domain.ColumnLeadsCount{
			ColumnID:   rr.ColumnID,
			ColumnName: rr.Name,
			OrderIndex: rr.OrderIndex,
			Count:      rr.Total,
		})
	}

	// 4) novos hoje / novos na semana (semana iniciando no domingo, alinhada com time.Weekday)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))

	qTime := r.db.WithContext(ctx).Table("leads").Where("tenant_id = ?", tenantID)
	if userID != nil {
		qTime = qTime.Where("responsible_user_id = ?", *userID)
	}
	if err := qTime.Session(&gorm.Session{}).Where("created_at >= ?", today).Count(&block.NewToday).Error; err != nil {
		return nil, "", err
	}
	if err := qTime.Session(&gorm.Session{}).Where("created_at >= ?", weekStart).Count(&block.NewThisWeek).Error; err != nil {
		return nil, "", err
	}

	return block, funnelName, nil
}

// ProdutosBlock — Bloco 5: leads agrupados por produto, com totais won/lost.
func (r *GormTenantStatsRepo) ProdutosBlock(
	ctx context.Context,
	tenantID string,
	userID *string,
) ([]domain.ProductLeadsCount, error) {
	type pRow struct {
		ProductID string
		Name      string
		Total     int64
		Won       int64
		Lost      int64
	}
	var rows []pRow
	q := r.db.WithContext(ctx).Table("leads AS l").
		Select(`p.id AS product_id, p.name AS name,
				COUNT(l.id) AS total,
				SUM(CASE WHEN l.status='won' THEN 1 ELSE 0 END) AS won,
				SUM(CASE WHEN l.status='lost' THEN 1 ELSE 0 END) AS lost`).
		// Leads sem product_id são excluídos do bloco por construção (INNER JOIN).
		Joins("JOIN products p ON p.id = l.product_id").
		Where("l.tenant_id = ?", tenantID)
	if userID != nil {
		q = q.Where("l.responsible_user_id = ?", *userID)
	}
	if err := q.Group("p.id, p.name").Order("total DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ProductLeadsCount, 0, len(rows))
	for _, rr := range rows {
		out = append(out, domain.ProductLeadsCount{
			ProductID:   rr.ProductID,
			ProductName: rr.Name,
			Total:       rr.Total,
			Won:         rr.Won,
			Lost:        rr.Lost,
		})
	}
	return out, nil
}

// Stubs para tasks 6 (WhatsApp) e 7 (Responsaveis/TempoFunil) — serão reimplementados.
// Existem agora apenas para satisfazer a interface application.TenantStatsProvider.
func (r *GormTenantStatsRepo) WhatsAppBlock(ctx context.Context, tenantID string, userID *string) (*domain.WhatsAppStats, error) {
	return &domain.WhatsAppStats{}, nil
}

func (r *GormTenantStatsRepo) ResponsaveisBlock(ctx context.Context, tenantID string, userID *string) ([]domain.ResponsiblePerformance, error) {
	return nil, nil
}

func (r *GormTenantStatsRepo) TempoFunilBlock(ctx context.Context, tenantID string, userID *string, now time.Time) ([]domain.ColumnDwell, error) {
	return nil, nil
}
