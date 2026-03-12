package handler

import (
	"net/http"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/service"
)

type ProxyHandler struct {
	proxyService *service.ProxyService
	log          zerolog.Logger
}

func NewProxyHandler(proxyService *service.ProxyService, log zerolog.Logger) *ProxyHandler {
	return &ProxyHandler{
		proxyService: proxyService,
		log:          log.With().Str("handler", "proxy").Logger(),
	}
}

func (h *ProxyHandler) Stream(w http.ResponseWriter, r *http.Request) {
	channelID, err := urlParamInt64(r, "channelID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid channel id")
		return
	}

	if err := h.proxyService.ProxyStream(r.Context(), w, r, channelID, r.URL.Query().Get("profile")); err != nil {
		h.log.Error().Err(err).Int64("channel_id", channelID).Msg("proxy stream failed")
		respondError(w, http.StatusInternalServerError, "failed to proxy stream")
	}
}

func (h *ProxyHandler) RawStream(w http.ResponseWriter, r *http.Request) {
	streamID, err := urlParamInt64(r, "streamID")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid stream id")
		return
	}

	if err := h.proxyService.ProxyRawStream(r.Context(), w, r, streamID, r.URL.Query().Get("profile")); err != nil {
		h.log.Error().Err(err).Int64("stream_id", streamID).Msg("raw stream proxy failed")
		respondError(w, http.StatusInternalServerError, "failed to proxy stream")
	}
}
