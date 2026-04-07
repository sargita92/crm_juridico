package domain

import "time"

// ValidPermissions maps each resource to its allowed actions.
var ValidPermissions = map[string][]string{
	"funnels":     {"manage", "customize"},
	"leads":       {"view", "manage"},
	"automations": {"manage"},
	"users":       {"manage"},
	"groups":      {"manage"},
	"products":    {"manage"},
	"specialists": {"manage"},
	"whatsapp":    {"view", "send"},
	"settings":    {"manage"},
}

type Permission struct {
	ID        string
	TenantID  string
	GroupID   string // empty when user-level permission
	UserID    string // empty when group-level permission
	Resource  string
	Action    string
	CreatedAt time.Time
}

// NewPermission creates a Permission enforcing the XOR rule: exactly one of
// groupID or userID must be provided.
func NewPermission(id, tenantID, groupID, userID, resource, action string) (*Permission, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if resource == "" {
		return nil, ErrResourceRequired
	}
	if action == "" {
		return nil, ErrActionRequired
	}

	// XOR: exactly one must be set
	if (groupID == "") == (userID == "") {
		return nil, ErrPermissionXOR
	}

	actions, ok := ValidPermissions[resource]
	if !ok {
		return nil, ErrInvalidResource
	}
	validAction := false
	for _, a := range actions {
		if a == action {
			validAction = true
			break
		}
	}
	if !validAction {
		return nil, ErrInvalidAction
	}

	return &Permission{
		ID:        id,
		TenantID:  tenantID,
		GroupID:   groupID,
		UserID:    userID,
		Resource:  resource,
		Action:    action,
		CreatedAt: time.Now(),
	}, nil
}

// IsGroupPermission returns true when the permission belongs to a group rather
// than a specific user.
func (p *Permission) IsGroupPermission() bool {
	return p.GroupID != ""
}
