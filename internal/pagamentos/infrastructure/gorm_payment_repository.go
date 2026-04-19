package infrastructure

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

type GormPaymentRepository struct {
	db *gorm.DB
}

func NewGormPaymentRepository(db *gorm.DB) *GormPaymentRepository {
	return &GormPaymentRepository{db: db}
}

func (r *GormPaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	return r.db.WithContext(ctx).Create(toModel(p)).Error
}

func (r *GormPaymentRepository) Update(ctx context.Context, p *domain.Payment) error {
	return r.db.WithContext(ctx).Save(toModel(p)).Error
}

func (r *GormPaymentRepository) FindByID(ctx context.Context, tenantID, id string) (*domain.Payment, error) {
	var m paymentModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	return toDomain(&m), nil
}

func (r *GormPaymentRepository) FindByIDAdmin(ctx context.Context, id string) (*domain.Payment, error) {
	var m paymentModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	return toDomain(&m), nil
}

func (r *GormPaymentRepository) ExistsRecorrente(ctx context.Context, tenantID string, comp time.Time) (bool, error) {
	// competencia is a DATE column; the MySQL driver applies the session
	// location to both INSERT and SELECT bind values, so as long as we pass
	// the same time.Time, the stored date and the comparison align. We
	// coerce both sides to DATE to avoid DATETIME-vs-DATE comparison quirks.
	var count int64
	err := r.db.WithContext(ctx).Model(&paymentModel{}).
		Where("tenant_id = ? AND tipo = ? AND competencia = DATE(?)",
			tenantID, string(domain.TypeRecorrente), comp).
		Count(&count).Error
	return count > 0, err
}

func (r *GormPaymentRepository) List(ctx context.Context, f domain.ListFilters) (*domain.ListResult, error) {
	return r.list(ctx, f, true)
}

func (r *GormPaymentRepository) ListAll(ctx context.Context, f domain.ListFilters) (*domain.ListResult, error) {
	return r.list(ctx, f, false)
}

func (r *GormPaymentRepository) list(ctx context.Context, f domain.ListFilters, requireTenant bool) (*domain.ListResult, error) {
	page := f.Page
	if page < 1 {
		page = 1
	}
	size := f.PageSize
	if size <= 0 {
		size = domain.DefaultPageSize
	}
	if size > domain.MaxPageSize {
		size = domain.MaxPageSize
	}

	q := r.db.WithContext(ctx).Model(&paymentModel{})
	if requireTenant {
		if f.TenantID == "" {
			return nil, domain.ErrTenantIDRequired
		}
		q = q.Where("tenant_id = ?", f.TenantID)
	} else if f.TenantID != "" {
		q = q.Where("tenant_id = ?", f.TenantID)
	}
	if f.Status != nil {
		q = q.Where("status = ?", string(*f.Status))
	}
	if f.DataInicial != nil {
		q = q.Where("data_vencimento >= ?", *f.DataInicial)
	}
	if f.DataFinal != nil {
		q = q.Where("data_vencimento <= ?", *f.DataFinal)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []paymentModel
	err := q.Session(&gorm.Session{}).
		Order("data_vencimento DESC, created_at DESC").
		Limit(size).Offset((page - 1) * size).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]domain.Payment, len(rows))
	for i := range rows {
		items[i] = *toDomain(&rows[i])
	}
	return &domain.ListResult{Items: items, Total: total, Page: page, PageSize: size}, nil
}

func (r *GormPaymentRepository) ListOverdueCandidates(ctx context.Context, today time.Time) ([]domain.Payment, error) {
	var rows []paymentModel
	err := r.db.WithContext(ctx).
		Where("status = ? AND data_vencimento < ?", string(domain.StatusPendente), today).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.Payment, len(rows))
	for i := range rows {
		out[i] = *toDomain(&rows[i])
	}
	return out, nil
}

func (r *GormPaymentRepository) Summary(ctx context.Context, tenantID string, today time.Time) (*domain.Summary, error) {
	year := today.Year()
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, today.Location())
	yearEnd := time.Date(year+1, 1, 1, 0, 0, 0, 0, today.Location())

	type row struct {
		Status string
		Total  int64
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&paymentModel{}).
		Select("status, COALESCE(SUM(valor_cents), 0) AS total").
		Where("tenant_id = ?", tenantID).
		Where("(status = ? AND data_pagamento >= ? AND data_pagamento < ?) OR status IN (?, ?)",
			string(domain.StatusPago), yearStart, yearEnd,
			string(domain.StatusPendente), string(domain.StatusAtrasado)).
		Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	s := &domain.Summary{}
	for _, rr := range rows {
		switch domain.PaymentStatus(rr.Status) {
		case domain.StatusPago:
			s.TotalPagoAnoCents = rr.Total
		case domain.StatusPendente:
			s.TotalPendenteCents = rr.Total
			s.HasPendente = rr.Total > 0
		case domain.StatusAtrasado:
			s.TotalAtrasadoCents = rr.Total
			s.HasAtrasado = rr.Total > 0
		}
	}
	return s, nil
}
