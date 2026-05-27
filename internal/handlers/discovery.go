package handlers

import (
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"net/http"
)

func (h *Handler) Discovery(w http.ResponseWriter, r *http.Request) {
	issuer := h.cfg.Issuer
	h.writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/authorize",
		"token_endpoint":                        issuer + "/token",
		"userinfo_endpoint":                     issuer + "/userinfo",
		"jwks_uri":                              issuer + "/.well-known/jwks.json",
		"registration_endpoint":                 issuer + "/register",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic"},
		"claims_supported":                      []string{"sub", "email", "name", "iat", "exp", "iss", "aud", "nonce"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
	})
}

func (h *Handler) JWKS(w http.ResponseWriter, r *http.Request) {
	pub := &h.keys.Private.PublicKey
	h.writeJSON(w, http.StatusOK, map[string]any{
		"keys": []map[string]any{jwkFromRSA(pub, h.keys.KeyID)},
	})
}

func jwkFromRSA(pub *rsa.PublicKey, kid string) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": kid,
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}
