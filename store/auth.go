package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrBootstrapClosed is returned when CreateFirstOwner runs after any user exists.
var ErrBootstrapClosed = errors.New("bootstrap closed: local users already exist")

// AuthStore provides local user/session persistence.
type AuthStore interface {
	CountUsers(ctx context.Context) (int, error)
	CreateUser(ctx context.Context, user User) (User, error)
	// CreateFirstOwner inserts the first owner inside an exclusive transaction.
	// Returns ErrBootstrapClosed when any user already exists.
	CreateFirstOwner(ctx context.Context, user User) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByID(ctx context.Context, id int64) (User, error)
	UpdateUserLastLogin(ctx context.Context, id int64, at time.Time) error

	CreateSession(ctx context.Context, session Session) error
	GetSession(ctx context.Context, id string) (Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteSessionsForUser(ctx context.Context, userID int64) error

	AddAuthAuditEvent(ctx context.Context, event AuthAuditEvent) error
}

func (s *SQLiteStore) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (s *SQLiteStore) CreateUser(ctx context.Context, user User) (User, error) {
	email := strings.TrimSpace(strings.ToLower(user.Email))
	if email == "" {
		return User{}, fmt.Errorf("email is required")
	}
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}
	enabled := 1
	if !user.Enabled {
		enabled = 0
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO users (email, display_name, password_hash, role, enabled, created_at, updated_at, last_login_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL)`,
		email, strings.TrimSpace(user.DisplayName), user.PasswordHash, user.Role, enabled,
		formatTime(user.CreatedAt), formatTime(user.UpdatedAt),
	)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("create user id: %w", err)
	}
	user.ID = id
	user.Email = email
	user.Enabled = enabled == 1
	return user, nil
}

// CreateFirstOwner atomically creates the first owner when the users table is empty.
func (s *SQLiteStore) CreateFirstOwner(ctx context.Context, user User) (User, error) {
	email := strings.TrimSpace(strings.ToLower(user.Email))
	if email == "" {
		return User{}, fmt.Errorf("email is required")
	}
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}
	enabled := 1
	if !user.Enabled {
		enabled = 0
	}

	conn, err := s.db.Conn(ctx)
	if err != nil {
		return User{}, fmt.Errorf("bootstrap conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return User{}, fmt.Errorf("begin immediate bootstrap: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), `ROLLBACK`)
		}
	}()

	var count int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&count); err != nil {
		return User{}, fmt.Errorf("count users for bootstrap: %w", err)
	}
	if count > 0 {
		return User{}, ErrBootstrapClosed
	}

	res, err := conn.ExecContext(ctx, `
		INSERT INTO users (email, display_name, password_hash, role, enabled, created_at, updated_at, last_login_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL)`,
		email, strings.TrimSpace(user.DisplayName), user.PasswordHash, user.Role, enabled,
		formatTime(user.CreatedAt), formatTime(user.UpdatedAt),
	)
	if err != nil {
		return User{}, fmt.Errorf("create first owner: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("create first owner id: %w", err)
	}
	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return User{}, fmt.Errorf("commit bootstrap: %w", err)
	}
	committed = true
	user.ID = id
	user.Email = email
	user.Enabled = enabled == 1
	return user, nil
}

func (s *SQLiteStore) GetUserByEmail(ctx context.Context, email string) (User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	var u User
	var enabled int
	var lastLogin sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, display_name, password_hash, role, enabled, created_at, updated_at, last_login_at
		FROM users WHERE email = ? COLLATE NOCASE`, email).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Role, &enabled,
		&formatTimeScan{&u.CreatedAt}, &formatTimeScan{&u.UpdatedAt}, &lastLogin,
	)
	if err != nil {
		if isNoRows(err) {
			return User{}, fmt.Errorf("user not found")
		}
		return User{}, fmt.Errorf("get user by email: %w", err)
	}
	u.Enabled = enabled == 1
	u.LastLoginAt = parseTimePtr(lastLogin)
	return u, nil
}

func (s *SQLiteStore) GetUserByID(ctx context.Context, id int64) (User, error) {
	var u User
	var enabled int
	var lastLogin sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, display_name, password_hash, role, enabled, created_at, updated_at, last_login_at
		FROM users WHERE id = ?`, id).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.Role, &enabled,
		&formatTimeScan{&u.CreatedAt}, &formatTimeScan{&u.UpdatedAt}, &lastLogin,
	)
	if err != nil {
		if isNoRows(err) {
			return User{}, fmt.Errorf("user not found")
		}
		return User{}, fmt.Errorf("get user by id: %w", err)
	}
	u.Enabled = enabled == 1
	u.LastLoginAt = parseTimePtr(lastLogin)
	return u, nil
}

func (s *SQLiteStore) UpdateUserLastLogin(ctx context.Context, id int64, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?`,
		formatTime(at.UTC()), formatTime(at.UTC()), id,
	)
	if err != nil {
		return fmt.Errorf("update user last login: %w", err)
	}
	return nil
}

func (s *SQLiteStore) CreateSession(ctx context.Context, session Session) error {
	if strings.TrimSpace(session.ID) == "" {
		return fmt.Errorf("session id is required")
	}
	if session.UserID <= 0 {
		return fmt.Errorf("session user_id is required")
	}
	now := time.Now().UTC()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, created_at, expires_at, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?)`,
		session.ID, session.UserID,
		formatTime(session.CreatedAt), formatTime(session.ExpiresAt),
		session.IPAddress, session.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetSession(ctx context.Context, id string) (Session, error) {
	var sess Session
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, created_at, expires_at, ip_address, user_agent
		FROM sessions WHERE id = ?`, id).Scan(
		&sess.ID, &sess.UserID,
		&formatTimeScan{&sess.CreatedAt}, &formatTimeScan{&sess.ExpiresAt},
		&sess.IPAddress, &sess.UserAgent,
	)
	if err != nil {
		if isNoRows(err) {
			return Session{}, fmt.Errorf("session not found")
		}
		return Session{}, fmt.Errorf("get session: %w", err)
	}
	return sess, nil
}

func (s *SQLiteStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteSessionsForUser(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete sessions for user: %w", err)
	}
	return nil
}

func (s *SQLiteStore) AddAuthAuditEvent(ctx context.Context, event AuthAuditEvent) error {
	now := time.Now().UTC()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	var userID any
	if event.UserID != nil {
		userID = *event.UserID
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_audit_events (event_type, user_id, email, ip_address, user_agent, details, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.EventType, userID, event.Email, event.IPAddress, event.UserAgent, event.Details,
		formatTime(event.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("add auth audit event: %w", err)
	}
	return nil
}
