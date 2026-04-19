package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

func TestNewFile_Valid_Inbound(t *testing.T) {
	f, err := NewFile(
		"file-1", "tenant-1", "conv-1", "contact-1",
		"contrato.pdf", "application/pdf", "tenant-1/2026/04/uuid.pdf",
		1024, MediaTypeDocument, DirectionInbound,
		strPtr("lead-1"), strPtr("msg-1"),
	)
	require.NoError(t, err)
	assert.Equal(t, "file-1", f.ID)
	assert.Equal(t, "tenant-1", f.TenantID)
	assert.Equal(t, "conv-1", f.ConversationID)
	assert.Equal(t, "contact-1", f.ContactID)
	assert.Equal(t, "contrato.pdf", f.Name)
	assert.Equal(t, "application/pdf", f.MimeType)
	assert.Equal(t, int64(1024), f.SizeBytes)
	assert.Equal(t, MediaTypeDocument, f.MediaType)
	assert.Equal(t, DirectionInbound, f.Direction)
	require.NotNil(t, f.LeadID)
	assert.Equal(t, "lead-1", *f.LeadID)
	require.NotNil(t, f.MessageID)
	assert.Equal(t, "msg-1", *f.MessageID)
	assert.False(t, f.CreatedAt.IsZero())
	assert.False(t, f.UpdatedAt.IsZero())
}

func TestNewFile_Valid_NoLead_NoMessage(t *testing.T) {
	f, err := NewFile(
		"file-1", "tenant-1", "conv-1", "contact-1",
		"img.jpg", "image/jpeg", "tenant-1/2026/04/uuid.jpg",
		500, MediaTypeImage, DirectionInbound,
		nil, nil,
	)
	require.NoError(t, err)
	assert.Nil(t, f.LeadID)
	assert.Nil(t, f.MessageID)
}

func TestNewFile_EmptyTenant(t *testing.T) {
	_, err := NewFile(
		"file-1", "", "conv-1", "contact-1",
		"x.pdf", "application/pdf", "k",
		1, MediaTypeDocument, DirectionInbound, nil, nil,
	)
	assert.ErrorIs(t, err, ErrTenantIDRequired)
}

func TestNewFile_EmptyConversation(t *testing.T) {
	_, err := NewFile(
		"file-1", "tenant-1", "", "contact-1",
		"x.pdf", "application/pdf", "k",
		1, MediaTypeDocument, DirectionInbound, nil, nil,
	)
	assert.ErrorIs(t, err, ErrConversationIDRequired)
}

func TestNewFile_EmptyContact(t *testing.T) {
	_, err := NewFile(
		"file-1", "tenant-1", "conv-1", "",
		"x.pdf", "application/pdf", "k",
		1, MediaTypeDocument, DirectionInbound, nil, nil,
	)
	assert.ErrorIs(t, err, ErrContactIDRequired)
}

func TestNewFile_EmptyName(t *testing.T) {
	_, err := NewFile(
		"file-1", "tenant-1", "conv-1", "contact-1",
		"", "application/pdf", "k",
		1, MediaTypeDocument, DirectionInbound, nil, nil,
	)
	assert.ErrorIs(t, err, ErrFileNameRequired)
}

func TestNewFile_NameTooLong(t *testing.T) {
	_, err := NewFile(
		"file-1", "tenant-1", "conv-1", "contact-1",
		strings.Repeat("a", MaxFileNameLength+1), "application/pdf", "k",
		1, MediaTypeDocument, DirectionInbound, nil, nil,
	)
	assert.ErrorIs(t, err, ErrFileNameTooLong)
}

func TestNewFile_MimeTooLong(t *testing.T) {
	_, err := NewFile(
		"file-1", "tenant-1", "conv-1", "contact-1",
		"x.pdf", strings.Repeat("a", MaxMimeTypeLength+1), "k",
		1, MediaTypeDocument, DirectionInbound, nil, nil,
	)
	assert.ErrorIs(t, err, ErrMimeTypeTooLong)
}

func TestNewFile_EmptyStorageKey(t *testing.T) {
	_, err := NewFile(
		"file-1", "tenant-1", "conv-1", "contact-1",
		"x.pdf", "application/pdf", "",
		1, MediaTypeDocument, DirectionInbound, nil, nil,
	)
	assert.ErrorIs(t, err, ErrStorageKeyRequired)
}

func TestNewFile_NegativeSize(t *testing.T) {
	_, err := NewFile(
		"file-1", "tenant-1", "conv-1", "contact-1",
		"x.pdf", "application/pdf", "k",
		-1, MediaTypeDocument, DirectionInbound, nil, nil,
	)
	assert.ErrorIs(t, err, ErrNegativeSize)
}

func TestNewFile_InvalidMediaType(t *testing.T) {
	_, err := NewFile(
		"file-1", "tenant-1", "conv-1", "contact-1",
		"x.pdf", "application/pdf", "k",
		1, MediaType("bogus"), DirectionInbound, nil, nil,
	)
	assert.ErrorIs(t, err, ErrInvalidMediaType)
}

func TestNewFile_InvalidDirection(t *testing.T) {
	_, err := NewFile(
		"file-1", "tenant-1", "conv-1", "contact-1",
		"x.pdf", "application/pdf", "k",
		1, MediaTypeDocument, Direction("bogus"), nil, nil,
	)
	assert.ErrorIs(t, err, ErrInvalidDirection)
}

func TestDetectMediaType(t *testing.T) {
	cases := []struct {
		mime string
		want MediaType
	}{
		{"image/jpeg", MediaTypeImage},
		{"image/png", MediaTypeImage},
		{"image/webp", MediaTypeImage},
		{"audio/ogg", MediaTypeAudio},
		{"audio/mpeg", MediaTypeAudio},
		{"video/mp4", MediaTypeVideo},
		{"video/webm", MediaTypeVideo},
		{"application/pdf", MediaTypeDocument},
		{"application/msword", MediaTypeDocument},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", MediaTypeDocument},
		{"application/vnd.ms-excel", MediaTypeDocument},
		{"text/plain", MediaTypeDocument},
		{"text/csv", MediaTypeDocument},
		{"application/zip", MediaTypeOther},
		{"application/x-archive", MediaTypeOther},
		{"", MediaTypeOther},
	}
	for _, c := range cases {
		t.Run(c.mime, func(t *testing.T) {
			assert.Equal(t, c.want, DetectMediaType(c.mime))
		})
	}
}

func TestSanitizeFileName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"contrato.pdf", "contrato.pdf"},
		{"  contrato.pdf  ", "contrato.pdf"},
		{"../../etc/passwd", "passwd"},
		{"foo/bar/baz.txt", "baz.txt"},
		{"foo\\bar\\baz.txt", "baz.txt"},
		{"", "arquivo"},
		{"   ", "arquivo"},
		{"a\x00b.pdf", "ab.pdf"},
		{"..", "arquivo"},
		{".", "arquivo"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assert.Equal(t, c.want, SanitizeFileName(c.in))
		})
	}
}

func TestIsValidMediaType(t *testing.T) {
	assert.True(t, IsValidMediaType(MediaTypeImage))
	assert.True(t, IsValidMediaType(MediaTypeDocument))
	assert.True(t, IsValidMediaType(MediaTypeAudio))
	assert.True(t, IsValidMediaType(MediaTypeVideo))
	assert.True(t, IsValidMediaType(MediaTypeOther))
	assert.False(t, IsValidMediaType(MediaType("bogus")))
	assert.False(t, IsValidMediaType(MediaType("")))
}

func TestIsValidDirection(t *testing.T) {
	assert.True(t, IsValidDirection(DirectionInbound))
	assert.True(t, IsValidDirection(DirectionOutbound))
	assert.False(t, IsValidDirection(Direction("bogus")))
	assert.False(t, IsValidDirection(Direction("")))
}
