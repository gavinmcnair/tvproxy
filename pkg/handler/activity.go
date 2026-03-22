package handler

import (
	"net/http"

	"github.com/gavinmcnair/tvproxy/pkg/service"
)

type ActivityHandler struct {
	activityService *service.ActivityService
}

func NewActivityHandler(activityService *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{activityService: activityService}
}

func (h *ActivityHandler) List(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, h.activityService.List())
}
