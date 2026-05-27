package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"sso/internal/db"
)

func (h *Handler) Token(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	if err := r.ParseForm(); err != nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "cannot parse form body")
		return
	}

	clientID, clientSecret := extractClientCredentials(r)
	if clientID == "" {
		h.writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "missing client credentials")
		return
	}

	client, err := h.db.GetClientByClientID(clientID)
	if err != nil {
		log.Printf("token: get client: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "database error")
		return
	}
	if client == nil || !checkPassword(client.ClientSecretHash, clientSecret) {
		h.writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "invalid client credentials")
		return
	}

	switch r.FormValue("grant_type") {
	case "authorization_code":
		h.handleAuthCodeGrant(w, r, clientID)
	case "refresh_token":
		h.handleRefreshGrant(w, r, clientID)
	default:
		h.writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "supported grant types: authorization_code, refresh_token")
	}
}

func (h *Handler) handleAuthCodeGrant(w http.ResponseWriter, r *http.Request, clientID string) {
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	codeVerifier := r.FormValue("code_verifier")

	if code == "" {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "missing code")
		return
	}

	authCode, err := h.db.UseAuthCode(code)
	if err != nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", err.Error())
		return
	}
	if authCode == nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "authorization code not found")
		return
	}
	if authCode.ClientID != clientID {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "code was not issued to this client")
		return
	}
	if authCode.RedirectURI != redirectURI {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "redirect_uri does not match")
		return
	}

	if authCode.CodeChallenge != "" {
		if codeVerifier == "" {
			h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "code_verifier required")
			return
		}
		if !verifyPKCE(codeVerifier, authCode.CodeChallenge, authCode.CodeChallengeMethod) {
			h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "code_verifier mismatch")
			return
		}
	}

	user, err := h.db.GetUserByID(authCode.UserID)
	if err != nil || user == nil {
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "user not found")
		return
	}

	h.issueTokens(w, clientID, user, authCode.Scope, authCode.Nonce)
}

func (h *Handler) handleRefreshGrant(w http.ResponseWriter, r *http.Request, clientID string) {
	tokenStr := r.FormValue("refresh_token")
	if tokenStr == "" {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "missing refresh_token")
		return
	}

	rt, err := h.db.UseRefreshToken(tokenStr)
	if err != nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", err.Error())
		return
	}
	if rt == nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "refresh token not found")
		return
	}
	if rt.ClientID != clientID {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "token was not issued to this client")
		return
	}

	user, err := h.db.GetUserByID(rt.UserID)
	if err != nil || user == nil {
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "user not found")
		return
	}

	h.issueTokens(w, clientID, user, rt.Scope, "")
}

func (h *Handler) issueTokens(w http.ResponseWriter, clientID string, user *db.User, scope, nonce string) {
	now := time.Now()
	expiry := now.Add(time.Hour)

	accessClaims := jwt.MapClaims{
		"iss":   h.cfg.Issuer,
		"sub":   user.ID,
		"aud":   []string{clientID},
		"iat":   now.Unix(),
		"exp":   expiry.Unix(),
		"scope": scope,
		"role":  user.Role,
	}
	accessTok := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessTok.Header["kid"] = h.keys.KeyID
	accessTokenStr, err := accessTok.SignedString(h.keys.Private)
	if err != nil {
		log.Printf("token: sign access token: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to sign token")
		return
	}

	var idTokenStr string
	if strings.Contains(scope, "openid") {
		idClaims := jwt.MapClaims{
			"iss": h.cfg.Issuer,
			"sub": user.ID,
			"aud": []string{clientID},
			"iat": now.Unix(),
			"exp": expiry.Unix(),
		}
		if nonce != "" {
			idClaims["nonce"] = nonce
		}
		if strings.Contains(scope, "email") {
			idClaims["email"] = user.Email
		}
		if strings.Contains(scope, "profile") {
			idClaims["name"] = user.Name
			idClaims["role"] = user.Role
		}
		idTok := jwt.NewWithClaims(jwt.SigningMethodRS256, idClaims)
		idTok.Header["kid"] = h.keys.KeyID
		idTokenStr, err = idTok.SignedString(h.keys.Private)
		if err != nil {
			log.Printf("token: sign id token: %v", err)
			h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "failed to sign id_token")
			return
		}
	}

	rt, err := h.db.CreateRefreshToken(clientID, user.ID, scope)
	if err != nil {
		log.Printf("token: create refresh token: %v", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "database error")
		return
	}

	resp := map[string]any{
		"access_token":  accessTokenStr,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": rt.Token,
		"scope":         scope,
	}
	if idTokenStr != "" {
		resp["id_token"] = idTokenStr
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func extractClientCredentials(r *http.Request) (clientID, secret string) {
	if id, sec, ok := r.BasicAuth(); ok {
		return id, sec
	}
	return r.FormValue("client_id"), r.FormValue("client_secret")
}

func verifyPKCE(verifier, challenge, method string) bool {
	switch method {
	case "S256":
		sum := sha256.Sum256([]byte(verifier))
		return base64.RawURLEncoding.EncodeToString(sum[:]) == challenge
	case "plain", "":
		return verifier == challenge
	default:
		return false
	}
}

