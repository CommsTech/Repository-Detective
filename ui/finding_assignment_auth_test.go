package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/internal/auth"
	"git.commsnet.org/commstech/repository-detective/internal/security"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func TestLocalFindingOwnershipRequiresRoleAndCSRF(t *testing.T) {
	ctx := context.Background()
	h, s := newAuthTestHandler(t)
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	hash, err := auth.HashPassword("abcdefghijkl1")
	if err != nil {
		t.Fatal(err)
	}
	create := func(email, role string, enabled bool) store.User {
		t.Helper()
		u, err := s.CreateUser(ctx, store.User{Email: email, DisplayName: email, PasswordHash: hash, Role: role, Enabled: enabled})
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	owner := create("owner@team.example", store.RoleOwner, true)
	developer := create("developer@team.example", store.RoleDeveloper, true)
	viewer := create("viewer@team.example", store.RoleViewer, true)
	disabled := create("disabled@team.example", store.RoleDeveloper, false)
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "team", Name: "app", FullName: "team/app"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "role-assignment", Status: "open", Severity: "high", Category: "security", Source: "gosec", Title: "Example vulnerability", FirstSeenAt: time.Now().UTC(), LastSeenAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	g := r.Group("/ui")
	g.Use(h.SessionAuthMiddleware())
	h.RegisterRoutes(g)
	sessionFor := func(u store.User) (string, string) {
		t.Helper()
		id, err := auth.NewSessionID()
		if err != nil {
			t.Fatal(err)
		}
		if err := s.CreateSession(ctx, store.Session{ID: id, UserID: u.ID, ExpiresAt: time.Now().UTC().Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
		signed, err := auth.SignSessionCookie(h.auth.SessionSecret, id)
		if err != nil {
			t.Fatal(err)
		}
		return signed, security.SessionCSRFToken(h.auth.SessionSecret, id, u.ID)
	}
	post := func(u store.User, target int64, csrf string, cookie string) *httptest.ResponseRecorder {
		form := url.Values{"csrf_token": {csrf}, "assignee_user_id": {strconv.FormatInt(target, 10)}}
		req := httptest.NewRequest(http.MethodPost, "/ui/findings/"+strconv.FormatInt(f.ID, 10)+"/assign", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "rd_session", Value: cookie})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	ownerCookie, ownerCSRF := sessionFor(owner)
	get := httptest.NewRequest(http.MethodGet, "/ui/findings/"+strconv.FormatInt(f.ID, 10), nil)
	get.AddCookie(&http.Cookie{Name: "rd_session", Value: ownerCookie})
	rendered := httptest.NewRecorder()
	r.ServeHTTP(rendered, get)
	if rendered.Code != http.StatusOK || !strings.Contains(rendered.Body.String(), "Assign to team member") {
		t.Fatalf("finding detail render %d: %s", rendered.Code, rendered.Body.String())
	}
	if status := post(owner, developer.ID, ownerCSRF, ownerCookie).Code; status != http.StatusSeeOther {
		t.Fatalf("owner assignment status %d", status)
	}
	// A developer sees their own work, while the assigning owner does not.
	developerCookie, _ := sessionFor(developer)
	getFindings := func(cookie, target string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.AddCookie(&http.Cookie{Name: "rd_session", Value: cookie})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	myWork := getFindings(developerCookie, "/ui/findings?owner=mine")
	if myWork.Code != http.StatusOK || !strings.Contains(myWork.Body.String(), "Example vulnerability") || !strings.Contains(myWork.Body.String(), "My findings") {
		t.Fatalf("developer work queue %d: %s", myWork.Code, myWork.Body.String())
	}
	unassignedWork := getFindings(ownerCookie, "/ui/findings?owner=unassigned")
	if unassignedWork.Code != http.StatusOK || strings.Contains(unassignedWork.Body.String(), "Example vulnerability") {
		t.Fatalf("assigned finding leaked to unassigned queue: %d", unassignedWork.Code)
	}
	ownerWork := getFindings(ownerCookie, "/ui/findings?owner=mine")
	if ownerWork.Code != http.StatusOK || strings.Contains(ownerWork.Body.String(), "Example vulnerability") {
		t.Fatalf("owner work queue should be empty %d", ownerWork.Code)
	}
	jsonDownload := getFindings(developerCookie, "/ui/findings/export?format=json&owner=mine")
	if jsonDownload.Code != http.StatusOK || !strings.Contains(jsonDownload.Body.String(), "Example vulnerability") || !strings.Contains(jsonDownload.Body.String(), "developer@team.example") {
		t.Fatalf("filtered export %d: %s", jsonDownload.Code, jsonDownload.Body.String())
	}
	csvDownload := getFindings(developerCookie, "/ui/findings/export?format=csv&owner=mine")
	if csvDownload.Code != http.StatusOK || !strings.Contains(csvDownload.Body.String(), "line,assignee") || !strings.Contains(csvDownload.Body.String(), "developer@team.example") {
		t.Fatalf("filtered CSV export %d: %s", csvDownload.Code, csvDownload.Body.String())
	}
	badScope := getFindings(developerCookie, "/ui/findings?owner=some-other-user")
	if badScope.Code != http.StatusBadRequest {
		t.Fatalf("unsupported ownership scope must fail: %d", badScope.Code)
	}
	badExport := getFindings(developerCookie, "/ui/findings/export?format=json&owner=some-other-user")
	if badExport.Code != http.StatusBadRequest {
		t.Fatalf("unsupported export scope must fail: %d", badExport.Code)
	}
	assigned, err := s.GetFindingAssignment(ctx, f.ID)
	if err != nil || assigned.AssigneeUserID != developer.ID {
		t.Fatalf("assignment %+v %v", assigned, err)
	}
	viewerCookie, viewerCSRF := sessionFor(viewer)
	if status := post(viewer, 0, viewerCSRF, viewerCookie).Code; status != http.StatusForbidden {
		t.Fatalf("viewer write status %d", status)
	}
	if status := post(owner, 0, "bad-token", ownerCookie).Code; status != http.StatusForbidden {
		t.Fatalf("CSRF status %d", status)
	}
	if status := post(owner, disabled.ID, ownerCSRF, ownerCookie).Code; status != http.StatusUnprocessableEntity {
		t.Fatalf("disabled assignee status %d", status)
	}
}
