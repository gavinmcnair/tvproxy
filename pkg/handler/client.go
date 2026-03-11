package handler

import (
	"net/http"

	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/service"
)

var validMatchTypes = map[string]bool{"exists": true, "contains": true, "equals": true, "prefix": true}

// ClientHandler handles client detection HTTP requests.
type ClientHandler struct {
	clientService *service.ClientService
}

// NewClientHandler creates a new ClientHandler.
func NewClientHandler(clientService *service.ClientService) *ClientHandler {
	return &ClientHandler{clientService: clientService}
}

// List returns all clients with their match rules.
func (h *ClientHandler) List(w http.ResponseWriter, r *http.Request) {
	clients, err := h.clientService.ListClients(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list clients")
		return
	}

	respondJSON(w, http.StatusOK, clients)
}

type clientCreateRequest struct {
	Name      string                   `json:"name"`
	Priority  int                      `json:"priority"`
	IsEnabled bool                     `json:"is_enabled"`
	Rules     []clientMatchRuleRequest `json:"match_rules"`
}

type clientMatchRuleRequest struct {
	HeaderName string `json:"header_name"`
	MatchType  string `json:"match_type"`
	MatchValue string `json:"match_value"`
}

// Create creates a new client with auto-created stream profile.
func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req clientCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(req.Rules) == 0 {
		respondError(w, http.StatusBadRequest, "at least one match rule is required")
		return
	}
	if err := validateRules(req.Rules); err != "" {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	client := &models.Client{
		Name:      req.Name,
		Priority:  req.Priority,
		IsEnabled: req.IsEnabled,
	}

	rules := toMatchRules(0, req.Rules)

	if err := h.clientService.CreateClient(r.Context(), client, rules); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create client")
		return
	}

	// Reload to get hydrated rules
	client, err := h.clientService.GetClient(r.Context(), client.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload client")
		return
	}

	respondJSON(w, http.StatusCreated, client)
}

// Get returns a client by ID.
func (h *ClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt64(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid client id")
		return
	}

	client, err := h.clientService.GetClient(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "client not found")
		return
	}

	respondJSON(w, http.StatusOK, client)
}

// Update updates a client by ID.
func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt64(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid client id")
		return
	}

	client, err := h.clientService.GetClient(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "client not found")
		return
	}

	var req struct {
		Name            string                   `json:"name"`
		Priority        *int                     `json:"priority"`
		StreamProfileID *int64                   `json:"stream_profile_id"`
		IsEnabled       *bool                    `json:"is_enabled"`
		Rules           []clientMatchRuleRequest `json:"match_rules"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate rules BEFORE making any changes
	if req.Rules != nil {
		if len(req.Rules) == 0 {
			respondError(w, http.StatusBadRequest, "at least one match rule is required")
			return
		}
		if errMsg := validateRules(req.Rules); errMsg != "" {
			respondError(w, http.StatusBadRequest, errMsg)
			return
		}
	}

	if req.Name != "" {
		client.Name = req.Name
	}
	if req.Priority != nil {
		client.Priority = *req.Priority
	}
	if req.StreamProfileID != nil {
		client.StreamProfileID = *req.StreamProfileID
	}
	if req.IsEnabled != nil {
		client.IsEnabled = *req.IsEnabled
	}

	var rules []models.ClientMatchRule
	if req.Rules != nil {
		rules = toMatchRules(client.ID, req.Rules)
	}

	if err := h.clientService.UpdateClient(r.Context(), client, rules); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update client")
		return
	}

	// Reload
	client, err = h.clientService.GetClient(r.Context(), client.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to reload client")
		return
	}

	respondJSON(w, http.StatusOK, client)
}

// Delete deletes a client by ID. Also cleans up the linked stream profile if orphaned.
func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt64(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid client id")
		return
	}

	if err := h.clientService.DeleteClient(r.Context(), id); err != nil {
		respondError(w, http.StatusNotFound, "client not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validateRules(rules []clientMatchRuleRequest) string {
	for _, rule := range rules {
		if rule.HeaderName == "" {
			return "header_name is required on each rule"
		}
		if !validMatchTypes[rule.MatchType] {
			return "match_type must be one of: exists, contains, equals, prefix"
		}
		if rule.MatchType != "exists" && rule.MatchValue == "" {
			return "match_value is required unless match_type is exists"
		}
	}
	return ""
}

// toMatchRules converts request rule DTOs to model objects.
func toMatchRules(clientID int64, reqs []clientMatchRuleRequest) []models.ClientMatchRule {
	rules := make([]models.ClientMatchRule, len(reqs))
	for i, rr := range reqs {
		rules[i] = models.ClientMatchRule{
			ClientID:   clientID,
			HeaderName: rr.HeaderName,
			MatchType:  rr.MatchType,
			MatchValue: rr.MatchValue,
		}
	}
	return rules
}
