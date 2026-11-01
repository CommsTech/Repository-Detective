package store

import "time"

// Auth role constants for Commercial RBAC (RD-COMMERCIAL-001).
const (
	RoleOwner     = "owner"
	RoleAdmin     = "admin"
	RoleSecurity  = "security"
	RoleDeveloper = "developer"
	RoleViewer    = "viewer"
	RoleReadOnly  = "read_only" // legacy alias for viewer
)

// NormalizeRole maps legacy / alias role names to canonical values.
func NormalizeRole(role string) string {
	switch role {
	case RoleOwner, RoleAdmin, RoleSecurity, RoleDeveloper, RoleViewer:
		return role
	case RoleReadOnly, "readonly", "read-only":
		return RoleViewer
	default:
		return role
	}
}

// RoleCanManageUsers is true for owner/admin.
func RoleCanManageUsers(role string) bool {
	switch NormalizeRole(role) {
	case RoleOwner, RoleAdmin:
		return true
	default:
		return false
	}
}

// RoleCanWriteSecurity is true for roles that may suppress/calibrate/configure security.
func RoleCanWriteSecurity(role string) bool {
	switch NormalizeRole(role) {
	case RoleOwner, RoleAdmin, RoleSecurity:
		return true
	default:
		return false
	}
}

// RoleCanMutatePlatform is true for owner/admin configure writes.
func RoleCanMutatePlatform(role string) bool {
	switch NormalizeRole(role) {
	case RoleOwner, RoleAdmin:
		return true
	default:
		return false
	}
}

// ValidAssignableRoles lists roles an admin may assign (not owner).
func ValidAssignableRoles() []string {
	return []string{RoleAdmin, RoleSecurity, RoleDeveloper, RoleViewer}
}

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
