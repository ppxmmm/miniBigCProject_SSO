package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) AdminCreateClient(w http.ResponseWriter, r *http.Request) {
	if !h.adminAuth(r) {
		h.writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid X-Admin-Secret header")
		return
	}

	var req struct {
		ClientID     string   `json:"client_id"`
		ClientSecret string   `json:"client_secret"`
		Name         string   `json:"name"`
		RedirectURIs []string `json:"redirect_uris"`
		Scopes       string   `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if req.ClientID == "" || req.ClientSecret == "" || req.Name == "" || len(req.RedirectURIs) == 0 {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "client_id, client_secret, name, and redirect_uris are required")
		return
	}
	if req.Scopes == "" {
		req.Scopes = "openid profile email"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.ClientSecret), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("admin: hash client secret: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "hash error")
		return
	}

	client, err := h.db.CreateClient(req.ClientID, string(hash), req.Name, req.RedirectURIs, req.Scopes)
	if err != nil {
		log.Printf("admin: create client: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "database error")
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]any{
		"id":            client.ID,
		"client_id":     client.ClientID,
		"name":          client.Name,
		"redirect_uris": client.RedirectURIs,
		"scopes":        client.Scopes,
	})
}

func (h *Handler) AdminListClients(w http.ResponseWriter, r *http.Request) {
	if !h.adminAuth(r) {
		h.writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid X-Admin-Secret header")
		return
	}

	clients, err := h.db.ListClients()
	if err != nil {
		log.Printf("admin: list clients: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "database error")
		return
	}

	type clientResp struct {
		ID           string   `json:"id"`
		ClientID     string   `json:"client_id"`
		Name         string   `json:"name"`
		RedirectURIs []string `json:"redirect_uris"`
		Scopes       string   `json:"scopes"`
	}
	result := make([]clientResp, 0, len(clients))
	for _, c := range clients {
		result = append(result, clientResp{
			ID: c.ID, ClientID: c.ClientID, Name: c.Name,
			RedirectURIs: c.RedirectURIs, Scopes: c.Scopes,
		})
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	if !h.adminAuth(r) {
		h.writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid X-Admin-Secret header")
		return
	}

	users, err := h.db.ListUsers()
	if err != nil {
		log.Printf("admin: list users: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "database error")
		return
	}

	type userResp struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	}
	result := make([]userResp, 0, len(users))
	for _, u := range users {
		result = append(result, userResp{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role})
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	if !h.adminAuth(r) {
		h.writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid X-Admin-Secret header")
		return
	}

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
		log.Printf("admin: check existing user: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "database error")
		return
	}
	if existing != nil {
		h.writeOAuthError(w, http.StatusConflict, "conflict", "email already registered")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("admin: hash password: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "hash error")
		return
	}

	user, err := h.db.CreateUser(req.Email, string(hash), req.Name, req.Role)
	if err != nil {
		log.Printf("admin: create user: %v", err)
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

func (h *Handler) AdminUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	if !h.adminAuth(r) {
		h.writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid X-Admin-Secret header")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "missing user id")
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if err := h.db.UpdateUserRole(id, req.Role); err != nil {
		if err.Error() == "user not found" {
			h.writeOAuthError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"id": id, "role": req.Role})
}
