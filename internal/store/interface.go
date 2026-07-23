package store

import (
	"headcontrol/internal/model"
	"time"
)

type Store interface {
	// Settings
	GetSettings() (*model.Settings, error)
	SaveSettings(string, string) error
	HasSettings() bool

	// Admin users
	HasAdmins() bool
	CreateAdmin(username, passwordHash string) error
	GetAdminByUsername(username string) (*model.AdminUser, error)

	// Sessions
	CreateSession(userID int, duration time.Duration) (string, error)
	GetSession(token string) (*model.Session, error)
	DeleteSession(token string) error
	DeleteUserSessions(userID int) error
	CleanExpiredSessions()

	// Webhook config
	GetWebhookConfig() (*model.WebhookConfig, error)
	SaveWebhookConfig(url, events string, enabled bool) error

	// Lifecycle
	Close() error
}
