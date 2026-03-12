package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/service"
)

type VODHandler struct {
	vodService *service.VODService
	log        zerolog.Logger
}

func NewVODHandler(vodService *service.VODService, log zerolog.Logger) *VODHandler {
	return &VODHandler{
		vodService: vodService,
		log:        log.With().Str("handler", "vod").Logger(),
	}
}

func (h *VODHandler) ProbeStream(w http.ResponseWriter, r *http.Request) {
	streamID, err := urlParamInt64(r, "streamID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid stream id")
		return
	}

	result, err := h.vodService.ProbeStream(r.Context(), streamID)
	if err != nil {
		h.log.Error().Err(err).Int64("stream_id", streamID).Msg("probe failed")
		respondError(w, http.StatusNotFound, "stream not found")
		return
	}

	if result.IsVOD {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"type":     "vod",
			"duration": result.Duration,
			"width":    result.Width,
			"height":   result.Height,
		})
	} else {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"type": "live",
		})
	}
}

func (h *VODHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	streamID, err := urlParamInt64(r, "streamID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid stream id")
		return
	}

	profileName := r.URL.Query().Get("profile")

	session, err := h.vodService.CreateSession(r.Context(), streamID, profileName)
	if err != nil {
		h.log.Error().Err(err).Int64("stream_id", streamID).Msg("create VOD session failed")
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"session_id": session.ID,
		"duration":   session.Duration,
	})
}

func (h *VODHandler) CreateChannelSession(w http.ResponseWriter, r *http.Request) {
	channelID, err := urlParamInt64(r, "channelID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	profileName := r.URL.Query().Get("profile")

	session, err := h.vodService.CreateSessionForChannel(r.Context(), channelID, profileName)
	if err != nil {
		h.log.Error().Err(err).Int64("channel_id", channelID).Msg("create channel VOD session failed")
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"session_id": session.ID,
		"duration":   session.Duration,
	})
}

func (h *VODHandler) Status(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	session, ok := h.vodService.GetSession(sessionID)
	if !ok {
		respondError(w, http.StatusNotFound, "session not found")
		return
	}

	buffered := session.GetBufferedSecs()

	ready := false
	select {
	case <-session.Ready:
		ready = true
	default:
	}

	errMsg := ""
	if session.Error != nil {
		errMsg = session.Error.Error()
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"buffered": buffered,
		"duration": session.Duration,
		"ready":    ready,
		"error":    errMsg,
	})
}

func (h *VODHandler) Seek(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	session, ok := h.vodService.GetSession(sessionID)
	if !ok {
		respondError(w, http.StatusNotFound, "session not found")
		return
	}

	tStr := r.URL.Query().Get("t")
	offset, err := strconv.ParseFloat(tStr, 64)
	if err != nil || offset < 0 {
		respondError(w, http.StatusBadRequest, "invalid time offset")
		return
	}

	reader, err := h.vodService.StreamSeek(r.Context(), session, offset)
	if err != nil {
		h.log.Error().Err(err).Str("session_id", sessionID).Float64("offset", offset).Msg("seek failed")
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	buf := make([]byte, 32*1024)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if readErr != nil {
			return
		}
	}
}

func (h *VODHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	h.vodService.DeleteSession(sessionID)
	w.WriteHeader(http.StatusNoContent)
}
