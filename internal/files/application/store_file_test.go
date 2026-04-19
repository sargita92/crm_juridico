package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

func newStoreUC(t *testing.T) (*StoreFileUseCase, *mockFileRepo, *mockStorage, *mockLeadLookup) {
	t.Helper()
	repo := newMockFileRepo()
	st := newMockStorage()
	lookup := newMockLeadLookup()
	uc := NewStoreFileUseCase(repo, st, lookup, 50*1024*1024)
	return uc, repo, st, lookup
}

func TestStoreFileUseCase_PersistsMetadataAndContent(t *testing.T) {
	uc, repo, st, _ := newStoreUC(t)

	f, err := uc.Execute(context.Background(), StoreFileInput{
		TenantID:       "t1",
		ConversationID: "c1",
		ContactID:      "k1",
		Name:           "contrato.pdf",
		MimeType:       "application/pdf",
		Direction:      domain.DirectionInbound,
		Content:        []byte("hello"),
	})
	require.NoError(t, err)
	assert.Equal(t, "contrato.pdf", f.Name)
	assert.Equal(t, domain.MediaTypeDocument, f.MediaType)
	assert.Equal(t, int64(5), f.SizeBytes)

	// Stored in repo
	saved, _ := repo.FindByID(context.Background(), "t1", f.ID)
	require.NotNil(t, saved)

	// Bytes in storage under the generated key
	assert.Contains(t, f.StorageKey, "t1/")
	assert.True(t, strings.HasSuffix(f.StorageKey, ".pdf"), "expected .pdf suffix")
	rc, err := st.Open(context.Background(), f.StorageKey)
	require.NoError(t, err)
	defer rc.Close()
}

func TestStoreFileUseCase_ResolvesLeadViaLookup(t *testing.T) {
	uc, repo, _, lookup := newStoreUC(t)
	lookup.set("t1", "c1", "lead-xyz")

	f, err := uc.Execute(context.Background(), StoreFileInput{
		TenantID:       "t1",
		ConversationID: "c1",
		ContactID:      "k1",
		Name:           "img.jpg",
		MimeType:       "image/jpeg",
		Direction:      domain.DirectionInbound,
		Content:        []byte("bytes"),
	})
	require.NoError(t, err)
	require.NotNil(t, f.LeadID)
	assert.Equal(t, "lead-xyz", *f.LeadID)
	// Saved copy also has lead id
	saved, _ := repo.FindByID(context.Background(), "t1", f.ID)
	require.NotNil(t, saved.LeadID)
}

func TestStoreFileUseCase_MissingLeadIsAcceptable(t *testing.T) {
	uc, _, _, _ := newStoreUC(t)
	f, err := uc.Execute(context.Background(), StoreFileInput{
		TenantID:       "t1",
		ConversationID: "c1",
		ContactID:      "k1",
		Name:           "img.jpg",
		MimeType:       "image/jpeg",
		Direction:      domain.DirectionInbound,
		Content:        []byte("x"),
	})
	require.NoError(t, err)
	assert.Nil(t, f.LeadID)
}

func TestStoreFileUseCase_ExplicitLeadOverridesLookup(t *testing.T) {
	uc, _, _, lookup := newStoreUC(t)
	lookup.set("t1", "c1", "from-lookup")
	explicit := "from-caller"

	f, err := uc.Execute(context.Background(), StoreFileInput{
		TenantID:       "t1",
		ConversationID: "c1",
		ContactID:      "k1",
		LeadID:         &explicit,
		Name:           "x.pdf",
		MimeType:       "application/pdf",
		Direction:      domain.DirectionInbound,
		Content:        []byte("x"),
	})
	require.NoError(t, err)
	require.NotNil(t, f.LeadID)
	assert.Equal(t, "from-caller", *f.LeadID)
}

func TestStoreFileUseCase_RejectsOversizedContent(t *testing.T) {
	uc, _, _, _ := newStoreUC(t)
	uc.maxBytes = 10
	_, err := uc.Execute(context.Background(), StoreFileInput{
		TenantID:       "t1",
		ConversationID: "c1",
		ContactID:      "k1",
		Name:           "big.pdf",
		MimeType:       "application/pdf",
		Direction:      domain.DirectionInbound,
		Content:        []byte("12345678901234567890"),
	})
	assert.ErrorIs(t, err, domain.ErrFileTooLarge)
}

func TestStoreFileUseCase_SanitizesName(t *testing.T) {
	uc, _, _, _ := newStoreUC(t)
	f, err := uc.Execute(context.Background(), StoreFileInput{
		TenantID:       "t1",
		ConversationID: "c1",
		ContactID:      "k1",
		Name:           "../../etc/passwd",
		MimeType:       "application/octet-stream",
		Direction:      domain.DirectionInbound,
		Content:        []byte("x"),
	})
	require.NoError(t, err)
	assert.Equal(t, "passwd", f.Name)
	assert.NotContains(t, f.StorageKey, "..")
}

func TestStoreFileUseCase_ClassifiesMediaTypeFromMime(t *testing.T) {
	uc, _, _, _ := newStoreUC(t)

	img, err := uc.Execute(context.Background(), StoreFileInput{
		TenantID: "t1", ConversationID: "c1", ContactID: "k1",
		Name: "x", MimeType: "image/png", Direction: domain.DirectionInbound, Content: []byte("x"),
	})
	require.NoError(t, err)
	assert.Equal(t, domain.MediaTypeImage, img.MediaType)

	aud, _ := uc.Execute(context.Background(), StoreFileInput{
		TenantID: "t1", ConversationID: "c1", ContactID: "k1",
		Name: "x", MimeType: "audio/ogg", Direction: domain.DirectionInbound, Content: []byte("x"),
	})
	assert.Equal(t, domain.MediaTypeAudio, aud.MediaType)

	oth, _ := uc.Execute(context.Background(), StoreFileInput{
		TenantID: "t1", ConversationID: "c1", ContactID: "k1",
		Name: "x", MimeType: "application/x-tar", Direction: domain.DirectionInbound, Content: []byte("x"),
	})
	assert.Equal(t, domain.MediaTypeOther, oth.MediaType)
}

func TestStoreFileUseCase_StorageErrorPropagated(t *testing.T) {
	uc, _, st, _ := newStoreUC(t)
	st.saveErr = errBoom
	_, err := uc.Execute(context.Background(), StoreFileInput{
		TenantID: "t1", ConversationID: "c1", ContactID: "k1",
		Name: "x", MimeType: "application/pdf", Direction: domain.DirectionInbound, Content: []byte("x"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestStoreFileUseCase_RepoErrorPropagated(t *testing.T) {
	uc, repo, _, _ := newStoreUC(t)
	repo.createErr = errBoom
	_, err := uc.Execute(context.Background(), StoreFileInput{
		TenantID: "t1", ConversationID: "c1", ContactID: "k1",
		Name: "x", MimeType: "application/pdf", Direction: domain.DirectionInbound, Content: []byte("x"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestStoreFileUseCase_LookupErrorPropagated(t *testing.T) {
	uc, _, _, lookup := newStoreUC(t)
	lookup.err = errBoom
	_, err := uc.Execute(context.Background(), StoreFileInput{
		TenantID: "t1", ConversationID: "c1", ContactID: "k1",
		Name: "x", MimeType: "application/pdf", Direction: domain.DirectionInbound, Content: []byte("x"),
	})
	require.Error(t, err)
}

func TestStoreFileUseCase_StorageKeyYearMonthScoping(t *testing.T) {
	repo := newMockFileRepo()
	st := newMockStorage()
	lookup := newMockLeadLookup()
	uc := NewStoreFileUseCase(repo, st, lookup, 0)
	uc.now = func() time.Time { return time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC) }

	f, err := uc.Execute(context.Background(), StoreFileInput{
		TenantID: "t1", ConversationID: "c1", ContactID: "k1",
		Name: "x.pdf", MimeType: "application/pdf", Direction: domain.DirectionInbound, Content: []byte("x"),
	})
	require.NoError(t, err)
	assert.Contains(t, f.StorageKey, "t1/2026/03/")
}

func TestStoreFileUseCase_InvalidDirectionSurfacesDomainError(t *testing.T) {
	uc, _, _, _ := newStoreUC(t)
	_, err := uc.Execute(context.Background(), StoreFileInput{
		TenantID: "t1", ConversationID: "c1", ContactID: "k1",
		Name: "x", MimeType: "application/pdf", Direction: domain.Direction("bogus"), Content: []byte("x"),
	})
	assert.ErrorIs(t, err, domain.ErrInvalidDirection)
}
