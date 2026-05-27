package main

import (
	"log"
	"net/http"

	"sso/internal/config"
	"sso/internal/db"
	"sso/internal/handlers"
	"sso/internal/keys"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	keyPair, err := keys.LoadOrGenerate("private.pem")
	if err != nil {
		log.Fatalf("load RSA key: %v", err)
	}

	h := handlers.New(cfg, database, keyPair)

	mux := http.NewServeMux()

	// OIDC Discovery
	mux.HandleFunc("GET /.well-known/openid-configuration", h.Discovery)
	mux.HandleFunc("GET /.well-known/jwks.json", h.JWKS)

	// OAuth2 / OIDC core endpoints
	mux.HandleFunc("GET /authorize", h.AuthorizeGET)
	mux.HandleFunc("POST /authorize", h.AuthorizePOST)
	mux.HandleFunc("POST /token", h.Token)
	mux.HandleFunc("GET /userinfo", h.Userinfo)
	mux.HandleFunc("POST /userinfo", h.Userinfo)

	// User self-service registration
	mux.HandleFunc("POST /register", h.Register)

	// Admin API (protected by X-Admin-Secret header)
	mux.HandleFunc("POST /admin/clients", h.AdminCreateClient)
	mux.HandleFunc("GET /admin/clients", h.AdminListClients)
	mux.HandleFunc("GET /admin/users", h.AdminListUsers)
	mux.HandleFunc("POST /admin/users", h.AdminCreateUser)
	mux.HandleFunc("PATCH /admin/users/{id}/role", h.AdminUpdateUserRole)

	log.Printf("SSO server starting on %s", cfg.ServerAddr)
	log.Printf("Issuer: %s", cfg.Issuer)
	log.Printf("Discovery: %s/.well-known/openid-configuration", cfg.Issuer)
	if err := http.ListenAndServe(cfg.ServerAddr, mux); err != nil {
		log.Fatal(err)
	}
}
