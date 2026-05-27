package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "email, password, and name are required")
		return
	}
	if len(req.Password) < 8 {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "password must be at least 8 characters")
		return
	}

	existing, err := h.db.GetUserByEmail(req.Email)
	if err != nil {
		log.Printf("register: check existing user: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "database error")
		return
	}
	if existing != nil {
		h.writeOAuthError(w, http.StatusConflict, "conflict", "email already registered")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("register: hash password: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "hash error")
		return
	}

	user, err := h.db.CreateUser(req.Email, string(hash), req.Name, req.Role)
	if err != nil {
		log.Printf("register: create user: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "database error")
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]any{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
	})
}
