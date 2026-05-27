package handlers

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"sso/internal/config"
	"sso/internal/db"
	"sso/internal/keys"
)

type Handler struct {
	cfg  *config.Config
	db   *db.DB
	keys *keys.KeyPair
}

func New(cfg *config.Config, database *db.DB, keyPair *keys.KeyPair) *Handler {
	return &Handler{cfg: cfg, db: database, keys: keyPair}
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) writeOAuthError(w http.ResponseWriter, status int, code, desc string) {
	h.writeJSON(w, status, map[string]string{
		"error":             code,
		"error_description": desc,
	})
}

func (h *Handler) adminAuth(r *http.Request) bool {
	return r.Header.Get("X-Admin-Secret") == h.cfg.AdminSecret
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
