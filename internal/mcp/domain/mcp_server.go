package domain

import (
	"errors"
	"net/url"
	"time"
)

const MaxMcpNameLength = 255
const MaxMcpConfigLength = 5000
const MaxMcpHeadersLength = 2000

var (
	ErrMcpNotFound          = errors.New("mcp server not found")
	ErrMcpNameRequired      = errors.New("mcp server name is required")
	ErrMcpNameTooLong       = errors.New("mcp server name exceeds maximum length")
	ErrMcpURLRequired       = errors.New("mcp server URL is required")
	ErrMcpURLInvalid        = errors.New("mcp server URL is invalid")
	ErrMcpConfigTooLong     = errors.New("mcp server config exceeds maximum length")
	ErrMcpHeadersTooLong    = errors.New("mcp server headers exceeds maximum length")
	ErrMcpAlreadyAssociated = errors.New("mcp server is already associated")
	ErrMcpNotAssociated     = errors.New("mcp server is not associated")
)

type McpServer struct {
	ID        string
	Name      string
	URL       string
	Config    string
	Headers   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewMcpServer(id, name, mcpURL, config, headers string) (*McpServer, error) {
	if name == "" {
		return nil, ErrMcpNameRequired
	}
	if len(name) > MaxMcpNameLength {
		return nil, ErrMcpNameTooLong
	}
	if mcpURL == "" {
		return nil, ErrMcpURLRequired
	}
	if !IsValidURL(mcpURL) {
		return nil, ErrMcpURLInvalid
	}
	if len(config) > MaxMcpConfigLength {
		return nil, ErrMcpConfigTooLong
	}
	if len(headers) > MaxMcpHeadersLength {
		return nil, ErrMcpHeadersTooLong
	}

	now := time.Now()
	return &McpServer{
		ID:        id,
		Name:      name,
		URL:       mcpURL,
		Config:    config,
		Headers:   headers,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (m *McpServer) Update(name, mcpURL, config, headers string) error {
	if name == "" {
		return ErrMcpNameRequired
	}
	if len(name) > MaxMcpNameLength {
		return ErrMcpNameTooLong
	}
	if mcpURL == "" {
		return ErrMcpURLRequired
	}
	if !IsValidURL(mcpURL) {
		return ErrMcpURLInvalid
	}
	if len(config) > MaxMcpConfigLength {
		return ErrMcpConfigTooLong
	}
	if len(headers) > MaxMcpHeadersLength {
		return ErrMcpHeadersTooLong
	}
	m.Name = name
	m.URL = mcpURL
	m.Config = config
	m.Headers = headers
	m.UpdatedAt = time.Now()
	return nil
}

func IsValidURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
