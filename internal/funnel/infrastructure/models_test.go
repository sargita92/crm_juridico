package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func TestLeadToModel_EmptyProductID_MapsToNil(t *testing.T) {
	lead := &domain.Lead{
		ID:             "l-1",
		TenantID:       "t-1",
		FunnelID:       "f-1",
		ColumnID:       "c-1",
		ContactID:      "ct-1",
		ConversationID: "cv-1",
		ProductID:      "", // no product detected
		Status:         domain.LeadStatusOpen,
	}

	m := leadToModel(lead)

	assert.Nil(t, m.ProductID, "empty ProductID must map to nil so GORM inserts NULL (FK constraint)")
}

func TestLeadToModel_EmptyResponsibleUserID_MapsToNil(t *testing.T) {
	lead := &domain.Lead{
		ID:                "l-1",
		TenantID:          "t-1",
		FunnelID:          "f-1",
		ColumnID:          "c-1",
		ContactID:         "ct-1",
		ConversationID:    "cv-1",
		ResponsibleUserID: "",
		Status:            domain.LeadStatusOpen,
	}

	m := leadToModel(lead)

	assert.Nil(t, m.ResponsibleUserID)
}

func TestLeadToModel_WithProductID_PointerSet(t *testing.T) {
	lead := &domain.Lead{
		ID:             "l-1",
		TenantID:       "t-1",
		FunnelID:       "f-1",
		ColumnID:       "c-1",
		ContactID:      "ct-1",
		ConversationID: "cv-1",
		ProductID:      "p-1",
		Status:         domain.LeadStatusOpen,
	}

	m := leadToModel(lead)

	if assert.NotNil(t, m.ProductID) {
		assert.Equal(t, "p-1", *m.ProductID)
	}
}

func TestMovementToModel_EmptyFromColumn_MapsToNil(t *testing.T) {
	m := &domain.LeadMovement{
		ID:           "m-1",
		LeadID:       "l-1",
		FromColumnID: "", // initial movement: lead has no previous column
		ToColumnID:   "col-entry",
	}

	out := movementToModel(m)

	assert.Nil(t, out.FromColumnID, "empty FromColumnID must map to nil so GORM inserts NULL (FK constraint)")
}

func TestMovementToModel_WithFromColumn_PointerSet(t *testing.T) {
	m := &domain.LeadMovement{
		ID:           "m-1",
		LeadID:       "l-1",
		FromColumnID: "col-a",
		ToColumnID:   "col-b",
	}

	out := movementToModel(m)

	if assert.NotNil(t, out.FromColumnID) {
		assert.Equal(t, "col-a", *out.FromColumnID)
	}
}

func TestMovementToDomain_NilFromColumn_MapsToEmptyString(t *testing.T) {
	m := &leadMovementModel{
		ID:           "m-1",
		LeadID:       "l-1",
		FromColumnID: nil,
		ToColumnID:   "col-b",
	}

	out := movementToDomain(m)

	assert.Equal(t, "", out.FromColumnID)
}

func TestLeadToDomain_NilProductID_MapsToEmptyString(t *testing.T) {
	m := &leadModel{
		ID:             "l-1",
		TenantID:       "t-1",
		FunnelID:       "f-1",
		ColumnID:       "c-1",
		ContactID:      "ct-1",
		ConversationID: "cv-1",
		ProductID:      nil,
		Status:         "open",
	}

	lead := leadToDomain(m)

	assert.Equal(t, "", lead.ProductID)
}
