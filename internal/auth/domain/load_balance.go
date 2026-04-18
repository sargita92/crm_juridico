package domain

import (
	"errors"
	"time"
)

// LoadBalanceAlgorithm defines the distribution strategy for assigning leads.
type LoadBalanceAlgorithm string

const (
	AlgorithmRoundRobin LoadBalanceAlgorithm = "round_robin"
	AlgorithmLeastLoad  LoadBalanceAlgorithm = "least_load"
	AlgorithmRandom     LoadBalanceAlgorithm = "random"
)

var (
	ErrInvalidAlgorithm    = errors.New("invalid load balance algorithm")
	ErrLoadBalanceNotFound = errors.New("load balance config not found")
)

// LoadBalanceConfig holds the distribution settings for a permission group within a tenant.
type LoadBalanceConfig struct {
	ID        string
	TenantID  string
	GroupID   string
	Algorithm LoadBalanceAlgorithm
	LastIndex int
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewLoadBalanceConfig creates a validated LoadBalanceConfig.
func NewLoadBalanceConfig(id, tenantID, groupID string, algorithm LoadBalanceAlgorithm) (*LoadBalanceConfig, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if groupID == "" {
		return nil, ErrGroupIDRequired
	}
	if err := validateAlgorithm(algorithm); err != nil {
		return nil, err
	}

	now := time.Now()
	return &LoadBalanceConfig{
		ID:        id,
		TenantID:  tenantID,
		GroupID:   groupID,
		Algorithm: algorithm,
		LastIndex: 0,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// IncrementIndex advances the round-robin cursor and returns the new index.
func (c *LoadBalanceConfig) IncrementIndex() int {
	c.LastIndex++
	c.UpdatedAt = time.Now()
	return c.LastIndex
}

func validateAlgorithm(a LoadBalanceAlgorithm) error {
	switch a {
	case AlgorithmRoundRobin, AlgorithmLeastLoad, AlgorithmRandom:
		return nil
	default:
		return ErrInvalidAlgorithm
	}
}
