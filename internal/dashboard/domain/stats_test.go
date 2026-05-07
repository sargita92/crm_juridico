package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
)

func TestTenantStats_ZeroValue(t *testing.T) {
	var s domain.TenantStats
	assert.Equal(t, float64(0), s.Bloco1_Funil.ConversionPct)
	assert.Nil(t, s.Bloco3_Responsive)
}

func TestAdminStats_ZeroValue(t *testing.T) {
	var s domain.AdminStats
	assert.Equal(t, int64(0), s.Bloco6_Financeiro.ReceitaAnoCents)
}
