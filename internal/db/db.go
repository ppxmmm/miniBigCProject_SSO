package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	Role         string
	CreatedAt    time.Time
}

type Client struct {
	ID               string
	ClientID         string
	ClientSecretHash string
	Name             string
	RedirectURIs     []string
	Scopes           string
	CreatedAt        time.Time
}

type AuthCode struct {
	Code                string
	ClientID            string
	UserID              string
	RedirectURI         string
	Scope               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	ExpiresAt           time.Time
	Used                bool
}

type RefreshToken struct {
	Token     string
	ClientID  string
	UserID    string
	Scope     string
	ExpiresAt time.Time
	Revoked   bool
}

func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)

	db := &DB{sqlDB}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func (db *DB) migrate() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id           TEXT PRIMARY KEY,
			email        TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			name         TEXT NOT NULL,
			role         TEXT NOT NULL DEFAULT 'staff',
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS clients (
			id                  TEXT PRIMARY KEY,
			client_id           TEXT UNIQUE NOT NULL,
			client_secret_hash  TEXT NOT NULL,
			name                TEXT NOT NULL,
			redirect_uris       TEXT NOT NULL,
			scopes              TEXT NOT NULL,
			created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS authorization_codes (
			code                  TEXT PRIMARY KEY,
			client_id             TEXT NOT NULL,
			user_id               TEXT NOT NULL,
			redirect_uri          TEXT NOT NULL,
			scope                 TEXT NOT NULL,
			nonce                 TEXT NOT NULL DEFAULT '',
			code_challenge        TEXT NOT NULL DEFAULT '',
			code_challenge_method TEXT NOT NULL DEFAULT '',
			expires_at            DATETIME NOT NULL,
			used                  INTEGER NOT NULL DEFAULT 0,
			created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS refresh_tokens (
			token      TEXT PRIMARY KEY,
			client_id  TEXT NOT NULL,
			user_id    TEXT NOT NULL,
			scope      TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			revoked    INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	// Add role column to existing databases that predate this migration
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'staff'`)

	return nil
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// --- User ---

func (db *DB) CreateUser(email, passwordHash, name, role string) (*User, error) {
	if role != "manager" && role != "staff" {
		role = "staff"
	}
	id := generateID()
	now := time.Now()
	_, err := db.Exec(
		`INSERT INTO users (id, email, password_hash, name, role, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, email, passwordHash, name, role, now,
	)
	if err != nil {
		return nil, err
	}
	return &User{ID: id, Email: email, PasswordHash: passwordHash, Name: name, Role: role, CreatedAt: now}, nil
}

func (db *DB) GetUserByEmail(email string) (*User, error) {
	u := &User{}
	err := db.QueryRow(
		`SELECT id, email, password_hash, name, role, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (db *DB) GetUserByID(id string) (*User, error) {
	u := &User{}
	err := db.QueryRow(
		`SELECT id, email, password_hash, name, role, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (db *DB) UpdateUserRole(id, role string) error {
	if role != "manager" && role != "staff" {
		return fmt.Errorf("invalid role: must be 'manager' or 'staff'")
	}
	res, err := db.Exec(`UPDATE users SET role = ? WHERE id = ?`, role, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (db *DB) ListUsers() ([]*User, error) {
	rows, err := db.Query(`SELECT id, email, name, role, created_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// --- Client ---

func (db *DB) CreateClient(clientID, clientSecretHash, name string, redirectURIs []string, scopes string) (*Client, error) {
	id := generateID()
	uriJSON, _ := json.Marshal(redirectURIs)
	now := time.Now()
	_, err := db.Exec(
		`INSERT INTO clients (id, client_id, client_secret_hash, name, redirect_uris, scopes, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, clientID, clientSecretHash, name, string(uriJSON), scopes, now,
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		ID: id, ClientID: clientID, ClientSecretHash: clientSecretHash,
		Name: name, RedirectURIs: redirectURIs, Scopes: scopes, CreatedAt: now,
	}, nil
}

func (db *DB) GetClientByClientID(clientID string) (*Client, error) {
	c := &Client{}
	var uriJSON string
	err := db.QueryRow(
		`SELECT id, client_id, client_secret_hash, name, redirect_uris, scopes, created_at FROM clients WHERE client_id = ?`, clientID,
	).Scan(&c.ID, &c.ClientID, &c.ClientSecretHash, &c.Name, &uriJSON, &c.Scopes, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(uriJSON), &c.RedirectURIs)
	return c, nil
}

func (db *DB) ListClients() ([]*Client, error) {
	rows, err := db.Query(`SELECT id, client_id, name, redirect_uris, scopes, created_at FROM clients ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []*Client
	for rows.Next() {
		c := &Client{}
		var uriJSON string
		if err := rows.Scan(&c.ID, &c.ClientID, &c.Name, &uriJSON, &c.Scopes, &c.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(uriJSON), &c.RedirectURIs)
		clients = append(clients, c)
	}
	return clients, rows.Err()
}

// --- Authorization Code ---

func (db *DB) CreateAuthCode(clientID, userID, redirectURI, scope, nonce, challenge, challengeMethod string) (*AuthCode, error) {
	code := generateToken()
	expiresAt := time.Now().Add(10 * time.Minute)
	_, err := db.Exec(
		`INSERT INTO authorization_codes (code, client_id, user_id, redirect_uri, scope, nonce, code_challenge, code_challenge_method, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		code, clientID, userID, redirectURI, scope, nonce, challenge, challengeMethod, expiresAt,
	)
	if err != nil {
		return nil, err
	}
	return &AuthCode{
		Code: code, ClientID: clientID, UserID: userID,
		RedirectURI: redirectURI, Scope: scope, Nonce: nonce,
		CodeChallenge: challenge, CodeChallengeMethod: challengeMethod,
		ExpiresAt: expiresAt,
	}, nil
}

func (db *DB) UseAuthCode(code string) (*AuthCode, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	ac := &AuthCode{}
	err = tx.QueryRow(
		`SELECT code, client_id, user_id, redirect_uri, scope, nonce, code_challenge, code_challenge_method, expires_at, used FROM authorization_codes WHERE code = ?`, code,
	).Scan(&ac.Code, &ac.ClientID, &ac.UserID, &ac.RedirectURI, &ac.Scope, &ac.Nonce, &ac.CodeChallenge, &ac.CodeChallengeMethod, &ac.ExpiresAt, &ac.Used)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if ac.Used {
		return nil, fmt.Errorf("authorization code already used")
	}
	if time.Now().After(ac.ExpiresAt) {
		return nil, fmt.Errorf("authorization code expired")
	}

	if _, err = tx.Exec(`UPDATE authorization_codes SET used = 1 WHERE code = ?`, code); err != nil {
		return nil, err
	}
	return ac, tx.Commit()
}

// --- Refresh Token ---

func (db *DB) CreateRefreshToken(clientID, userID, scope string) (*RefreshToken, error) {
	token := generateToken()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	_, err := db.Exec(
		`INSERT INTO refresh_tokens (token, client_id, user_id, scope, expires_at) VALUES (?, ?, ?, ?, ?)`,
		token, clientID, userID, scope, expiresAt,
	)
	if err != nil {
		return nil, err
	}
	return &RefreshToken{Token: token, ClientID: clientID, UserID: userID, Scope: scope, ExpiresAt: expiresAt}, nil
}

func (db *DB) UseRefreshToken(token string) (*RefreshToken, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rt := &RefreshToken{}
	err = tx.QueryRow(
		`SELECT token, client_id, user_id, scope, expires_at, revoked FROM refresh_tokens WHERE token = ?`, token,
	).Scan(&rt.Token, &rt.ClientID, &rt.UserID, &rt.Scope, &rt.ExpiresAt, &rt.Revoked)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if rt.Revoked {
		return nil, fmt.Errorf("refresh token has been revoked")
	}
	if time.Now().After(rt.ExpiresAt) {
		return nil, fmt.Errorf("refresh token expired")
	}

	if _, err = tx.Exec(`UPDATE refresh_tokens SET revoked = 1 WHERE token = ?`, token); err != nil {
		return nil, err
	}
	return rt, tx.Commit()
}
