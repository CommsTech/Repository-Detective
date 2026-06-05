package store

import "time"

// Auth role constants for slice 1 (full RBAC later).
const (
	RoleOwner    = "owner"
	RoleAdmin    = "admin"
	RoleReadOnly = "read_only"
)

// User is a local admin account.
type User struct {
	ID           int64
	Email        string
	DisplayName  string
	PasswordHash string
	Role         string
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
}

// Session is a browser session bound to a user.
type Session struct {
	ID        string
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time
	IPAddress string
	UserAgent string
}

// AuthAuditEvent is an append-only security audit record.
type AuthAuditEvent struct {
	ID        int64
	EventType string
	UserID    *int64
	Email     string
	IPAddress string
	UserAgent string
	Details   string
	CreatedAt time.Time
}
