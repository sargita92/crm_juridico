package domain

import "context"

type McpFilter struct {
	Search string
	Page   int
	Limit  int
}

type McpList struct {
	McpServers []McpServer
	Total      int64
	Page       int
	Limit      int
}

type McpServerRepository interface {
	Create(ctx context.Context, mcp *McpServer) error
	FindByID(ctx context.Context, id string) (*McpServer, error)
	Update(ctx context.Context, mcp *McpServer) error
	Delete(ctx context.Context, id string) error
	FindWithFilter(ctx context.Context, filter McpFilter) (*McpList, error)
	FindByIDs(ctx context.Context, ids []string) ([]McpServer, error)
}

type SpecialistMcpRepository interface {
	Associate(ctx context.Context, specialistID, mcpID string) error
	Dissociate(ctx context.Context, specialistID, mcpID string) error
	DissociateAllByMcpID(ctx context.Context, mcpID string) error
	FindMcpIDsBySpecialistID(ctx context.Context, specialistID string) ([]string, error)
	FindSpecialistIDsByMcpID(ctx context.Context, mcpID string) ([]string, error)
	Exists(ctx context.Context, specialistID, mcpID string) (bool, error)
	CountByMcpID(ctx context.Context, mcpID string) (int, error)
}
