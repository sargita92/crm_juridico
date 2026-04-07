package http

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildConfig_Expiration(t *testing.T) {
	form := url.Values{
		"config_action":         {"archive"},
		"config_duration_hours": {"48"},
	}
	cfg := buildConfig("expiration", form)
	assert.Equal(t, "archive", cfg["action"])
	assert.Equal(t, float64(48), cfg["duration_hours"])
}

func TestBuildConfig_MoveFunnel(t *testing.T) {
	form := url.Values{
		"config_target_funnel_id": {"funnel-123"},
		"config_target_column_id": {"col-456"},
	}
	cfg := buildConfig("move_funnel", form)
	assert.Equal(t, "funnel-123", cfg["target_funnel_id"])
	assert.Equal(t, "col-456", cfg["target_column_id"])
}

func TestBuildConfig_AutoMessage(t *testing.T) {
	form := url.Values{
		"config_template": {"Olá {{nome}}, recebemos sua mensagem"},
	}
	cfg := buildConfig("auto_message", form)
	assert.Equal(t, "Olá {{nome}}, recebemos sua mensagem", cfg["template"])
}

func TestBuildConfig_AutoNote(t *testing.T) {
	form := url.Values{
		"config_template": {"Lead qualificado automaticamente"},
	}
	cfg := buildConfig("auto_note", form)
	assert.Equal(t, "Lead qualificado automaticamente", cfg["template"])
}

func TestBuildConfig_SwitchSpecialist(t *testing.T) {
	form := url.Values{
		"config_specialist_id": {"spec-789"},
	}
	cfg := buildConfig("switch_specialist", form)
	assert.Equal(t, "spec-789", cfg["specialist_id"])
}

func TestBuildConfig_RateLimit(t *testing.T) {
	form := url.Values{
		"config_max_messages": {"50"},
		"config_period_hours": {"24"},
	}
	cfg := buildConfig("rate_limit", form)
	assert.Equal(t, float64(50), cfg["max_messages"])
	assert.Equal(t, float64(24), cfg["period_hours"])
}

func TestBuildConfig_DetectProduct(t *testing.T) {
	form := url.Values{
		"config_switch_specialist": {"true"},
	}
	cfg := buildConfig("detect_product", form)
	assert.Equal(t, true, cfg["switch_specialist"])
}

func TestBuildConfig_DetectProduct_Unchecked(t *testing.T) {
	form := url.Values{}
	cfg := buildConfig("detect_product", form)
	assert.Equal(t, false, cfg["switch_specialist"])
}
