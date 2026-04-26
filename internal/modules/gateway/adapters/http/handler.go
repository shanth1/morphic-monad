package http

import (
	"encoding/json"
	"net/http"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/internal/modules/gateway"
)

type Handler struct {
	svc    gateway.GatewayService
	logger log.Logger
}

func NewHandler(svc gateway.GatewayService, l log.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: l,
	}
}

// HandleIngest — endpoint for uploading documents/text (POST /v1/ingest)
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

	// Extract short text (context) if it was passed
	contextText := r.FormValue("context_text")

	// Trying to extract a file (it's now optional)
	var filename, mimeType string
	var size int64

	file, header, err := r.FormFile("file")
	if err == nil {
		defer file.Close()
		filename = header.Filename
		mimeType = header.Header.Get("Content-Type")
		size = header.Size
	} else if err != http.ErrMissingFile {
		h.writeError(w, http.StatusBadRequest, "error reading file field")
		return
	}

	// Business validation: Either text, a file, or both must be submitted
	if contextText == "" && file == nil {
		h.writeError(w, http.StatusBadRequest, "either 'context_text' or 'file' must be provided")
		return
	}

	// We pass everything to the Gateway core (it will sort it out itself: it will throw the text into the bus, and the file into the BlobStore)
	docID, err := h.svc.IngestDocument(r.Context(), tenantID, contextText, filename, mimeType, size, file)
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

// SearchRequest DTO for synchronous search
type SearchRequest struct {
	QueryText string `json:"query_text"`
	TopK      int    `json:"top_k"`
}

// HandleSearch — endpoint for searching (POST /v1/search)
func (h *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.writeError(w, http.StatusBadRequest, "X-Tenant-ID header is required")
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	// We call a synchronous method, which under the hood waits for an asynchronous response from the bus
	results, err := h.svc.SearchDocuments(r.Context(), tenantID, req.QueryText, "", req.TopK)
	if err != nil {
		h.logger.Error().Err(err).Msg("search use case failed")
		h.writeError(w, http.StatusGatewayTimeout, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"results": results,
	})
}

// Helper function for standardizing error output
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
