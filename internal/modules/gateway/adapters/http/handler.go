package http

import (
	"encoding/json"
	"net/http"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/internal/modules/gateway"
)

type Handler struct {
	useCase gateway.IngestService
	logger  log.Logger
}

func NewHandler(uc gateway.IngestService, l log.Logger) *Handler {
	return &Handler{
		useCase: uc,
		logger:  l,
	}
}

// HandleIngest — endpoint for uploading documents (POST /v1/ingest)
func (h *Handler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.writeError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "expected form field 'file'")
		return
	}
	defer file.Close()

	filename := header.Filename
	mimeType := header.Header.Get("Content-Type")
	size := header.Size

	docID, err := h.useCase.IngestDocument(r.Context(), tenantID, filename, mimeType, size, file)
	if err != nil {
		h.logger.Error().Err(err).Msg("ingest use case failed")
		h.writeError(w, http.StatusInternalServerError, "internal processing error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":      "accepted",
		"document_id": docID,
	})
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
