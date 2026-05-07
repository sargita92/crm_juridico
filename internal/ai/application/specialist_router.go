package application

import (
	"context"
	"errors"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
)

// ErrNoSpecialistAvailable is returned when no specialist can be routed.
var ErrNoSpecialistAvailable = errors.New("no specialist available")

// PhoneNumberFinder finds the product associated with an inbound phone number.
type PhoneNumberFinder interface {
	FindProductIDByPhone(ctx context.Context, phone string) (string, error)
}

// SpecialistProductFinder finds specialists linked to a product.
type SpecialistProductFinder interface {
	FindByProductID(ctx context.Context, productID string) ([]domain.SpecialistProduct, error)
}

// ProductDetectorForRouter detects the product from the text of a message.
type ProductDetectorForRouter interface {
	DetectFromMessage(ctx context.Context, tenantID, text string) (productID string, found bool, err error)
}

// DefaultSpecialistFinder finds the fallback specialist for a tenant.
type DefaultSpecialistFinder interface {
	FindDefaultByTenantID(ctx context.Context, tenantID string) (specialistID string, err error)
}

// SpecialistRouter routes an inbound message to the appropriate specialist.
type SpecialistRouter struct {
	phoneFinder     PhoneNumberFinder
	spFinder        SpecialistProductFinder
	productDetector ProductDetectorForRouter
	defaultSpFinder DefaultSpecialistFinder
}

// NewSpecialistRouter creates a SpecialistRouter with the required dependencies.
func NewSpecialistRouter(
	phoneFinder PhoneNumberFinder,
	spFinder SpecialistProductFinder,
	productDetector ProductDetectorForRouter,
	defaultSpFinder DefaultSpecialistFinder,
) *SpecialistRouter {
	return &SpecialistRouter{
		phoneFinder:     phoneFinder,
		spFinder:        spFinder,
		productDetector: productDetector,
		defaultSpFinder: defaultSpFinder,
	}
}

// Route returns the specialistID and productID for the given context.
// Resolution order:
//  1. Look up product by phone number, pick first specialist for that product.
//  2. Detect product from message text, pick first specialist for that product.
//  3. Fall back to the tenant's default specialist (productID will be empty).
func (r *SpecialistRouter) Route(ctx context.Context, tenantID, phone, messageText string) (specialistID, productID string, err error) {
	// 1. Route by phone → product → specialist.
	if pid, phErr := r.phoneFinder.FindProductIDByPhone(ctx, phone); phErr == nil && pid != "" {
		if sps, spErr := r.spFinder.FindByProductID(ctx, pid); spErr == nil && len(sps) > 0 {
			return sps[0].SpecialistID, pid, nil
		}
	}

	// 2. Route by message text → product → specialist.
	if pid, found, detErr := r.productDetector.DetectFromMessage(ctx, tenantID, messageText); detErr == nil && found {
		if sps, spErr := r.spFinder.FindByProductID(ctx, pid); spErr == nil && len(sps) > 0 {
			return sps[0].SpecialistID, pid, nil
		}
	}

	// 3. Default specialist for tenant.
	if sid, defErr := r.defaultSpFinder.FindDefaultByTenantID(ctx, tenantID); defErr == nil && sid != "" {
		return sid, "", nil
	}

	return "", "", ErrNoSpecialistAvailable
}
