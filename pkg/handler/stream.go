package handler

import (
	"net/http"
	"strconv"

	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

// StreamHandler handles stream-related HTTP requests.
type StreamHandler struct {
	streamRepo *repository.StreamRepository
}

// NewStreamHandler creates a new StreamHandler.
func NewStreamHandler(streamRepo *repository.StreamRepository) *StreamHandler {
	return &StreamHandler{streamRepo: streamRepo}
}

// List returns streams. Uses lightweight summaries by default.
// Add ?full=true for full stream details, or ?account_id= to filter.
func (h *StreamHandler) List(w http.ResponseWriter, r *http.Request) {
	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr != "" {
		accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid account_id")
			return
		}

		streams, err := h.streamRepo.ListByAccountID(r.Context(), accountID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to list streams")
			return
		}

		respondJSON(w, http.StatusOK, streams)
		return
	}

	if r.URL.Query().Get("full") == "true" {
		streams, err := h.streamRepo.List(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to list streams")
			return
		}
		respondJSON(w, http.StatusOK, streams)
		return
	}

	// Default: lightweight summaries (id, name, group, account_id only)
	summaries, err := h.streamRepo.ListSummaries(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list streams")
		return
	}

	respondJSON(w, http.StatusOK, summaries)
}

// Get returns a stream by ID (full details).
func (h *StreamHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt64(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid stream id")
		return
	}

	stream, err := h.streamRepo.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "stream not found")
		return
	}

	respondJSON(w, http.StatusOK, stream)
}

// Delete deletes a stream by ID.
func (h *StreamHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt64(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid stream id")
		return
	}

	if err := h.streamRepo.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete stream")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
