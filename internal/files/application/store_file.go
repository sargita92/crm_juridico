package application

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

type StoreFileInput struct {
	TenantID       string
	ConversationID string
	ContactID      string
	LeadID         *string // if nil, use case tries LeadLookup
	MessageID      *string
	Name           string
	MimeType       string
	Direction      domain.Direction
	Content        []byte
}

type StoreFileUseCase struct {
	repo     domain.FileRepository
	storage  domain.Storage
	lookup   domain.LeadLookup
	maxBytes int64
	now      func() time.Time
}

func NewStoreFileUseCase(
	repo domain.FileRepository,
	storage domain.Storage,
	lookup domain.LeadLookup,
	maxBytes int64,
) *StoreFileUseCase {
	if maxBytes <= 0 {
		maxBytes = 50 * 1024 * 1024 // 50 MB default
	}
	return &StoreFileUseCase{repo: repo, storage: storage, lookup: lookup, maxBytes: maxBytes, now: time.Now}
}

func (uc *StoreFileUseCase) Execute(ctx context.Context, in StoreFileInput) (*domain.File, error) {
	if int64(len(in.Content)) > uc.maxBytes {
		return nil, domain.ErrFileTooLarge
	}

	name := domain.SanitizeFileName(in.Name)
	mediaType := domain.DetectMediaType(in.MimeType)

	leadID := in.LeadID
	if leadID == nil && uc.lookup != nil && in.ConversationID != "" {
		if id, found, err := uc.lookup.FindByConversation(ctx, in.TenantID, in.ConversationID); err != nil {
			return nil, fmt.Errorf("lead lookup: %w", err)
		} else if found {
			copy := id
			leadID = &copy
		}
	}

	fileID := uuid.New().String()
	ext := extFor(name, in.MimeType)
	now := uc.now()
	storageKey := fmt.Sprintf("%s/%04d/%02d/%s%s",
		in.TenantID, now.Year(), int(now.Month()), fileID, ext,
	)

	f, err := domain.NewFile(
		fileID,
		in.TenantID, in.ConversationID, in.ContactID,
		name, strings.TrimSpace(in.MimeType), storageKey,
		int64(len(in.Content)),
		mediaType, in.Direction,
		leadID, in.MessageID,
	)
	if err != nil {
		return nil, err
	}

	if err := uc.storage.Save(ctx, storageKey, in.Content); err != nil {
		return nil, fmt.Errorf("save content: %w", err)
	}

	if err := uc.repo.Create(ctx, f); err != nil {
		return nil, fmt.Errorf("persist file: %w", err)
	}

	return f, nil
}

func extFor(name, mime string) string {
	if ext := strings.ToLower(filepath.Ext(name)); ext != "" && len(ext) <= 10 {
		return ext
	}
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "audio/ogg":
		return ".ogg"
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav":
		return ".wav"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "application/pdf":
		return ".pdf"
	}
	return ""
}
