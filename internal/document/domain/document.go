package domain

import (
	"errors"
	"time"
)

const MaxDocumentNameLength = 255

var AllowedDocumentTypes = []string{"pdf", "txt", "docx"}

var (
	ErrDocumentNotFound          = errors.New("document not found")
	ErrDocumentNameRequired      = errors.New("document name is required")
	ErrDocumentNameTooLong       = errors.New("document name exceeds maximum length")
	ErrDocumentTypeNotAllowed    = errors.New("document type is not allowed")
	ErrDocumentFilePathRequired  = errors.New("document file path is required")
	ErrDocumentAlreadyAssociated = errors.New("document is already associated")
	ErrDocumentNotAssociated     = errors.New("document is not associated")
)

type Document struct {
	ID        string
	Name      string
	Type      string
	FilePath  string
	FileSize  int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewDocument(id, name, docType, filePath string, fileSize int64) (*Document, error) {
	if name == "" {
		return nil, ErrDocumentNameRequired
	}
	if len(name) > MaxDocumentNameLength {
		return nil, ErrDocumentNameTooLong
	}
	if !IsAllowedType(docType) {
		return nil, ErrDocumentTypeNotAllowed
	}
	if filePath == "" {
		return nil, ErrDocumentFilePathRequired
	}

	now := time.Now()
	return &Document{
		ID:        id,
		Name:      name,
		Type:      docType,
		FilePath:  filePath,
		FileSize:  fileSize,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func IsAllowedType(docType string) bool {
	for _, allowed := range AllowedDocumentTypes {
		if docType == allowed {
			return true
		}
	}
	return false
}
