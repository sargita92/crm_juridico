package domain

import (
	"strings"
	"time"
)

const MaxProductNameLength = 255

type Product struct {
	ID          string
	Name        string
	Description string
	Keywords    []string
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewProduct(id, name, description string, keywords []string) (*Product, error) {
	if name == "" {
		return nil, ErrProductNameRequired
	}
	if len(name) > MaxProductNameLength {
		return nil, ErrProductNameTooLong
	}
	if keywords == nil {
		keywords = []string{}
	}
	now := time.Now()
	return &Product{
		ID: id, Name: name,
		Description: description, Keywords: keywords,
		Active: true, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (p *Product) Update(name, description string, keywords []string) error {
	if name == "" {
		return ErrProductNameRequired
	}
	if len(name) > MaxProductNameLength {
		return ErrProductNameTooLong
	}
	if keywords == nil {
		keywords = []string{}
	}
	p.Name = name
	p.Description = description
	p.Keywords = keywords
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Product) Activate() {
	p.Active = true
	p.UpdatedAt = time.Now()
}

func (p *Product) Deactivate() {
	p.Active = false
	p.UpdatedAt = time.Now()
}

// MatchesText checks if any keyword appears in the text (case-insensitive).
func (p *Product) MatchesText(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range p.Keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}
