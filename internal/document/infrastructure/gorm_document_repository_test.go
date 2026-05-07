package infrastructure

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
)

var sharedContainer *testhelper.MySQLContainer

func TestMain(m *testing.M) {
	short := false
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "-short" || strings.HasPrefix(arg, "-test.short=") || strings.HasPrefix(arg, "-short=") {
			short = true
			break
		}
	}

	if !short {
		ctx := context.Background()
		sharedContainer = testhelper.NewMySQLContainerForMain(ctx)
		code := m.Run()
		_ = sharedContainer.Container.Terminate(ctx)
		os.Exit(code)
	}

	os.Exit(m.Run())
}

func setupRepos(t *testing.T) (*GormDocumentRepository, *GormSpecialistDocumentRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	db.Exec("DELETE FROM specialist_documents")
	db.Exec("DELETE FROM documents")
	db.Exec("DELETE FROM specialist_tenants")
	db.Exec("DELETE FROM specialists")

	return NewGormDocumentRepository(db), NewGormSpecialistDocumentRepository(db), db
}

func createTestDocument(t *testing.T, repo *GormDocumentRepository, name, docType string) *domain.Document {
	t.Helper()
	doc, err := domain.NewDocument(uuid.New().String(), name, docType, "storage/documents/"+uuid.New().String()+"."+docType, 1024)
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), doc))
	return doc
}

func createTestSpecialist(t *testing.T, db *gorm.DB, name string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		"INSERT INTO specialists (id, name, description, prompt, status) VALUES (?, ?, '', 'test prompt', 'active')",
		id, name,
	).Error
	require.NoError(t, err)
	return id
}

// --- Document Repository Tests ---

func TestGormDocumentRepository_Create_And_FindByID(t *testing.T) {
	repo, _, _ := setupRepos(t)

	doc, err := domain.NewDocument(uuid.New().String(), "Legislacao Trabalhista", "pdf", "storage/documents/test.pdf", 2048)
	require.NoError(t, err)

	err = repo.Create(context.Background(), doc)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), doc.ID)
	require.NoError(t, err)
	assert.Equal(t, doc.ID, found.ID)
	assert.Equal(t, doc.Name, found.Name)
	assert.Equal(t, doc.Type, found.Type)
	assert.Equal(t, doc.FilePath, found.FilePath)
	assert.Equal(t, doc.FileSize, found.FileSize)
}

func TestGormDocumentRepository_FindByID_NotFound(t *testing.T) {
	repo, _, _ := setupRepos(t)

	_, err := repo.FindByID(context.Background(), "nonexistent-id")
	assert.ErrorIs(t, err, domain.ErrDocumentNotFound)
}

func TestGormDocumentRepository_Delete_Success(t *testing.T) {
	repo, _, _ := setupRepos(t)
	ctx := context.Background()

	doc := createTestDocument(t, repo, "Doc para excluir", "txt")

	err := repo.Delete(ctx, doc.ID)
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, doc.ID)
	assert.ErrorIs(t, err, domain.ErrDocumentNotFound)
}

func TestGormDocumentRepository_Delete_NotFound(t *testing.T) {
	repo, _, _ := setupRepos(t)

	err := repo.Delete(context.Background(), "nonexistent-id")
	assert.ErrorIs(t, err, domain.ErrDocumentNotFound)
}

func TestGormDocumentRepository_FindWithFilter_All(t *testing.T) {
	repo, _, _ := setupRepos(t)
	ctx := context.Background()

	createTestDocument(t, repo, "Doc A", "pdf")
	createTestDocument(t, repo, "Doc B", "txt")
	createTestDocument(t, repo, "Doc C", "docx")

	result, err := repo.FindWithFilter(ctx, domain.DocumentFilter{Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Total)
	assert.Len(t, result.Documents, 3)
}

func TestGormDocumentRepository_FindWithFilter_Search(t *testing.T) {
	repo, _, _ := setupRepos(t)
	ctx := context.Background()

	createTestDocument(t, repo, "Legislacao Trabalhista", "pdf")
	createTestDocument(t, repo, "Contrato Social", "pdf")
	createTestDocument(t, repo, "Legislacao Civil", "txt")

	result, err := repo.FindWithFilter(ctx, domain.DocumentFilter{Search: "Legislacao", Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.Documents, 2)
}

func TestGormDocumentRepository_FindWithFilter_Pagination(t *testing.T) {
	repo, _, _ := setupRepos(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		createTestDocument(t, repo, "Doc "+string(rune('A'+i)), "pdf")
	}

	result, err := repo.FindWithFilter(ctx, domain.DocumentFilter{Page: 1, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), result.Total)
	assert.Len(t, result.Documents, 2)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 2, result.Limit)
}

func TestGormDocumentRepository_FindWithFilter_SQLInjection(t *testing.T) {
	repo, _, _ := setupRepos(t)
	ctx := context.Background()

	createTestDocument(t, repo, "Doc Seguro", "pdf")

	result, err := repo.FindWithFilter(ctx, domain.DocumentFilter{Search: "'; DROP TABLE documents; --", Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)

	// tabela ainda existe
	result2, err := repo.FindWithFilter(ctx, domain.DocumentFilter{Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result2.Total)
}

func TestGormDocumentRepository_FindByIDs(t *testing.T) {
	repo, _, _ := setupRepos(t)
	ctx := context.Background()

	doc1 := createTestDocument(t, repo, "Doc 1", "pdf")
	doc2 := createTestDocument(t, repo, "Doc 2", "txt")
	createTestDocument(t, repo, "Doc 3", "docx")

	found, err := repo.FindByIDs(ctx, []string{doc1.ID, doc2.ID})
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestGormDocumentRepository_FindByIDs_EmptySlice(t *testing.T) {
	repo, _, _ := setupRepos(t)

	found, err := repo.FindByIDs(context.Background(), []string{})
	require.NoError(t, err)
	assert.Empty(t, found)
}

// --- SpecialistDocument Repository Tests ---

func TestGormSpecialistDocumentRepository_Associate_And_Exists(t *testing.T) {
	docRepo, specDocRepo, db := setupRepos(t)
	ctx := context.Background()

	specID := createTestSpecialist(t, db, "Especialista 1")
	doc := createTestDocument(t, docRepo, "Doc 1", "pdf")

	err := specDocRepo.Associate(ctx, specID, doc.ID)
	require.NoError(t, err)

	exists, err := specDocRepo.Exists(ctx, specID, doc.ID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestGormSpecialistDocumentRepository_Associate_Duplicate(t *testing.T) {
	docRepo, specDocRepo, db := setupRepos(t)
	ctx := context.Background()

	specID := createTestSpecialist(t, db, "Especialista 1")
	doc := createTestDocument(t, docRepo, "Doc 1", "pdf")

	require.NoError(t, specDocRepo.Associate(ctx, specID, doc.ID))

	err := specDocRepo.Associate(ctx, specID, doc.ID)
	assert.ErrorIs(t, err, domain.ErrDocumentAlreadyAssociated)
}

func TestGormSpecialistDocumentRepository_Dissociate_Success(t *testing.T) {
	docRepo, specDocRepo, db := setupRepos(t)
	ctx := context.Background()

	specID := createTestSpecialist(t, db, "Especialista 1")
	doc := createTestDocument(t, docRepo, "Doc 1", "pdf")

	require.NoError(t, specDocRepo.Associate(ctx, specID, doc.ID))

	err := specDocRepo.Dissociate(ctx, specID, doc.ID)
	require.NoError(t, err)

	exists, err := specDocRepo.Exists(ctx, specID, doc.ID)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestGormSpecialistDocumentRepository_Dissociate_NotAssociated(t *testing.T) {
	_, specDocRepo, _ := setupRepos(t)

	err := specDocRepo.Dissociate(context.Background(), "spec-1", "doc-1")
	assert.ErrorIs(t, err, domain.ErrDocumentNotAssociated)
}

func TestGormSpecialistDocumentRepository_DissociateAllByDocumentID(t *testing.T) {
	docRepo, specDocRepo, db := setupRepos(t)
	ctx := context.Background()

	spec1 := createTestSpecialist(t, db, "Especialista 1")
	spec2 := createTestSpecialist(t, db, "Especialista 2")
	doc := createTestDocument(t, docRepo, "Doc Compartilhado", "pdf")

	require.NoError(t, specDocRepo.Associate(ctx, spec1, doc.ID))
	require.NoError(t, specDocRepo.Associate(ctx, spec2, doc.ID))

	err := specDocRepo.DissociateAllByDocumentID(ctx, doc.ID)
	require.NoError(t, err)

	exists1, _ := specDocRepo.Exists(ctx, spec1, doc.ID)
	exists2, _ := specDocRepo.Exists(ctx, spec2, doc.ID)
	assert.False(t, exists1)
	assert.False(t, exists2)
}

func TestGormSpecialistDocumentRepository_FindDocumentIDsBySpecialistID(t *testing.T) {
	docRepo, specDocRepo, db := setupRepos(t)
	ctx := context.Background()

	specID := createTestSpecialist(t, db, "Especialista 1")
	doc1 := createTestDocument(t, docRepo, "Doc 1", "pdf")
	doc2 := createTestDocument(t, docRepo, "Doc 2", "txt")

	require.NoError(t, specDocRepo.Associate(ctx, specID, doc1.ID))
	require.NoError(t, specDocRepo.Associate(ctx, specID, doc2.ID))

	ids, err := specDocRepo.FindDocumentIDsBySpecialistID(ctx, specID)
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, doc1.ID)
	assert.Contains(t, ids, doc2.ID)
}

func TestGormSpecialistDocumentRepository_FindSpecialistIDsByDocumentID(t *testing.T) {
	docRepo, specDocRepo, db := setupRepos(t)
	ctx := context.Background()

	spec1 := createTestSpecialist(t, db, "Especialista 1")
	spec2 := createTestSpecialist(t, db, "Especialista 2")
	doc := createTestDocument(t, docRepo, "Doc Compartilhado", "pdf")

	require.NoError(t, specDocRepo.Associate(ctx, spec1, doc.ID))
	require.NoError(t, specDocRepo.Associate(ctx, spec2, doc.ID))

	ids, err := specDocRepo.FindSpecialistIDsByDocumentID(ctx, doc.ID)
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, spec1)
	assert.Contains(t, ids, spec2)
}

func TestGormSpecialistDocumentRepository_CountByDocumentID(t *testing.T) {
	docRepo, specDocRepo, db := setupRepos(t)
	ctx := context.Background()

	spec1 := createTestSpecialist(t, db, "Especialista 1")
	spec2 := createTestSpecialist(t, db, "Especialista 2")
	doc := createTestDocument(t, docRepo, "Doc Compartilhado", "pdf")

	require.NoError(t, specDocRepo.Associate(ctx, spec1, doc.ID))
	require.NoError(t, specDocRepo.Associate(ctx, spec2, doc.ID))

	count, err := specDocRepo.CountByDocumentID(ctx, doc.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestGormSpecialistDocumentRepository_Exists_NotAssociated(t *testing.T) {
	_, specDocRepo, _ := setupRepos(t)

	exists, err := specDocRepo.Exists(context.Background(), "spec-1", "doc-1")
	require.NoError(t, err)
	assert.False(t, exists)
}
