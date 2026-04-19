package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTenant_WithValidData_ReturnsTenant(t *testing.T) {
	tenant, err := NewTenant("uuid-1", "Escritório Silva", TenantTypePJ, "12.345.678/0001-00")

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", tenant.ID)
	assert.Equal(t, "Escritório Silva", tenant.Name)
	assert.Equal(t, TenantTypePJ, tenant.Type)
	assert.Equal(t, "12.345.678/0001-00", tenant.Document)
	assert.Equal(t, TenantStatusActive, tenant.Status)
	assert.Empty(t, tenant.BlockReason)
	assert.False(t, tenant.CreatedAt.IsZero())
	assert.False(t, tenant.UpdatedAt.IsZero())
}

func TestNewTenant_PF_ReturnsTenant(t *testing.T) {
	tenant, err := NewTenant("uuid-2", "Dr. João", TenantTypePF, "123.456.789-00")

	require.NoError(t, err)
	assert.Equal(t, TenantTypePF, tenant.Type)
}

func TestNewTenant_EmptyName_ReturnsError(t *testing.T) {
	_, err := NewTenant("uuid-1", "", TenantTypePJ, "12.345.678/0001-00")

	assert.ErrorIs(t, err, ErrTenantNameRequired)
}

func TestNewTenant_EmptyDocument_ReturnsError(t *testing.T) {
	_, err := NewTenant("uuid-1", "Escritório", TenantTypePJ, "")

	assert.ErrorIs(t, err, ErrTenantDocRequired)
}

func TestNewTenant_InvalidType_ReturnsError(t *testing.T) {
	_, err := NewTenant("uuid-1", "Escritório", TenantType("INVALID"), "12345")

	assert.ErrorIs(t, err, ErrInvalidTenantType)
}

func TestTenant_IsActive(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusActive}
	assert.True(t, tenant.IsActive())

	tenant.Status = TenantStatusBlocked
	assert.False(t, tenant.IsActive())
}

func TestTenant_IsBlocked(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusBlocked, BlockReason: "Inadimplência"}
	assert.True(t, tenant.IsBlocked())
	assert.Equal(t, "Inadimplência", tenant.BlockReason)

	tenant.Status = TenantStatusActive
	assert.False(t, tenant.IsBlocked())
}

func TestTenant_IsInactive(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusInactive}
	assert.True(t, tenant.IsInactive())

	tenant.Status = TenantStatusActive
	assert.False(t, tenant.IsInactive())
}

func TestTenant_Block_Success(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusActive}

	err := tenant.Block("Inadimplência")

	require.NoError(t, err)
	assert.Equal(t, TenantStatusBlocked, tenant.Status)
	assert.Equal(t, "Inadimplência", tenant.BlockReason)
	assert.False(t, tenant.UpdatedAt.IsZero())
}

func TestTenant_Block_EmptyReason_ReturnsError(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusActive}

	err := tenant.Block("")

	assert.ErrorIs(t, err, ErrBlockReasonRequired)
	assert.Equal(t, TenantStatusActive, tenant.Status)
}

func TestTenant_Block_AlreadyBlocked_ReturnsError(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusBlocked}

	err := tenant.Block("Outro motivo")

	assert.ErrorIs(t, err, ErrTenantAlreadyBlocked)
}

func TestTenant_Block_Inactive_ReturnsError(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusInactive}

	err := tenant.Block("Motivo")

	assert.ErrorIs(t, err, ErrTenantInactive)
}

func TestTenant_Unblock_Success(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusBlocked, BlockReason: "Inadimplência"}

	err := tenant.Unblock("Pagamento regularizado")

	require.NoError(t, err)
	assert.Equal(t, TenantStatusActive, tenant.Status)
	assert.Empty(t, tenant.BlockReason)
}

func TestTenant_Unblock_EmptyReason_ReturnsError(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusBlocked}

	err := tenant.Unblock("")

	assert.ErrorIs(t, err, ErrUnblockReasonRequired)
	assert.Equal(t, TenantStatusBlocked, tenant.Status)
}

func TestTenant_Unblock_NotBlocked_ReturnsError(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusActive}

	err := tenant.Unblock("Motivo")

	assert.ErrorIs(t, err, ErrTenantNotBlocked)
}

func TestTenant_Deactivate_Success(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusActive}

	err := tenant.Deactivate()

	require.NoError(t, err)
	assert.Equal(t, TenantStatusInactive, tenant.Status)
}

func TestTenant_Deactivate_FromBlocked_Success(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusBlocked}

	err := tenant.Deactivate()

	require.NoError(t, err)
	assert.Equal(t, TenantStatusInactive, tenant.Status)
}

func TestTenant_Deactivate_AlreadyInactive_ReturnsError(t *testing.T) {
	tenant := &Tenant{Status: TenantStatusInactive}

	err := tenant.Deactivate()

	assert.ErrorIs(t, err, ErrTenantInactive)
}

func TestTenant_Update_Success(t *testing.T) {
	tenant, _ := NewTenant("uuid-1", "Nome Antigo", TenantTypePF, "111")

	err := tenant.Update("Nome Novo", TenantTypePJ, "222")

	require.NoError(t, err)
	assert.Equal(t, "Nome Novo", tenant.Name)
	assert.Equal(t, TenantTypePJ, tenant.Type)
	assert.Equal(t, "222", tenant.Document)
}

func TestTenant_Update_EmptyName_ReturnsError(t *testing.T) {
	tenant := &Tenant{Name: "Antigo"}

	err := tenant.Update("", TenantTypePJ, "111")

	assert.ErrorIs(t, err, ErrTenantNameRequired)
	assert.Equal(t, "Antigo", tenant.Name)
}

func TestTenant_Update_EmptyDocument_ReturnsError(t *testing.T) {
	tenant := &Tenant{Document: "111"}

	err := tenant.Update("Nome", TenantTypePJ, "")

	assert.ErrorIs(t, err, ErrTenantDocRequired)
	assert.Equal(t, "111", tenant.Document)
}

func TestTenant_Update_InvalidType_ReturnsError(t *testing.T) {
	tenant := &Tenant{Type: TenantTypePF}

	err := tenant.Update("Nome", TenantType("INVALID"), "111")

	assert.ErrorIs(t, err, ErrInvalidTenantType)
	assert.Equal(t, TenantTypePF, tenant.Type)
}

func TestNewTenant_BillingDefaults(t *testing.T) {
	tn, err := NewTenant("id-1", "Nome", TenantTypePJ, "doc")
	require.NoError(t, err)
	assert.Equal(t, "mensal", tn.Plano)
	assert.True(t, tn.ExibirPagamentos)
	assert.Nil(t, tn.ValorCobrancaCents)
	assert.Nil(t, tn.DiaVencimento)
	assert.Nil(t, tn.DataInicioCobranca)
}

func TestSetBillingConfig_Mensal(t *testing.T) {
	tn, _ := NewTenant("id-1", "Nome", TenantTypePJ, "doc")
	valor := int64(50000)
	dia := uint8(10)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	tn.SetBillingConfig("mensal", &valor, &dia, &start, true)
	assert.Equal(t, "mensal", tn.Plano)
	require.NotNil(t, tn.ValorCobrancaCents)
	assert.Equal(t, int64(50000), *tn.ValorCobrancaCents)
	require.NotNil(t, tn.DiaVencimento)
	assert.Equal(t, uint8(10), *tn.DiaVencimento)
	assert.True(t, tn.ExibirPagamentos)
}

func TestSetBillingConfig_Vitalicio_KeepsExibir(t *testing.T) {
	tn, _ := NewTenant("id-1", "Nome", TenantTypePJ, "doc")
	tn.SetBillingConfig("vitalicio", nil, nil, nil, false)
	assert.Equal(t, "vitalicio", tn.Plano)
	assert.Nil(t, tn.ValorCobrancaCents)
	assert.False(t, tn.ExibirPagamentos)
}
