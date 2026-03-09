package handler

import (
	"net/http"
	"strconv"

	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

// EPGDataHandler handles EPG data HTTP requests.
type EPGDataHandler struct {
	epgDataRepo     *repository.EPGDataRepository
	programDataRepo *repository.ProgramDataRepository
}

// NewEPGDataHandler creates a new EPGDataHandler.
func NewEPGDataHandler(epgDataRepo *repository.EPGDataRepository, programDataRepo *repository.ProgramDataRepository) *EPGDataHandler {
	return &EPGDataHandler{
		epgDataRepo:     epgDataRepo,
		programDataRepo: programDataRepo,
	}
}

// epgDataWithPrograms represents EPG data combined with its programs for the API response.
type epgDataWithPrograms struct {
	ID          int64       `json:"id"`
	EPGSourceID int64       `json:"epg_source_id"`
	ChannelID   string      `json:"channel_id"`
	Name        string      `json:"name"`
	Icon        string      `json:"icon,omitempty"`
	Programs    interface{} `json:"programs"`
}

// List returns all EPG data, optionally filtered by source_id query parameter.
// By default returns channel summaries without programs. Add ?programs=true to include programs.
func (h *EPGDataHandler) List(w http.ResponseWriter, r *http.Request) {
	sourceIDStr := r.URL.Query().Get("source_id")
	includePrograms := r.URL.Query().Get("programs") == "true"

	var data []models.EPGData
	var err error

	if sourceIDStr != "" {
		sourceID, parseErr := strconv.ParseInt(sourceIDStr, 10, 64)
		if parseErr != nil {
			respondError(w, http.StatusBadRequest, "invalid source_id")
			return
		}
		data, err = h.epgDataRepo.ListBySourceID(r.Context(), sourceID)
	} else {
		data, err = h.epgDataRepo.List(r.Context())
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list epg data")
		return
	}

	if !includePrograms {
		// Return lightweight channel list without programs
		respondJSON(w, http.StatusOK, data)
		return
	}

	// Include programs (expensive — only when explicitly requested)
	results := make([]epgDataWithPrograms, 0, len(data))
	for _, d := range data {
		programs, progErr := h.programDataRepo.ListByEPGDataID(r.Context(), d.ID)
		if progErr != nil {
			respondError(w, http.StatusInternalServerError, "failed to list program data")
			return
		}
		results = append(results, epgDataWithPrograms{
			ID:          d.ID,
			EPGSourceID: d.EPGSourceID,
			ChannelID:   d.ChannelID,
			Name:        d.Name,
			Icon:        d.Icon,
			Programs:    programs,
		})
	}

	respondJSON(w, http.StatusOK, results)
}
