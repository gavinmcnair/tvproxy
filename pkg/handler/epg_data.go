package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

type EPGDataHandler struct {
	epgDataRepo     *repository.EPGDataRepository
	programDataRepo *repository.ProgramDataRepository
}

func NewEPGDataHandler(epgDataRepo *repository.EPGDataRepository, programDataRepo *repository.ProgramDataRepository) *EPGDataHandler {
	return &EPGDataHandler{
		epgDataRepo:     epgDataRepo,
		programDataRepo: programDataRepo,
	}
}

type epgDataWithPrograms struct {
	ID          string      `json:"id"`
	EPGSourceID string      `json:"epg_source_id"`
	ChannelID   string      `json:"channel_id"`
	Name        string      `json:"name"`
	Icon        string      `json:"icon,omitempty"`
	Programs    interface{} `json:"programs"`
}

func (h *EPGDataHandler) List(w http.ResponseWriter, r *http.Request) {
	sourceIDStr := r.URL.Query().Get("source_id")
	includePrograms := r.URL.Query().Get("programs") == "true"

	var data []models.EPGData
	var err error

	if sourceIDStr != "" {
		data, err = h.epgDataRepo.ListBySourceID(r.Context(), sourceIDStr)
	} else {
		data, err = h.epgDataRepo.List(r.Context())
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list epg data")
		return
	}

	if !includePrograms {
		respondJSON(w, http.StatusOK, data)
		return
	}

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

func (h *EPGDataHandler) NowPlaying(w http.ResponseWriter, r *http.Request) {
	channelID := r.URL.Query().Get("channel_id")
	if channelID == "" {
		respondError(w, http.StatusBadRequest, "channel_id is required")
		return
	}

	program, err := h.programDataRepo.GetNowByChannelID(r.Context(), channelID, time.Now())
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusOK, nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get current program")
		return
	}
	respondJSON(w, http.StatusOK, program)
}

type guideResponse struct {
	Start    time.Time                            `json:"start"`
	Stop     time.Time                            `json:"stop"`
	Programs map[string][]repository.GuideProgram `json:"programs"`
}

func (h *EPGDataHandler) Guide(w http.ResponseWriter, r *http.Request) {
	hours := 6
	if hs := r.URL.Query().Get("hours"); hs != "" {
		if parsed, err := strconv.Atoi(hs); err == nil && parsed > 0 && parsed <= 48 {
			hours = parsed
		}
	}

	var start time.Time
	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = parsed.Truncate(30 * time.Minute)
		}
	}
	if start.IsZero() {
		start = time.Now().Truncate(30 * time.Minute)
	}
	stop := start.Add(time.Duration(hours) * time.Hour)

	programs, err := h.programDataRepo.ListForGuide(r.Context(), start, stop)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list guide programs")
		return
	}

	grouped := make(map[string][]repository.GuideProgram)
	for _, p := range programs {
		grouped[p.ChannelID] = append(grouped[p.ChannelID], p)
	}

	respondJSON(w, http.StatusOK, guideResponse{
		Start:    start,
		Stop:     stop,
		Programs: grouped,
	})
}
