package domain

import "errors"

var (
	ErrTenantIDRequired    = errors.New("tenant ID is required")
	ErrProductNotFound     = errors.New("product not found")
	ErrProductNameRequired = errors.New("product name is required")
	ErrProductNameTooLong  = errors.New("product name exceeds maximum length")

	ErrFunnelProductNotFound      = errors.New("funnel-product link not found")
	ErrFunnelProductAlreadyExists = errors.New("funnel-product link already exists")

	ErrTenantProductAlreadyExists = errors.New("tenant-product association already exists")
	ErrTenantProductNotFound      = errors.New("tenant-product association not found")
)
