package handler

import (
	"net/http"

	"github.com/yourusername/maradhi-api/internal/auth"
)

type loginRequest    struct { Username string `json:"username"`; Password string `json:"password"` }
type registerRequest struct { Username string `json:"username"`; Password string `json:"password"`; Email string `json:"email"`; IsAdmin bool `json:"is_admin"` }
type authResponse    struct { Token string `json:"token"`; UserID string `json:"user_id"`; Username string `json:"username"`; IsAdmin bool `json:"is_admin"` }

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decode(w, r, &req) { return }
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}
	user, err := h.users.GetByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := auth.GenerateToken(h.cfg.JWTSecret, user.ID, user.Username, user.IsAdmin)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate token")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, UserID: user.ID, Username: user.Username, IsAdmin: user.IsAdmin})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Admin-Secret") != h.cfg.AdminSecret {
		writeError(w, http.StatusForbidden, "invalid admin secret")
		return
	}
	var req registerRequest
	if !decode(w, r, &req) { return }
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.users.Create(r.Context(), req.Username, req.Email, hash, req.IsAdmin)
	if err != nil {
		writeError(w, http.StatusConflict, "username already taken")
		return
	}
	token, err := auth.GenerateToken(h.cfg.JWTSecret, user.ID, user.Username, user.IsAdmin)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate token")
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{Token: token, UserID: user.ID, Username: user.Username, IsAdmin: user.IsAdmin})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.users.GetByID(r.Context(), userID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": user.ID, "username": user.Username,
		"email": user.Email, "is_admin": user.IsAdmin,
	})
}
