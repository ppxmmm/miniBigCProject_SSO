package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func (h *Handler) Userinfo(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	tokenStr, found := strings.CutPrefix(authHeader, "Bearer ")
	if !found || tokenStr == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="SSO"`)
		h.writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "missing Bearer token")
		return
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return &h.keys.Private.PublicKey, nil
	}, jwt.WithIssuer(h.cfg.Issuer))

	if err != nil || !token.Valid {
		w.Header().Set("WWW-Authenticate", `Bearer realm="SSO", error="invalid_token"`)
		h.writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "token validation failed")
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		h.writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "invalid token claims")
		return
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		h.writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "missing sub claim")
		return
	}

	user, err := h.db.GetUserByID(sub)
	if err != nil {
		log.Printf("userinfo: get user: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "database error")
		return
	}
	if user == nil {
		h.writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "user not found")
		return
	}

	scope, _ := claims["scope"].(string)

	result := map[string]any{"sub": user.ID}
	if strings.Contains(scope, "email") {
		result["email"] = user.Email
	}
	if strings.Contains(scope, "profile") {
		result["name"] = user.Name
		result["role"] = user.Role
	}

	h.writeJSON(w, http.StatusOK, result)
}
