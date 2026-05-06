package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
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
		h.writeError(w, http.StatusBadRequest, "X-Tenant-ID is required")
		return
	}

	err := r.ParseMultipartForm(100 << 20) // 100 MB max
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to parse form")
		return
	}
	defer r.MultipartForm.RemoveAll()

	text := r.FormValue("text")
	files := r.MultipartForm.File["files"]

	if text == "" && len(files) == 0 {
		h.writeError(w, http.StatusBadRequest, "text or files required")
		return
	}

	correlationID := uuid.NewString()
	var docIDs []string

	// 1. Process Text
	if text != "" {
		docID, err := h.svc.IngestDocument(r.Context(), tenantID, correlationID, text, "", "text/plain", int64(len(text)), nil)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		docIDs = append(docIDs, docID)
	}

	// 2. Process Files
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}
		mimeType := fileHeader.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		docID, err := h.svc.IngestDocument(r.Context(), tenantID, correlationID, "", fileHeader.Filename, mimeType, fileHeader.Size, file)
		file.Close()
		if err != nil {
			continue
		}
		docIDs = append(docIDs, docID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"status":         "accepted",
		"correlation_id": correlationID,
		"document_ids":   docIDs,
	})
}

// HandleSearch — endpoint for searching (POST /v1/search)
func (h *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		h.writeError(w, http.StatusBadRequest, "X-Tenant-ID is required")
		return
	}

	// Парсим multipart, а не JSON!
	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to parse form, must be multipart/form-data")
		return
	}
	defer r.MultipartForm.RemoveAll()

	queryText := r.FormValue("query_text")
	topKStr := r.FormValue("top_k")
	topK := 5
	if k, err := strconv.Atoi(topKStr); err == nil && k > 0 {
		topK = k
	}

	var file io.ReadCloser
	var filename, mimeType string
	var size int64

	f, header, err := r.FormFile("file")
	if err == nil {
		file = f
		filename = header.Filename
		mimeType = header.Header.Get("Content-Type")
		size = header.Size
		defer file.Close()
	}

	results, err := h.svc.SearchDocuments(r.Context(), tenantID, queryText, filename, mimeType, size, file, topK)
	if err != nil {
		h.logger.Error().Err(err).Msg("search failed")
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"results": results})
}

// Helper function for standardizing error output
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func (h *Handler) HandleStreamEvents(w http.ResponseWriter, r *http.Request) {
	correlationID := r.URL.Query().Get("correlation_id")
	if correlationID == "" {
		h.writeError(w, http.StatusBadRequest, "correlation_id required")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, err := h.svc.ListenEvents(r.Context(), correlationID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case env := <-ch:
			data, _ := json.Marshal(map[string]any{
				"type":   env.Type,
				"source": env.Source,
			})
			w.Write([]byte("data: "))
			w.Write(data)
			w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

func (h *Handler) HandleBlob(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		h.writeError(w, http.StatusBadRequest, "uri required")
		return
	}

	rc, err := h.svc.GetBlob(r.Context(), uri)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "blob not found")
		return
	}
	defer rc.Close()

	if strings.HasSuffix(uri, ".png") {
		w.Header().Set("Content-Type", "image/png")
	} else if strings.HasSuffix(uri, ".jpg") {
		w.Header().Set("Content-Type", "image/jpeg")
	}

	io.Copy(w, rc)
}
