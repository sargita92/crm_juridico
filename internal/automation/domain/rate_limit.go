package domain

import "time"

// RateLimitCounter tracks message volume per tenant/specialist per day.
type RateLimitCounter struct {
	ID           string
	TenantID     string
	SpecialistID string // empty means tenant-wide counter
	PeriodStart  time.Time
	MessageCount int
	CreatedAt    time.Time
}

// NewRateLimitCounter constructs a RateLimitCounter for today.
func NewRateLimitCounter(id, tenantID, specialistID string) *RateLimitCounter {
	return &RateLimitCounter{
		ID:           id,
		TenantID:     tenantID,
		SpecialistID: specialistID,
		PeriodStart:  startOfDay(time.Now()),
		MessageCount: 0,
		CreatedAt:    time.Now(),
	}
}

// Increment adds one message to the counter.
func (r *RateLimitCounter) Increment() {
	r.MessageCount++
}

// IsExceeded reports whether the counter has reached or exceeded the given maximum.
func (r *RateLimitCounter) IsExceeded(max int) bool {
	return r.MessageCount >= max
}

// startOfDay returns the midnight (00:00:00 UTC) of the given time.
func startOfDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
