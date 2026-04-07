package domain

import "time"

type NotificationType string

const (
	TypeLeadAssigned     NotificationType = "lead_assigned"
	TypeLeadMoved        NotificationType = "lead_moved"
	TypeLeadHandoff      NotificationType = "lead_handoff"
	TypeLeadQualified    NotificationType = "lead_qualified"
	TypeRateLimitReached NotificationType = "rate_limit_reached"
	TypeAutomationError  NotificationType = "automation_error"
)

const (
	MaxTitleLength = 200
	MaxBodyLength  = 1000
)

type Notification struct {
	ID        string
	TenantID  string
	UserID    string
	Type      NotificationType
	Title     string
	Body      string
	Metadata  map[string]string
	Read      bool
	CreatedAt time.Time
}

func NewNotification(id, tenantID, userID string, notifType NotificationType, title, body string, metadata map[string]string) (*Notification, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if userID == "" {
		return nil, ErrUserIDRequired
	}
	if title == "" {
		return nil, ErrTitleRequired
	}
	if len(title) > MaxTitleLength {
		return nil, ErrTitleTooLong
	}
	if len(body) > MaxBodyLength {
		return nil, ErrBodyTooLong
	}
	if metadata == nil {
		metadata = make(map[string]string)
	}
	return &Notification{
		ID:        id,
		TenantID:  tenantID,
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Body:      body,
		Metadata:  metadata,
		Read:      false,
		CreatedAt: time.Now(),
	}, nil
}

func (n *Notification) MarkRead() {
	n.Read = true
}
