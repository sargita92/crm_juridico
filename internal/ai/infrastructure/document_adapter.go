package infrastructure

import (
	"context"
	"fmt"
	"os"

	docDomain "github.com/sasrgita/crm-juridico/internal/document/domain"
)

// DocumentFetcherAdapter satisfies application.DocumentFetcher.
type DocumentFetcherAdapter struct {
	docRepo     docDomain.DocumentRepository
	specDocRepo docDomain.SpecialistDocumentRepository
}

func NewDocumentFetcherAdapter(
	docRepo docDomain.DocumentRepository,
	specDocRepo docDomain.SpecialistDocumentRepository,
) *DocumentFetcherAdapter {
	return &DocumentFetcherAdapter{
		docRepo:     docRepo,
		specDocRepo: specDocRepo,
	}
}

// FetchDocumentContents retrieves the file contents of all documents associated
// with the given specialist. Errors reading individual files are skipped.
func (a *DocumentFetcherAdapter) FetchDocumentContents(ctx context.Context, specialistID string) ([]string, error) {
	ids, err := a.specDocRepo.FindDocumentIDsBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, fmt.Errorf("document_fetcher_adapter: find document ids: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}

	docs, err := a.docRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("document_fetcher_adapter: find documents: %w", err)
	}

	contents := make([]string, 0, len(docs))
	for _, doc := range docs {
		data, readErr := os.ReadFile(doc.FilePath)
		if readErr != nil {
			// Skip unreadable files rather than failing the entire request.
			continue
		}
		contents = append(contents, string(data))
	}
	return contents, nil
}
