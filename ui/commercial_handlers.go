package ui

import (
	"net/http"
	"strings"

	"git.commsnet.org/commstech/repository-detective/internal/auth"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func (h *Handler) currentRole(c *gin.Context) string {
	if u := h.currentUser(c); u != nil {
		return store.NormalizeRole(u.Role)
	}
	// API-key sessions have no role — treat as owner for single-operator mode.
	return store.RoleOwner
}

func (h *Handler) currentUser(c *gin.Context) *store.User {
	if v, ok := c.Get(ctxAuthUser); ok {
		if u, ok := v.(store.User); ok {
			cp := u
			return &cp
		}
		if u, ok := v.(*store.User); ok {
			return u
		}
	}
	return nil
}

func (h *Handler) authStore() (store.AuthStore, bool) {
	as, ok := any(h.store).(store.AuthStore)
	return as, ok
}

func (h *Handler) requireCommercial(c *gin.Context, navSection string) bool {
	if h.editionCommercial {
		return true
	}
	c.Status(http.StatusPaymentRequired)
	h.renderNav(c, "error.html", "Commercial feature", navSection, map[string]any{
		"Message": "Multi-user RBAC and the operator audit trail require Commercial edition. Set edition=commercial and a license_key, or keep Community with a single operator.",
	})
	return false
}

func (h *Handler) requireManageUsers(c *gin.Context) bool {
	if !h.auth.IsLocal() {
		h.renderNav(c, "error.html", "Local auth required", "users", map[string]any{
			"Message": "User management requires auth_mode=local.",
		})
		return false
	}
	if !store.RoleCanManageUsers(h.currentRole(c)) {
		c.Status(http.StatusForbidden)
		h.renderNav(c, "error.html", "Forbidden", "users", map[string]any{
			"Message": "Only owner or admin roles can manage users.",
		})
		return false
	}
	return true
}

// UsersPage lists local users (Commercial) or explains the Community limit.
func (h *Handler) UsersPage(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	if !h.auth.IsLocal() {
		h.renderNav(c, "users.html", "Users", "users", map[string]any{
			"Commercial": h.editionCommercial,
			"LocalAuth":  false,
			"Users":      []store.User{},
			"Roles":      store.ValidAssignableRoles(),
			"Upsell":     "Enable auth_mode=local to use operator accounts.",
			"CanManage":  false,
			"Notice":     "",
		})
		return
	}
	if !h.editionCommercial {
		h.renderNav(c, "users.html", "Users", "users", map[string]any{
			"Commercial": false,
			"LocalAuth":  true,
			"Users":      []store.User{},
			"Roles":      store.ValidAssignableRoles(),
			"Upsell":     "Community edition is single-operator. Unlock Commercial for Admin / Security / Developer / Viewer RBAC.",
			"CanManage":  false,
			"Notice":     "",
		})
		return
	}
	var listed []store.User
	if as, ok := h.authStore(); ok {
		listed, _ = as.ListUsers(c.Request.Context(), 200)
	}
	h.renderNav(c, "users.html", "Users", "users", map[string]any{
		"Commercial": true,
		"LocalAuth":  true,
		"Users":      listed,
		"Roles":      store.ValidAssignableRoles(),
		"Upsell":     "",
		"CanManage":  store.RoleCanManageUsers(h.currentRole(c)),
		"Notice":     c.Query("notice"),
	})
}

// CreateUserSubmit creates an additional operator (Commercial + local auth).
func (h *Handler) CreateUserSubmit(c *gin.Context) {
	if !h.requireStore(c) || !h.requireCommercial(c, "users") || !h.requireManageUsers(c) {
		return
	}
	if !h.requireCSRF(c) {
		return
	}
	email := strings.TrimSpace(c.PostForm("email"))
	display := strings.TrimSpace(c.PostForm("display_name"))
	password := c.PostForm("password")
	role := store.NormalizeRole(strings.TrimSpace(c.PostForm("role")))
	validRole := false
	for _, r := range store.ValidAssignableRoles() {
		if r == role {
			validRole = true
			break
		}
	}
	if !validRole {
		h.renderNav(c, "error.html", "Invalid role", "users", map[string]any{"Message": "Choose admin, security, developer, or viewer."})
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		h.renderNav(c, "error.html", "Invalid password", "users", map[string]any{"Message": err.Error()})
		return
	}
	as, ok := h.authStore()
	if !ok {
		h.renderNav(c, "error.html", "Unavailable", "users", map[string]any{"Message": "Auth store unavailable."})
		return
	}
	user, err := as.CreateUser(c.Request.Context(), store.User{
		Email: email, DisplayName: display, PasswordHash: hash, Role: role, Enabled: true,
	})
	if err != nil {
		h.renderNav(c, "error.html", "Create user failed", "users", map[string]any{"Message": err.Error()})
		return
	}
	actor := h.currentUser(c)
	var uid *int64
	emailAudit := ""
	if actor != nil {
		uid = &actor.ID
		emailAudit = actor.Email
	}
	_ = as.AddAuthAuditEvent(c.Request.Context(), store.AuthAuditEvent{
		EventType: "user_created",
		UserID:    uid,
		Email:     emailAudit,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Details:   "created " + user.Email + " role=" + user.Role,
	})
	c.Redirect(http.StatusSeeOther, h.basePath+"/users?notice=created")
}

// AuditTrailPage shows auth_audit_events (Commercial).
func (h *Handler) AuditTrailPage(c *gin.Context) {
	if !h.requireStore(c) {
		return
	}
	if !h.auth.IsLocal() {
		h.renderNav(c, "audit.html", "Audit trail", "audit", map[string]any{
			"Commercial": h.editionCommercial,
			"Events":     []store.AuthAuditEvent{},
			"Upsell":     "Enable auth_mode=local to record operator audit events.",
		})
		return
	}
	if !h.requireCommercial(c, "audit") {
		return
	}
	var events []store.AuthAuditEvent
	if as, ok := h.authStore(); ok {
		events, _ = as.ListAuthAuditEvents(c.Request.Context(), 200)
	}
	h.renderNav(c, "audit.html", "Audit trail", "audit", map[string]any{
		"Commercial": true,
		"Events":     events,
		"Upsell":     "",
	})
}
