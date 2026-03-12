package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/tvproxy/pkg/service"
)

type UserHandler struct {
	authService *service.AuthService
}

func NewUserHandler(authService *service.AuthService) *UserHandler {
	return &UserHandler{authService: authService}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.authService.ListUsers(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	respondJSON(w, http.StatusOK, users)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		IsAdmin  bool   `json:"is_admin"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	user, err := h.authService.CreateUser(r.Context(), req.Username, req.Password, req.IsAdmin)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	respondJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := h.authService.GetUser(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		IsAdmin  bool   `json:"is_admin"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.authService.GetUser(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	if req.Username != "" {
		user.Username = req.Username
	}
	user.IsAdmin = req.IsAdmin

	if err := h.authService.UpdateUser(r.Context(), user, req.Password); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.authService.DeleteUser(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) Invite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" {
		respondError(w, http.StatusBadRequest, "username is required")
		return
	}

	user, err := h.authService.CreateInvite(r.Context(), req.Username)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create invite")
		return
	}

	respondJSON(w, http.StatusCreated, user)
}
