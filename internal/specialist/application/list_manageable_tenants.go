package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

// ManageableTenantItem representa um escritório no modal "Gerenciar Escritórios":
// inclui TODOS os escritórios visíveis para gestão (associados + disponíveis ativos),
// cada um com flags que dizem ao front-end como pré-marcar o checkbox e o badge.
type ManageableTenantItem struct {
	TenantID     string
	Name         string
	Type         string
	Document     string
	Status       string
	IsAssociated bool
	IsDefault    bool
}

type ListManageableTenantsUseCase struct {
	specRepo   domain.SpecialistRepository
	stRepo     domain.SpecialistTenantRepository
	tenantRepo tenantdomain.TenantRepository
}

func NewListManageableTenantsUseCase(
	specRepo domain.SpecialistRepository,
	stRepo domain.SpecialistTenantRepository,
	tenantRepo tenantdomain.TenantRepository,
) *ListManageableTenantsUseCase {
	return &ListManageableTenantsUseCase{
		specRepo:   specRepo,
		stRepo:     stRepo,
		tenantRepo: tenantRepo,
	}
}

func (uc *ListManageableTenantsUseCase) Execute(ctx context.Context, specialistID string, search string) ([]ManageableTenantItem, error) {
	if _, err := uc.specRepo.FindByID(ctx, specialistID); err != nil {
		return nil, err
	}

	associations, err := uc.stRepo.FindBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	associatedSet := make(map[string]bool, len(associations))
	defaultSet := make(map[string]bool, len(associations))
	associatedIDs := make([]string, 0, len(associations))
	for _, a := range associations {
		associatedSet[a.TenantID] = true
		if a.IsDefault {
			defaultSet[a.TenantID] = true
		}
		associatedIDs = append(associatedIDs, a.TenantID)
	}

	// Ativos (associados ou não) — passam pelo filtro de busca.
	activeList, err := uc.tenantRepo.FindWithFilter(ctx, tenantdomain.TenantFilter{
		Search: search,
		Status: tenantdomain.TenantStatusActive,
		Page:   1,
		Limit:  500,
	})
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(activeList.Tenants)+len(associatedIDs))
	items := make([]ManageableTenantItem, 0, len(activeList.Tenants)+len(associatedIDs))
	for _, t := range activeList.Tenants {
		seen[t.ID] = true
		items = append(items, ManageableTenantItem{
			TenantID:     t.ID,
			Name:         t.Name,
			Type:         string(t.Type),
			Document:     t.Document,
			Status:       string(t.Status),
			IsAssociated: associatedSet[t.ID],
			IsDefault:    defaultSet[t.ID],
		})
	}

	// Inativos associados continuam visíveis para que o admin consiga
	// desassociar pelo modal — sem isso, o usuário não tem como remover.
	missingIDs := make([]string, 0)
	for _, id := range associatedIDs {
		if !seen[id] {
			missingIDs = append(missingIDs, id)
		}
	}
	if len(missingIDs) > 0 {
		extras, err := uc.tenantRepo.FindByIDs(ctx, missingIDs)
		if err != nil {
			return nil, err
		}
		for _, t := range extras {
			items = append(items, ManageableTenantItem{
				TenantID:     t.ID,
				Name:         t.Name,
				Type:         string(t.Type),
				Document:     t.Document,
				Status:       string(t.Status),
				IsAssociated: true,
				IsDefault:    defaultSet[t.ID],
			})
		}
	}

	return items, nil
}
