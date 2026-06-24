package application

import (
	"context"
	"errors"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

// ErrTooManyTenants protege o sync contra payloads inflacionados — limite
// alinhado com FindWithFilter.Limit usado em ListManageableTenants.
var ErrTooManyTenants = errors.New("too many tenants in sync payload")

const syncMaxTenants = 500

type SyncSpecialistTenantsInput struct {
	SpecialistID string
	TenantIDs    []string
}

type SyncSpecialistTenantsResult struct {
	Added   int
	Removed int
}

// SyncSpecialistTenantsUseCase recebe a lista FINAL de tenants que devem ficar
// associados ao especialista e converge o estado atual para essa lista:
// calcula o diff (toAdd / toRemove) e aplica ambas as operações. Centraliza
// numa única chamada o que o modal "Gerenciar Escritórios" precisa enviar.
type SyncSpecialistTenantsUseCase struct {
	specRepo   domain.SpecialistRepository
	tenantRepo tenantdomain.TenantRepository
	stRepo     domain.SpecialistTenantRepository
}

func NewSyncSpecialistTenantsUseCase(
	specRepo domain.SpecialistRepository,
	tenantRepo tenantdomain.TenantRepository,
	stRepo domain.SpecialistTenantRepository,
) *SyncSpecialistTenantsUseCase {
	return &SyncSpecialistTenantsUseCase{
		specRepo:   specRepo,
		tenantRepo: tenantRepo,
		stRepo:     stRepo,
	}
}

func (uc *SyncSpecialistTenantsUseCase) Execute(ctx context.Context, input SyncSpecialistTenantsInput) (SyncSpecialistTenantsResult, error) {
	if len(input.TenantIDs) > syncMaxTenants {
		return SyncSpecialistTenantsResult{}, ErrTooManyTenants
	}

	specialist, err := uc.specRepo.FindByID(ctx, input.SpecialistID)
	if err != nil {
		return SyncSpecialistTenantsResult{}, err
	}
	if !specialist.IsActive() {
		return SyncSpecialistTenantsResult{}, domain.ErrSpecialistInactive
	}

	desired := make(map[string]bool, len(input.TenantIDs))
	for _, id := range input.TenantIDs {
		if id == "" {
			continue
		}
		desired[id] = true
	}

	currentIDs, err := uc.stRepo.FindTenantIDsBySpecialistID(ctx, input.SpecialistID)
	if err != nil {
		return SyncSpecialistTenantsResult{}, err
	}
	current := make(map[string]bool, len(currentIDs))
	for _, id := range currentIDs {
		current[id] = true
	}

	toAdd := make([]string, 0)
	for id := range desired {
		if !current[id] {
			toAdd = append(toAdd, id)
		}
	}
	toRemove := make([]string, 0)
	for id := range current {
		if !desired[id] {
			toRemove = append(toRemove, id)
		}
	}

	// Valida candidatos a adição antes de mutar — para que a transição
	// "add+remove" não fique parcialmente aplicada se um tenant for inválido.
	for _, id := range toAdd {
		tenant, err := uc.tenantRepo.FindByID(ctx, id)
		if err != nil {
			return SyncSpecialistTenantsResult{}, err
		}
		if !tenant.IsActive() {
			return SyncSpecialistTenantsResult{}, tenantdomain.ErrTenantInactive
		}
	}

	for _, id := range toRemove {
		if err := uc.stRepo.Dissociate(ctx, input.SpecialistID, id); err != nil {
			return SyncSpecialistTenantsResult{}, err
		}
	}
	for _, id := range toAdd {
		if err := uc.stRepo.Associate(ctx, input.SpecialistID, id); err != nil {
			return SyncSpecialistTenantsResult{}, err
		}
	}

	return SyncSpecialistTenantsResult{
		Added:   len(toAdd),
		Removed: len(toRemove),
	}, nil
}
