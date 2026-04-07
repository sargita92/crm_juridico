package domain

import (
	"encoding/json"
	"time"
)

// AutomationType enumerates all supported automation types.
type AutomationType string

const (
	TypeExpiration       AutomationType = "expiration"
	TypeMoveFunnel       AutomationType = "move_funnel"
	TypeAutoMessage      AutomationType = "auto_message"
	TypeAutoNote         AutomationType = "auto_note"
	TypeSwitchSpecialist AutomationType = "switch_specialist"
	TypeRateLimit        AutomationType = "rate_limit"
	TypeDetectProduct    AutomationType = "detect_product"
)

var validTypes = map[AutomationType]bool{
	TypeExpiration:       true,
	TypeMoveFunnel:       true,
	TypeAutoMessage:      true,
	TypeAutoNote:         true,
	TypeSwitchSpecialist: true,
	TypeRateLimit:        true,
	TypeDetectProduct:    true,
}

// Automation is the core domain entity for an automation rule.
type Automation struct {
	ID        string
	TenantID  string
	FunnelID  string
	ColumnID  string // empty for rate_limit type
	Type      AutomationType
	Config    map[string]interface{}
	Active    bool
	Priority  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewAutomation constructs and validates an Automation entity.
func NewAutomation(id, tenantID, funnelID, columnID string, automationType AutomationType, config map[string]interface{}, priority int) (*Automation, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if funnelID == "" {
		return nil, ErrFunnelIDRequired
	}
	if !validTypes[automationType] {
		return nil, ErrInvalidType
	}
	if config == nil {
		config = map[string]interface{}{}
	}
	now := time.Now()
	return &Automation{
		ID:        id,
		TenantID:  tenantID,
		FunnelID:  funnelID,
		ColumnID:  columnID,
		Type:      automationType,
		Config:    config,
		Active:    true,
		Priority:  priority,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Update updates the mutable fields of the automation.
func (a *Automation) Update(columnID string, config map[string]interface{}, priority int) {
	a.ColumnID = columnID
	if config != nil {
		a.Config = config
	}
	a.Priority = priority
	a.UpdatedAt = time.Now()
}

// Activate marks the automation as active.
func (a *Automation) Activate() {
	a.Active = true
	a.UpdatedAt = time.Now()
}

// Deactivate marks the automation as inactive.
func (a *Automation) Deactivate() {
	a.Active = false
	a.UpdatedAt = time.Now()
}

// ConfigString returns a string value from the Config map, or empty string if absent/wrong type.
func (a *Automation) ConfigString(key string) string {
	v, ok := a.Config[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// ConfigFloat returns a float64 value from the Config map, or 0 if absent/wrong type.
func (a *Automation) ConfigFloat(key string) float64 {
	v, ok := a.Config[key]
	if !ok {
		return 0
	}
	f, _ := v.(float64)
	return f
}

// ConfigBool returns a bool value from the Config map, or false if absent/wrong type.
func (a *Automation) ConfigBool(key string) bool {
	v, ok := a.Config[key]
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

// ConfigJSON serialises the Config map to a JSON byte slice.
func (a *Automation) ConfigJSON() ([]byte, error) {
	return json.Marshal(a.Config)
}
