package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"headcontrol/internal/model"
	"time"

	_ "modernc.org/sqlite"
)

type Sqlite struct {
	db *sql.DB
}

func NewSqlite(dbPath string) (*Sqlite, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	s := &Sqlite{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Sqlite) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			base_url TEXT NOT NULL,
			api_key TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS admin_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token TEXT NOT NULL UNIQUE,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES admin_users(id)
		);

		CREATE TABLE IF NOT EXISTS webhook_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT NOT NULL DEFAULT '',
			events TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

// --- Settings ---

func (s *Sqlite) GetSettings() (*model.Settings, error) {
	var st model.Settings
	err := s.db.QueryRow(
		"SELECT id, base_url, api_key, created_at, updated_at FROM settings ORDER BY id DESC LIMIT 1",
	).Scan(&st.ID, &st.BaseURL, &st.APIKey, &st.CreatedAt, &st.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Sqlite) SaveSettings(baseURL, apiKey string) error {
	existing, err := s.GetSettings()
	if err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)

	if existing != nil {
		_, err = s.db.Exec(
			"UPDATE settings SET base_url = ?, api_key = ?, updated_at = ? WHERE id = ?",
			baseURL, apiKey, now, existing.ID,
		)
	} else {
		_, err = s.db.Exec(
			"INSERT INTO settings (base_url, api_key, created_at, updated_at) VALUES (?, ?, ?, ?)",
			baseURL, apiKey, now, now,
		)
	}
	return err
}

func (s *Sqlite) HasSettings() bool {
	st, err := s.GetSettings()
	return err == nil && st != nil
}

// --- Admin Users ---

func (s *Sqlite) HasAdmins() bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
	return err == nil && count > 0
}

func (s *Sqlite) CreateAdmin(username, passwordHash string) error {
	_, err := s.db.Exec(
		"INSERT INTO admin_users (username, password_hash) VALUES (?, ?)",
		username, passwordHash,
	)
	return err
}

func (s *Sqlite) GetAdminByUsername(username string) (*model.AdminUser, error) {
	var u model.AdminUser
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, created_at FROM admin_users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// --- Sessions ---

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Sqlite) CreateSession(userID int, duration time.Duration) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(duration)
	_, err = s.db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expiresAt.Format(time.RFC3339),
	)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Sqlite) GetSession(token string) (*model.Session, error) {
	var sess model.Session
	var expiresAtStr string
	err := s.db.QueryRow(
		"SELECT s.id, s.token, s.user_id, s.expires_at, u.username FROM sessions s JOIN admin_users u ON s.user_id = u.id WHERE s.token = ?",
		token,
	).Scan(&sess.ID, &sess.Token, &sess.UserID, &expiresAtStr, &sess.Username)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		return nil, err
	}

	if time.Now().After(expiresAt) {
		s.DeleteSession(token)
		return nil, nil
	}

	sess.ExpiresAt = expiresAtStr
	return &sess, nil
}

func (s *Sqlite) DeleteSession(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (s *Sqlite) DeleteUserSessions(userID int) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

func (s *Sqlite) CleanExpiredSessions() {
	s.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now().Format(time.RFC3339))
}

// --- Webhook Config ---

func (s *Sqlite) GetWebhookConfig() (*model.WebhookConfig, error) {
	var wh model.WebhookConfig
	err := s.db.QueryRow(
		"SELECT id, url, events, enabled FROM webhook_config ORDER BY id DESC LIMIT 1",
	).Scan(&wh.ID, &wh.URL, &wh.Events, &wh.Enabled)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &wh, nil
}

func (s *Sqlite) SaveWebhookConfig(url, events string, enabled bool) error {
	existing, err := s.GetWebhookConfig()
	if err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	if existing != nil {
		_, err = s.db.Exec(
			"UPDATE webhook_config SET url = ?, events = ?, enabled = ?, updated_at = ? WHERE id = ?",
			url, events, enabledInt, now, existing.ID,
		)
	} else {
		_, err = s.db.Exec(
			"INSERT INTO webhook_config (url, events, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			url, events, enabledInt, now, now,
		)
	}
	return err
}

func (s *Sqlite) Close() error {
	return s.db.Close()
}
