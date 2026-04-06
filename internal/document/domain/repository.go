package domain

import "context"

type DocumentFilter struct {
	Search string
	Page   int
	Limit  int
}

type DocumentList struct {
	Documents []Document
	Total     int64
	Page      int
	Limit     int
}

type DocumentRepository interface {
	Create(ctx context.Context, doc *Document) error
	FindByID(ctx context.Context, id string) (*Document, error)
	Delete(ctx context.Context, id string) error
	FindWithFilter(ctx context.Context, filter DocumentFilter) (*DocumentList, error)
	FindByIDs(ctx context.Context, ids []string) ([]Document, error)
}

type SpecialistDocumentRepository interface {
	Associate(ctx context.Context, specialistID, documentID string) error
	Dissociate(ctx context.Context, specialistID, documentID string) error
	DissociateAllByDocumentID(ctx context.Context, documentID string) error
	FindDocumentIDsBySpecialistID(ctx context.Context, specialistID string) ([]string, error)
	FindSpecialistIDsByDocumentID(ctx context.Context, documentID string) ([]string, error)
	Exists(ctx context.Context, specialistID, documentID string) (bool, error)
	CountByDocumentID(ctx context.Context, documentID string) (int, error)
}
