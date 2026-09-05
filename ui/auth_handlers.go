package ui

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/internal/auth"
	"git.commsnet.org/commstech/repository-detective/internal/security"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

const (
	ctxAuthUserID    = "auth_user_id"
	ctxAuthSessionID = "auth_session_id"
	ctxAuthUser      = "auth_user"
)

// RegisterAuthRoutes mounts login, bootstrap, and logout (no session required).
func (h *Handler) RegisterAuthRoutes(g *gin.RouterGroup) {
	if h == nil || !h.auth.IsLocal() {
		return
	}
	g.GET("/bootstrap", h.BootstrapPage)
	g.POST("/bootstrap", h.BootstrapSubmit)
	g.GET("/login", h.LoginPage)
	g.POST("/login", h.LoginSubmit)
	g.POST("/logout", h.Logout)
}

// SessionAuthMiddleware requires a valid session when auth mode is local.
func (h *Handler) SessionAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h == nil || !h.auth.IsLocal() {
			c.Next()
			return
		}
		user, sessionID, ok := h.resolveSession(c)
		if !ok {
			next := c.Request.URL.Path
			if c.Request.URL.RawQuery != "" {
				next += "?" + c.Request.URL.RawQuery
			}
			c.Redirect(http.StatusFound, h.basePath+"/login?next="+urlPathEscape(next))
			c.Abort()
			return
		}
		c.Set(ctxAuthUser, user)
		c.Set(ctxAuthUserID, user.ID)
		c.Set(ctxAuthSessionID, sessionID)
		c.Next()
	}
}

func urlPathEscape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, " ", "%20"), "&", "%26")
}

func (h *Handler) resolveSession(c *gin.Context) (store.User, string, bool) {
	var zero store.User
	if h.store == nil {
		return zero, "", false
	}
	raw, err := c.Cookie(h.sessionCookieName())
	if err != nil || raw == "" {
		return zero, "", false
	}
	sessionID, ok := auth.ParseSessionCookie(h.auth.SessionSecret, raw)
	if !ok {
		return zero, "", false
	}
	sess, err := h.store.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		return zero, "", false
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		_ = h.store.DeleteSession(c.Request.Context(), sessionID)
		return zero, "", false
	}
	user, err := h.store.GetUserByID(c.Request.Context(), sess.UserID)
	if err != nil || !user.Enabled {
		return zero, "", false
	}
	return user, sessionID, true
}

func (h *Handler) sessionCookieName() string {
	if h.auth.SessionCookieName != "" {
		return h.auth.SessionCookieName
	}
	return "rd_session"
}

func (h *Handler) setSessionCookie(c *gin.Context, sessionID string, expiresAt time.Time) {
	signed, err := auth.SignSessionCookie(h.auth.SessionSecret, sessionID)
	if err != nil {
		return
	}
	secure := strings.HasPrefix(strings.ToLower(strings.TrimSpace(h.auth.PublicURL)), "https://")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.sessionCookieName(), signed, int(time.Until(expiresAt).Seconds()), h.basePath+"/", "", secure, true)
}

func (h *Handler) clearSessionCookie(c *gin.Context) {
	secure := strings.HasPrefix(strings.ToLower(strings.TrimSpace(h.auth.PublicURL)), "https://")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.sessionCookieName(), "", -1, h.basePath+"/", "", secure, true)
}

func (h *Handler) bootstrapAllowed(c *gin.Context) bool {
	if !h.auth.IsLocal() || !h.auth.LocalAdminBootstrapEnabled || h.store == nil {
		return false
	}
	count, err := h.store.CountUsers(c.Request.Context())
	return err == nil && count == 0
}

func (h *Handler) BootstrapPage(c *gin.Context) {
	if !h.bootstrapAllowed(c) {
		c.Redirect(http.StatusFound, h.basePath+"/login")
		return
	}
	h.renderAuth(c, "bootstrap.html", "Create admin account", map[string]any{
		"Error": "",
	})
}

func (h *Handler) BootstrapSubmit(c *gin.Context) {
	if !h.bootstrapAllowed(c) {
		c.String(http.StatusForbidden, "bootstrap is not available")
		return
	}
	if h.loginLimiter != nil && !h.loginLimiter.Allow(c.ClientIP()) {
		c.Header("Retry-After", "2")
		c.String(http.StatusTooManyRequests, "too many attempts — try again shortly")
		return
	}
	if h.auth.CSRFEnabled && !h.requireAuthCSRF(c, 0, "") {
		return
	}
	displayName := strings.TrimSpace(c.PostForm("display_name"))
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	confirm := c.PostForm("password_confirm")

	data := map[string]any{"Error": "", "DisplayName": displayName, "Email": email}
	if displayName == "" || email == "" {
		data["Error"] = "Display name and email are required."
		h.renderAuth(c, "bootstrap.html", "Create admin account", data)
		return
	}
	if password != confirm {
		data["Error"] = "Passwords do not match."
		h.renderAuth(c, "bootstrap.html", "Create admin account", data)
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		data["Error"] = "Choose a stronger password (12+ characters with letters and numbers)."
		h.renderAuth(c, "bootstrap.html", "Create admin account", data)
		return
	}
	user, err := h.store.CreateFirstOwner(c.Request.Context(), store.User{
		DisplayName:  displayName,
		Email:        email,
		PasswordHash: hash,
		Role:         store.RoleOwner,
		Enabled:      true,
	})
	if err != nil {
		if errors.Is(err, store.ErrBootstrapClosed) {
			c.String(http.StatusForbidden, "bootstrap is not available")
			return
		}
		h.logger.Errorf("bootstrap create user: %v", err)
		data["Error"] = "Could not create account. Try again."
		h.renderAuth(c, "bootstrap.html", "Create admin account", data)
		return
	}
	_ = h.store.AddAuthAuditEvent(c.Request.Context(), store.AuthAuditEvent{
		EventType: "auth.bootstrap.completed",
		UserID:    &user.ID,
		Email:     user.Email,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err := h.startSession(c, user); err != nil {
		c.String(http.StatusInternalServerError, "account created but session failed")
		return
	}
	c.Redirect(http.StatusFound, h.basePath+"/")
}

func (h *Handler) LoginPage(c *gin.Context) {
	if _, _, ok := h.resolveSession(c); ok {
		c.Redirect(http.StatusFound, h.basePath+"/")
		return
	}
	if h.bootstrapAllowed(c) {
		c.Redirect(http.StatusFound, h.basePath+"/bootstrap")
		return
	}
	errMsg := ""
	switch c.Query("error") {
	case "invalid":
		errMsg = "Invalid email or password."
	case "disabled":
		errMsg = "This account is disabled."
	}
	h.renderAuth(c, "login.html", "Sign in", map[string]any{
		"Error": errMsg,
		"Next":  c.Query("next"),
	})
}

func (h *Handler) LoginSubmit(c *gin.Context) {
	if h.bootstrapAllowed(c) {
		c.Redirect(http.StatusFound, h.basePath+"/bootstrap")
		return
	}
	if h.loginLimiter != nil && !h.loginLimiter.Allow(c.ClientIP()) {
		c.Header("Retry-After", "2")
		c.String(http.StatusTooManyRequests, "too many attempts — try again shortly")
		return
	}
	if h.auth.CSRFEnabled && !h.requireAuthCSRF(c, 0, "") {
		return
	}
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	next := strings.TrimSpace(c.PostForm("next"))

	user, err := h.store.GetUserByEmail(c.Request.Context(), email)
	if err != nil || !auth.CheckPassword(user.PasswordHash, password) {
		_ = h.store.AddAuthAuditEvent(c.Request.Context(), store.AuthAuditEvent{
			EventType: "auth.login.failed",
			Email:     email,
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
		c.Redirect(http.StatusFound, h.basePath+"/login?error=invalid")
		return
	}
	if !user.Enabled {
		c.Redirect(http.StatusFound, h.basePath+"/login?error=disabled")
		return
	}
	_ = h.store.AddAuthAuditEvent(c.Request.Context(), store.AuthAuditEvent{
		EventType: "auth.login.success",
		UserID:    &user.ID,
		Email:     user.Email,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err := h.startSession(c, user); err != nil {
		c.String(http.StatusInternalServerError, "login failed")
		return
	}
	dest := h.basePath + "/"
	if next != "" && strings.HasPrefix(next, h.basePath) {
		dest = next
	}
	c.Redirect(http.StatusFound, dest)
}

func (h *Handler) Logout(c *gin.Context) {
	user, sessionID, ok := h.resolveSession(c)
	if !ok {
		c.Redirect(http.StatusFound, h.basePath+"/login")
		return
	}
	if h.auth.CSRFEnabled && !h.requireAuthCSRF(c, user.ID, sessionID) {
		return
	}
	_ = h.store.DeleteSession(c.Request.Context(), sessionID)
	h.clearSessionCookie(c)
	c.Redirect(http.StatusFound, h.basePath+"/login")
}

func (h *Handler) startSession(c *gin.Context, user store.User) error {
	sessionID, err := auth.NewSessionID()
	if err != nil {
		return err
	}
	expiresAt := auth.SessionExpiresAt(h.auth.SessionTTLHours)
	if err := h.store.CreateSession(c.Request.Context(), store.Session{
		ID:        sessionID,
		UserID:    user.ID,
		ExpiresAt: expiresAt,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}); err != nil {
		return err
	}
	_ = h.store.UpdateUserLastLogin(c.Request.Context(), user.ID, time.Now().UTC())
	h.setSessionCookie(c, sessionID, expiresAt)
	c.Set(ctxAuthUser, user)
	c.Set(ctxAuthUserID, user.ID)
	c.Set(ctxAuthSessionID, sessionID)
	return nil
}

func (h *Handler) requireAuthCSRF(c *gin.Context, userID int64, sessionID string) bool {
	token := c.PostForm("csrf_token")
	if sessionID == "" {
		sessionID = c.PostForm("csrf_session")
	}
	if userID <= 0 {
		// Bootstrap/login forms before session exists use a bootstrap CSRF token.
		if !security.ValidSessionCSRFToken(h.auth.SessionSecret, "bootstrap", 0, token) {
			c.String(http.StatusForbidden, "invalid or missing CSRF token")
			return false
		}
		return true
	}
	if !security.ValidSessionCSRFToken(h.auth.SessionSecret, sessionID, userID, token) {
		c.String(http.StatusForbidden, "invalid or missing CSRF token")
		return false
	}
	return true
}

func (h *Handler) renderAuth(c *gin.Context, name, title string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	pd := pageData{
		Title:     title,
		BasePath:  h.basePath,
		CSRFToken: security.SessionCSRFToken(h.auth.SessionSecret, "bootstrap", 0),
		Data:      data,
	}
	if err := h.tmpl.ExecuteTemplate(c.Writer, name, pd); err != nil {
		h.logger.Errorf("render %s: %v", name, err)
		c.String(http.StatusInternalServerError, "template error")
	}
}
