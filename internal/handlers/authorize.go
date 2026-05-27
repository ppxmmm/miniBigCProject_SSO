package handlers

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type loginData struct {
	AppName             string
	ClientID            string
	RedirectURI         string
	ResponseType        string
	Scope               string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	Error               string
}

const loginHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Sign In</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background: #f0f2f5;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .card {
      background: #fff;
      border-radius: 12px;
      box-shadow: 0 4px 24px rgba(0,0,0,0.08);
      padding: 2.5rem;
      width: 100%;
      max-width: 420px;
    }
    .logo { font-size: 1.75rem; font-weight: 700; color: #1a1a2e; margin-bottom: 0.25rem; }
    .subtitle { color: #666; font-size: 0.875rem; margin-bottom: 2rem; }
    .subtitle strong { color: #333; }
    .error {
      background: #fff0f0;
      border: 1px solid #ffcdd2;
      color: #c62828;
      padding: 0.625rem 0.875rem;
      border-radius: 6px;
      margin-bottom: 1.25rem;
      font-size: 0.875rem;
    }
    label { display: block; font-size: 0.8125rem; font-weight: 600; color: #444; margin-bottom: 0.375rem; }
    input[type=email], input[type=password] {
      width: 100%;
      padding: 0.625rem 0.875rem;
      border: 1.5px solid #ddd;
      border-radius: 8px;
      font-size: 0.9375rem;
      margin-bottom: 1.125rem;
      transition: border-color 0.2s;
      outline: none;
    }
    input:focus { border-color: #4f46e5; box-shadow: 0 0 0 3px rgba(79,70,229,0.12); }
    button {
      width: 100%;
      padding: 0.75rem;
      background: #4f46e5;
      color: #fff;
      border: none;
      border-radius: 8px;
      font-size: 1rem;
      font-weight: 600;
      cursor: pointer;
      transition: background 0.2s;
    }
    button:hover { background: #4338ca; }
    .footer { text-align: center; margin-top: 1.5rem; font-size: 0.8125rem; color: #888; }
  </style>
</head>
<body>
  <div class="card">
    <div class="logo">SSO</div>
    {{if .AppName}}
    <p class="subtitle">Sign in to access <strong>{{.AppName}}</strong></p>
    {{else}}
    <p class="subtitle">Single Sign-On</p>
    {{end}}
    {{if .Error}}<div class="error">{{.Error}}</div>{{end}}
    <form method="POST" action="/authorize">
      <input type="hidden" name="client_id"             value="{{.ClientID}}">
      <input type="hidden" name="redirect_uri"          value="{{.RedirectURI}}">
      <input type="hidden" name="response_type"         value="{{.ResponseType}}">
      <input type="hidden" name="scope"                 value="{{.Scope}}">
      <input type="hidden" name="state"                 value="{{.State}}">
      <input type="hidden" name="nonce"                 value="{{.Nonce}}">
      <input type="hidden" name="code_challenge"        value="{{.CodeChallenge}}">
      <input type="hidden" name="code_challenge_method" value="{{.CodeChallengeMethod}}">
      <label for="email">Email</label>
      <input type="email" id="email" name="email" required autofocus autocomplete="email">
      <label for="password">Password</label>
      <input type="password" id="password" name="password" required autocomplete="current-password">
      <button type="submit">Sign In</button>
    </form>
    <p class="footer">Secured by SSO &mdash; OAuth2 / OIDC</p>
  </div>
</body>
</html>`

var loginTmpl = template.Must(template.New("login").Parse(loginHTML))

func (h *Handler) AuthorizeGET(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	responseType := q.Get("response_type")

	if clientID == "" || redirectURI == "" || responseType == "" {
		http.Error(w, "missing required parameters: client_id, redirect_uri, response_type", http.StatusBadRequest)
		return
	}
	if responseType != "code" {
		http.Error(w, "unsupported response_type: only 'code' is supported", http.StatusBadRequest)
		return
	}

	client, err := h.db.GetClientByClientID(clientID)
	if err != nil {
		log.Printf("authorize: get client: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if client == nil {
		http.Error(w, "unknown client_id", http.StatusBadRequest)
		return
	}
	if !containsURI(client.RedirectURIs, redirectURI) {
		http.Error(w, "redirect_uri not registered for this client", http.StatusBadRequest)
		return
	}

	data := loginData{
		AppName:             client.Name,
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		ResponseType:        responseType,
		Scope:               q.Get("scope"),
		State:               q.Get("state"),
		Nonce:               q.Get("nonce"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	loginTmpl.Execute(w, data)
}

func (h *Handler) AuthorizePOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	scope := r.FormValue("scope")
	state := r.FormValue("state")
	nonce := r.FormValue("nonce")
	challenge := r.FormValue("code_challenge")
	challengeMethod := r.FormValue("code_challenge_method")
	email := r.FormValue("email")
	password := r.FormValue("password")

	client, err := h.db.GetClientByClientID(clientID)
	if err != nil || client == nil || !containsURI(client.RedirectURIs, redirectURI) {
		http.Error(w, "invalid client", http.StatusBadRequest)
		return
	}

	baseData := loginData{
		AppName:             client.Name,
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		ResponseType:        "code",
		Scope:               scope,
		State:               state,
		Nonce:               nonce,
		CodeChallenge:       challenge,
		CodeChallengeMethod: challengeMethod,
	}

	user, err := h.db.GetUserByEmail(email)
	if err != nil {
		log.Printf("authorize: get user: %v", err)
		h.renderLoginError(w, baseData, "An internal error occurred. Please try again.")
		return
	}
	if user == nil || !checkPassword(user.PasswordHash, password) {
		h.renderLoginError(w, baseData, "Invalid email or password.")
		return
	}

	finalScope := intersectScope(scope, client.Scopes)
	if !strings.Contains(finalScope, "openid") {
		finalScope = strings.TrimSpace("openid " + finalScope)
	}

	code, err := h.db.CreateAuthCode(clientID, user.ID, redirectURI, finalScope, nonce, challenge, challengeMethod)
	if err != nil {
		log.Printf("authorize: create auth code: %v", err)
		h.renderLoginError(w, baseData, "An internal error occurred. Please try again.")
		return
	}

	redirectURL, _ := url.Parse(redirectURI)
	q := redirectURL.Query()
	q.Set("code", code.Code)
	if state != "" {
		q.Set("state", state)
	}
	redirectURL.RawQuery = q.Encode()
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func (h *Handler) renderLoginError(w http.ResponseWriter, data loginData, msg string) {
	data.Error = msg
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	loginTmpl.Execute(w, data)
}

func containsURI(uris []string, uri string) bool {
	for _, u := range uris {
		if u == uri {
			return true
		}
	}
	return false
}

func intersectScope(requested, allowed string) string {
	allowedSet := make(map[string]bool)
	for _, s := range strings.Fields(allowed) {
		allowedSet[s] = true
	}
	var result []string
	for _, s := range strings.Fields(requested) {
		if allowedSet[s] {
			result = append(result, s)
		}
	}
	return strings.Join(result, " ")
}
