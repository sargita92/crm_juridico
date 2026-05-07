package http

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/document/application"
	"github.com/sasrgita/crm-juridico/internal/document/domain"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

const storageDir = "storage/documents"
const maxFileSize = 10 * 1024 * 1024 // 10MB

func validUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

type Handler struct {
	uploadUC        *application.UploadDocumentUseCase
	listUC          *application.ListDocumentsUseCase
	getUC           *application.GetDocumentUseCase
	deleteUC        *application.DeleteDocumentUseCase
	associateUC     *application.AssociateDocumentUseCase
	dissociateUC    *application.DissociateDocumentUseCase
	listSpecDocsUC  *application.ListSpecialistDocumentsUseCase
	listAvailableUC *application.ListAvailableDocumentsUseCase
}

func NewHandler(
	uploadUC *application.UploadDocumentUseCase,
	listUC *application.ListDocumentsUseCase,
	getUC *application.GetDocumentUseCase,
	deleteUC *application.DeleteDocumentUseCase,
	associateUC *application.AssociateDocumentUseCase,
	dissociateUC *application.DissociateDocumentUseCase,
	listSpecDocsUC *application.ListSpecialistDocumentsUseCase,
	listAvailableUC *application.ListAvailableDocumentsUseCase,
) *Handler {
	return &Handler{
		uploadUC:        uploadUC,
		listUC:          listUC,
		getUC:           getUC,
		deleteUC:        deleteUC,
		associateUC:     associateUC,
		dissociateUC:    dissociateUC,
		listSpecDocsUC:  listSpecDocsUC,
		listAvailableUC: listAvailableUC,
	}
}

func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	// Document CRUD routes
	admin := router.Group("/admin/documents")
	admin.Use(authMw, adminMw)

	admin.GET("", h.RenderList)
	admin.POST("", h.HandleUpload)
	admin.GET("/:id", h.RenderDetail)
	admin.DELETE("/:id", h.HandleDelete)

	// Specialist-document association routes
	specAdmin := router.Group("/admin/specialists")
	specAdmin.Use(authMw, adminMw)

	specAdmin.GET("/:id/documents", h.HandleListSpecialistDocuments)
	specAdmin.GET("/:id/documents/available", h.HandleListAvailableDocuments)
	specAdmin.POST("/:id/documents", h.HandleAssociateDocuments)
	specAdmin.DELETE("/:id/documents/:doc_id", h.HandleDissociateDocument)
}

func (h *Handler) RenderList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	input := application.ListDocumentsInput{
		Search: c.Query("search"),
		Page:   page,
		Limit:  limit,
	}

	output, err := h.listUC.Execute(c.Request.Context(), input)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "document/list.html", gin.H{"Error": "Erro ao carregar documentos"})
		return
	}

	data := gin.H{
		"Documents": output.Documents,
		"Total":     output.Total,
		"Page":      output.Page,
		"Limit":     output.Limit,
		"Search":    input.Search,
		"HasPrev":   output.Page > 1,
		"HasNext":   int64(output.Page*output.Limit) < output.Total,
		"PrevPage":  output.Page - 1,
		"NextPage":  output.Page + 1,
		"StartItem": (output.Page-1)*output.Limit + 1,
		"EndItem":   min(int64(output.Page*output.Limit), output.Total),
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "document/table.html", data)
		return
	}
	c.HTML(http.StatusOK, "document/list.html", data)
}

func (h *Handler) HandleUpload(c *gin.Context) {
	name := c.PostForm("name")
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.HTML(http.StatusBadRequest, "document/upload_modal.html", gin.H{"Error": "Arquivo é obrigatório"})
		return
	}
	defer func() { _ = file.Close() }()

	if header.Size > maxFileSize {
		c.HTML(http.StatusBadRequest, "document/upload_modal.html", gin.H{
			"Error": "Arquivo não pode exceder 10MB",
			"Name":  name,
		})
		return
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(header.Filename)), ".")
	if !domain.IsAllowedType(ext) {
		c.HTML(http.StatusBadRequest, "document/upload_modal.html", gin.H{
			"Error": "Tipo de arquivo não permitido. Use PDF, TXT ou DOCX",
			"Name":  name,
		})
		return
	}

	fileID := uuid.New().String()
	fileName := fileID + "." + ext
	filePath := filepath.Join(storageDir, fileName)

	if err := os.MkdirAll(storageDir, 0755); err != nil {
		c.HTML(http.StatusInternalServerError, "document/upload_modal.html", gin.H{"Error": "Erro ao salvar arquivo"})
		return
	}

	dst, err := os.Create(filePath)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "document/upload_modal.html", gin.H{"Error": "Erro ao salvar arquivo"})
		return
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(filePath)
		c.HTML(http.StatusInternalServerError, "document/upload_modal.html", gin.H{"Error": "Erro ao salvar arquivo"})
		return
	}

	output, err := h.uploadUC.Execute(c.Request.Context(), application.UploadDocumentInput{
		Name:     name,
		Type:     ext,
		FilePath: filePath,
		FileSize: header.Size,
	})
	if err != nil {
		_ = os.Remove(filePath)
		c.HTML(http.StatusBadRequest, "document/upload_modal.html", gin.H{
			"Error": mapDomainError(err),
			"Name":  name,
		})
		return
	}

	c.Header("HX-Redirect", "/admin/documents/"+output.ID)
	c.Status(http.StatusOK)
}

func (h *Handler) RenderDetail(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.HTML(http.StatusBadRequest, "document/detail.html", gin.H{"Error": "ID inválido"})
		return
	}

	output, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			c.HTML(http.StatusNotFound, "document/detail.html", gin.H{"Error": "Documento não encontrado"})
			return
		}
		c.HTML(http.StatusInternalServerError, "document/detail.html", gin.H{"Error": "Erro ao carregar documento"})
		return
	}

	c.HTML(http.StatusOK, "document/detail.html", gin.H{
		"Document": output,
		"Error":    "",
	})
}

func (h *Handler) HandleDelete(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	doc, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Documento não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao excluir documento"})
		return
	}

	if err := h.deleteUC.Execute(c.Request.Context(), id); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao excluir documento"})
		return
	}

	// Remove file from storage (best effort)
	_ = os.Remove(doc.FilePath)

	c.Header("HX-Redirect", "/admin/documents")
	c.Status(http.StatusOK)
}

// --- Specialist-Document association handlers ---

func (h *Handler) HandleListSpecialistDocuments(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	items, err := h.listSpecDocsUC.Execute(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar documentos"})
		return
	}

	c.HTML(http.StatusOK, "specialist/documents_section.html", gin.H{
		"Documents":    items,
		"SpecialistID": id,
	})
}

func (h *Handler) HandleListAvailableDocuments(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	search := c.Query("search")

	items, err := h.listAvailableUC.Execute(c.Request.Context(), id, search)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar documentos disponíveis"})
		return
	}

	c.HTML(http.StatusOK, "specialist/documents_modal.html", gin.H{
		"AvailableDocuments": items,
		"SpecialistID":       id,
		"Search":             search,
	})
}

func (h *Handler) HandleAssociateDocuments(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	docIDs := c.PostFormArray("document_ids[]")
	if len(docIDs) == 0 {
		docIDs = c.PostFormArray("document_ids")
	}

	if len(docIDs) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Selecione pelo menos um documento"})
		return
	}

	err := h.associateUC.Execute(c.Request.Context(), application.AssociateDocumentInput{
		SpecialistID: id,
		DocumentIDs:  docIDs,
	})
	if err != nil {
		if errors.Is(err, specialistdomain.ErrSpecialistNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Especialista não encontrado"})
			return
		}
		if errors.Is(err, specialistdomain.ErrSpecialistInactive) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Especialista está inativo"})
			return
		}
		if errors.Is(err, application.ErrAssociationBatchTooLarge) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Máximo de %d documentos por associação", application.MaxAssociationBatchSize)})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapDomainError(err)})
		return
	}

	h.HandleListSpecialistDocuments(c)
}

func (h *Handler) HandleDissociateDocument(c *gin.Context) {
	id := c.Param("id")
	docID := c.Param("doc_id")

	err := h.dissociateUC.Execute(c.Request.Context(), id, docID)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotAssociated) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Documento não está associado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao desassociar documento"})
		return
	}

	h.HandleListSpecialistDocuments(c)
}

func mapDomainError(err error) string {
	switch {
	case errors.Is(err, domain.ErrDocumentNameRequired):
		return "Nome é obrigatório"
	case errors.Is(err, domain.ErrDocumentNameTooLong):
		return "Nome excede o tamanho máximo"
	case errors.Is(err, domain.ErrDocumentTypeNotAllowed):
		return "Tipo de arquivo não permitido"
	case errors.Is(err, domain.ErrDocumentFilePathRequired):
		return "Arquivo é obrigatório"
	case errors.Is(err, domain.ErrDocumentAlreadyAssociated):
		return "Documento já está associado"
	case errors.Is(err, domain.ErrDocumentNotAssociated):
		return "Documento não está associado"
	case errors.Is(err, domain.ErrDocumentNotFound):
		return "Documento não encontrado"
	default:
		return "Erro interno"
	}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
