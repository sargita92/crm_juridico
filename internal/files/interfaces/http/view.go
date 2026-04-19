package http

import (
	"strings"
	"time"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

// fileView is the shape template files consume — a flattened projection with
// display-friendly fields (pre-formatted direction label, icon, etc.).
type fileView struct {
	ID             string
	Name           string
	MediaType      string
	MimeType       string
	SizeBytes      int64
	Direction      string
	DirectionLabel string
	CreatedAt      time.Time
	RelTime        string
	LeadID         string
	ConversationID string
	MessageID      string
	ContactName    string
	Icon           string
	IsImage        bool
}

func newFileView(f *domain.File, contactName string) fileView {
	v := fileView{
		ID:             f.ID,
		Name:           f.Name,
		MediaType:      string(f.MediaType),
		MimeType:       f.MimeType,
		SizeBytes:      f.SizeBytes,
		Direction:      string(f.Direction),
		DirectionLabel: directionLabel(f.Direction),
		CreatedAt:      f.CreatedAt,
		RelTime:        relativeTime(f.CreatedAt),
		Icon:           mediaIcon(f.MediaType),
		IsImage:        f.MediaType == domain.MediaTypeImage,
		ContactName:    strings.TrimSpace(contactName),
	}
	if f.LeadID != nil {
		v.LeadID = *f.LeadID
	}
	if f.MessageID != nil {
		v.MessageID = *f.MessageID
	}
	v.ConversationID = f.ConversationID
	return v
}

func directionLabel(d domain.Direction) string {
	switch d {
	case domain.DirectionInbound:
		return "recebido"
	case domain.DirectionOutbound:
		return "enviado"
	}
	return string(d)
}

func mediaIcon(t domain.MediaType) string {
	switch t {
	case domain.MediaTypeImage:
		return "🖼"
	case domain.MediaTypeDocument:
		return "📄"
	case domain.MediaTypeAudio:
		return "🎵"
	case domain.MediaTypeVideo:
		return "🎬"
	}
	return "📦"
}

func relativeTime(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "agora"
	case diff < time.Hour:
		return formatInt(int(diff.Minutes())) + "m"
	case diff < 24*time.Hour:
		return formatInt(int(diff.Hours())) + "h"
	case diff < 30*24*time.Hour:
		return formatInt(int(diff.Hours()/24)) + "d"
	default:
		return t.Format("02/01/2006")
	}
}

func formatInt(n int) string {
	if n < 0 {
		n = 0
	}
	// small ints only — avoid strconv import
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
