package http

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/files/application"
	"github.com/sasrgita/crm-juridico/internal/files/domain"
	"github.com/sasrgita/crm-juridico/internal/files/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// Handler serves the HTTP endpoints of the files module. All routes are
// tenant-scoped and produce HTML fragments (HTMX) except for /download and
// /thumbnail which stream bytes.
type Handler struct {
	listUC     *application.ListFilesUseCase
	getUC      *application.GetFileUseCase
	downloadUC *application.DownloadFileUseCase
	summaryUC  *application.LeadFilesSummaryUseCase
	log        *zap.Logger
}

func NewHandler(
	listUC *application.ListFilesUseCase,
	getUC *application.GetFileUseCase,
	downloadUC *application.DownloadFileUseCase,
	summaryUC *application.LeadFilesSummaryUseCase,
	log *zap.Logger,
) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{
		listUC:     listUC,
		getUC:      getUC,
		downloadUC: downloadUC,
		summaryUC:  summaryUC,
		log:        log,
	}
}

// ListPage renders the full /tenant/files page.
func (h *Handler) ListPage(c *gin.Context) {
	ctx, span := otel.Tracer("files").Start(c.Request.Context(), "files.list_page")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	data, err := h.buildListView(c)
	if err != nil {
		h.renderError(c, err)
		return
	}
	c.HTML(http.StatusOK, "files/list.html", data)
}

// ListFragment renders the list + pagination fragment for HTMX filter updates.
func (h *Handler) ListFragment(c *gin.Context) {
	ctx, span := otel.Tracer("files").Start(c.Request.Context(), "files.list_fragment")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	data, err := h.buildListView(c)
	if err != nil {
		h.renderError(c, err)
		return
	}
	c.HTML(http.StatusOK, "files/_list_fragment.html", data)
}

// PreviewDrawer renders the sidebar drawer with file metadata + preview.
func (h *Handler) PreviewDrawer(c *gin.Context) {
	ctx, span := otel.Tracer("files").Start(c.Request.Context(), "files.preview")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	tenantID := middleware.GetTenantID(c.Request.Context())
	id := c.Param("id")

	f, err := h.getUC.Execute(c.Request.Context(), tenantID, id)
	if err != nil {
		c.HTML(h.statusFor(err), "files/_preview_drawer.html", gin.H{"Error": "Arquivo não encontrado."})
		return
	}

	c.HTML(http.StatusOK, "files/_preview_drawer.html", gin.H{
		"File": newFileView(f, ""),
	})
}

// Download streams the file bytes with the original name and correct mime.
func (h *Handler) Download(c *gin.Context) {
	ctx, span := otel.Tracer("files").Start(c.Request.Context(), "files.download")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	tenantID := middleware.GetTenantID(c.Request.Context())
	id := c.Param("id")

	f, rc, err := h.downloadUC.Execute(c.Request.Context(), tenantID, id)
	if err != nil {
		h.log.Warn("file download failed",
			zap.String("tenant_id", tenantID),
			zap.String("file_id", id),
			zap.Error(err))
		c.Status(h.statusFor(err))
		return
	}
	defer rc.Close()

	infrastructure.DownloadsTotal.WithLabelValues(string(f.MediaType)).Inc()
	h.log.Info("file downloaded",
		zap.String("tenant_id", tenantID),
		zap.String("file_id", id),
		zap.String("media_type", string(f.MediaType)))

	mime := f.MimeType
	if mime == "" {
		mime = "application/octet-stream"
	}
	c.Header("Content-Type", mime)
	c.Header("Content-Disposition", `attachment; filename*=UTF-8''`+url.PathEscape(f.Name))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, max-age=0, no-cache")
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, rc); err != nil {
		h.log.Warn("file copy failed",
			zap.String("tenant_id", tenantID),
			zap.String("file_id", id),
			zap.Error(err))
	}
}

// Thumbnail serves the raw image bytes inline (for <img src>). Only images
// respond; any other media type returns 404 to avoid serving untrusted files
// inline.
func (h *Handler) Thumbnail(c *gin.Context) {
	ctx, span := otel.Tracer("files").Start(c.Request.Context(), "files.thumbnail")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	tenantID := middleware.GetTenantID(c.Request.Context())
	id := c.Param("id")

	f, rc, err := h.downloadUC.Execute(c.Request.Context(), tenantID, id)
	if err != nil {
		c.Status(h.statusFor(err))
		return
	}
	defer rc.Close()

	if f.MediaType != domain.MediaTypeImage {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Content-Type", f.MimeType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, max-age=300")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, rc)
}

// LeadFilesSummary renders the "Arquivos (N)" section inside the lead detail
// modal. Mounted under /tenant/leads/:id/files-summary.
func (h *Handler) LeadFilesSummary(c *gin.Context) {
	ctx, span := otel.Tracer("files").Start(c.Request.Context(), "files.lead_summary")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	tenantID := middleware.GetTenantID(c.Request.Context())
	leadID := c.Param("id")

	out, err := h.summaryUC.Execute(c.Request.Context(), tenantID, leadID)
	if err != nil {
		h.log.Error("lead files summary failed",
			zap.String("tenant_id", tenantID),
			zap.String("lead_id", leadID),
			zap.Error(err))
		c.HTML(http.StatusOK, "files/_lead_summary.html", gin.H{
			"LeadID": leadID,
			"Total":  int64(0),
			"Recent": []fileView{},
		})
		return
	}

	recent := make([]fileView, 0, len(out.Recent))
	for i := range out.Recent {
		recent = append(recent, newFileView(&out.Recent[i], ""))
	}
	c.HTML(http.StatusOK, "files/_lead_summary.html", gin.H{
		"LeadID": leadID,
		"Total":  out.Total,
		"Recent": recent,
	})
}

// buildListView reads query params, calls the list use case and returns the
// data map the template expects.
func (h *Handler) buildListView(c *gin.Context) (gin.H, error) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	in := application.ListFilesInput{TenantID: tenantID}
	in.Search = strings.TrimSpace(c.Query("q"))

	if t := strings.TrimSpace(c.Query("type")); t != "" {
		mt := domain.MediaType(t)
		if !domain.IsValidMediaType(mt) {
			return nil, fmt.Errorf("invalid type: %w", domain.ErrInvalidMediaType)
		}
		in.MediaType = &mt
	}

	from, to, err := resolvePeriod(c.Query("period"), c.Query("from"), c.Query("to"))
	if err != nil {
		return nil, err
	}
	in.From = from
	in.To = to

	if lead := strings.TrimSpace(c.Query("lead_id")); lead != "" {
		in.LeadID = &lead
	}

	page := 1
	if v := strings.TrimSpace(c.Query("page")); v != "" {
		if n, errConv := strconv.Atoi(v); errConv == nil && n > 0 {
			page = n
		}
	}
	in.Page = page
	in.PageSize = domain.DefaultPageSize

	out, err := h.listUC.Execute(c.Request.Context(), in)
	if err != nil {
		return nil, err
	}

	views := make([]fileView, 0, len(out.Items))
	for i := range out.Items {
		item := out.Items[i]
		views = append(views, newFileView(&item.File, item.ContactName))
	}

	totalPages := int((out.Total + int64(out.PageSize) - 1) / int64(out.PageSize))
	if totalPages < 1 {
		totalPages = 1
	}

	return gin.H{
		"Items":      views,
		"Total":      out.Total,
		"Page":       out.Page,
		"PageSize":   out.PageSize,
		"TotalPages": totalPages,
		"Filters":    activeFilters(c),
		"Query":      c.Query("q"),
		"LeadID":     c.Query("lead_id"),
		"Type":       c.Query("type"),
		"Period":     c.Query("period"),
		"From":       c.Query("from"),
		"To":         c.Query("to"),
	}, nil
}

func (h *Handler) statusFor(err error) int {
	if err == nil {
		return http.StatusOK
	}
	// ErrFileNotFound or content unavailable are both surfaced as 404 — never
	// leak that a file of another tenant exists.
	if isNotFoundErr(err) {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

func isNotFoundErr(err error) bool {
	return strings.Contains(err.Error(), "file not found")
}

func (h *Handler) renderError(c *gin.Context, err error) {
	h.log.Warn("files handler error", zap.Error(err))
	c.HTML(http.StatusBadRequest, "files/_error.html", gin.H{"Message": err.Error()})
}

// resolvePeriod translates a preset + optional from/to pair into a concrete
// time range. Supported presets: today, 7d, 30d, custom, "" (all time).
func resolvePeriod(preset, fromStr, toStr string) (*time.Time, *time.Time, error) {
	now := time.Now()
	switch strings.TrimSpace(preset) {
	case "", "all":
		return nil, nil, nil
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return &start, nil, nil
	case "7d":
		start := now.Add(-7 * 24 * time.Hour)
		return &start, nil, nil
	case "30d":
		start := now.Add(-30 * 24 * time.Hour)
		return &start, nil, nil
	case "custom":
		var fromT, toT *time.Time
		if fromStr != "" {
			t, err := time.Parse("2006-01-02", fromStr)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid 'from' date")
			}
			fromT = &t
		}
		if toStr != "" {
			t, err := time.Parse("2006-01-02", toStr)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid 'to' date")
			}
			// include entire end day
			end := t.Add(24*time.Hour - time.Second)
			toT = &end
		}
		if fromT != nil && toT != nil && fromT.After(*toT) {
			return nil, nil, domain.ErrInvalidDateRange
		}
		return fromT, toT, nil
	default:
		return nil, nil, fmt.Errorf("unknown period")
	}
}

// activeFilters returns a slice of human labels for chips in the UI.
func activeFilters(c *gin.Context) []string {
	var chips []string
	if v := c.Query("q"); v != "" {
		chips = append(chips, `busca: "`+v+`"`)
	}
	if v := c.Query("type"); v != "" {
		chips = append(chips, "tipo: "+v)
	}
	if v := c.Query("period"); v != "" && v != "all" {
		chips = append(chips, "período: "+v)
	}
	if v := c.Query("lead_id"); v != "" {
		chips = append(chips, "lead: "+v)
	}
	return chips
}
