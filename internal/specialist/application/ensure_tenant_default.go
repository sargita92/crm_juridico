package application

import (
	"context"
	"errors"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// ensureTenantDefault promotes a tenant's sole associated specialist to be its
// default when the tenant has no default yet. Routing only resolves a specialist
// via the is_default flag (or a product/phone link), so a tenant that has exactly
// one associated specialist but no default would be unroutable — a surprising gap
// for admins who "just associate" without touching the "Tornar Default" toggle.
//
// It is intentionally conservative: it never overrides an existing default and,
// with two or more associated specialists and no default, it does not guess —
// disambiguation is left to the explicit toggle.
func ensureTenantDefault(ctx context.Context, stRepo domain.SpecialistTenantRepository, tenantID string) error {
	if _, err := stRepo.FindDefaultByTenantID(ctx, tenantID); err == nil {
		return nil // a default already exists — leave it untouched
	} else if !errors.Is(err, domain.ErrSpecialistNotFound) {
		return err
	}

	ids, err := stRepo.FindSpecialistIDsByTenantID(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(ids) == 1 {
		return stRepo.SetDefault(ctx, ids[0], tenantID)
	}
	return nil
}
