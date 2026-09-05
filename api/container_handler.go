package api

import (
	"net/http"
	"strconv"
	"strings"

	"git.commsnet.org/commstech/repository-detective/containers"
	"git.commsnet.org/commstech/repository-detective/runner"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

// ContainerScanService enqueues and lists container image scans.
type ContainerScanService interface {
	Config(c *gin.Context) (containers.Config, error)
	ListReferences(c *gin.Context, repoID int64) ([]store.ContainerImageReference, error)
	ListScans(c *gin.Context, repoID int64) ([]store.ContainerImageScan, error)
	Discover(c *gin.Context, repoID int64) ([]containers.ImageReference, error)
	EnqueueScan(c *gin.Context, repoID int64, image string) (map[string]any, error)
}

// ContainerHandler serves container image scanning API routes.
type ContainerHandler struct {
	store   store.QueryStore
	service ContainerScanService
}

// NewContainerHandler creates a container scanning handler.
func NewContainerHandler(s store.QueryStore, svc ContainerScanService) *ContainerHandler {
	return &ContainerHandler{store: s, service: svc}
}

// RegisterRoutes mounts container scanning routes.
func (h *ContainerHandler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/containers/config", h.GetConfig)
	g.GET("/repositories/:id/containers/images", h.ListReferences)
	g.GET("/repositories/:id/containers/scans", h.ListScans)
	g.POST("/repositories/:id/containers/discover", h.Discover)
	g.POST("/repositories/:id/containers/scan", h.EnqueueScan)
}

func (h *ContainerHandler) requireStore(c *gin.Context) bool {
	if h.store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database disabled"})
		return false
	}
	return true
}

func (h *ContainerHandler) repoID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository id"})
		return 0, false
	}
	return id, true
}

func (h *ContainerHandler) GetConfig(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "container scanning unavailable"})
		return
	}
	cfg, err := h.service.Config(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":                    cfg.Enabled,
		"require_runner":             cfg.RequireRunner,
		"allow_core_docker_socket":   cfg.AllowCoreDockerSocket,
		"create_issues":              cfg.CreateIssues,
		"default_policy":             cfg.DefaultPolicy,
		"tools":                      cfg.Tools,
		"allowed_runner_labels":      cfg.AllowedRunnerLabels,
		"note":                       "Container scanning is opt-in. Core never mounts Docker socket by default.",
	})
}

func (h *ContainerHandler) ListReferences(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	repoID, ok := h.repoID(c)
	if !ok {
		return
	}
	refs, err := h.service.ListReferences(c, repoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"references": refs, "count": len(refs)})
}

func (h *ContainerHandler) ListScans(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	repoID, ok := h.repoID(c)
	if !ok {
		return
	}
	scans, err := h.service.ListScans(c, repoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"scans": scans, "count": len(scans)})
}

func (h *ContainerHandler) Discover(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	repoID, ok := h.repoID(c)
	if !ok {
		return
	}
	refs, err := h.service.Discover(c, repoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"discovered": refs, "count": len(refs)})
}

type enqueueContainerScanRequest struct {
	Image string `json:"image" binding:"required"`
}

func (h *ContainerHandler) EnqueueScan(c *gin.Context) {
	if !h.requireStore(c) || h.service == nil {
		return
	}
	repoID, ok := h.repoID(c)
	if !ok {
		return
	}
	var req enqueueContainerScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image required"})
		return
	}
	out, err := h.service.EnqueueScan(c, repoID, strings.TrimSpace(req.Image))
	if err != nil {
		switch err {
		case containers.ErrScanningDisabled:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case containers.ErrRunnerRequired:
			c.JSON(http.StatusAccepted, gin.H{"error": err.Error(), "require_runner": true})
		case containers.ErrImageNotAllowed, containers.ErrCoreDockerForbidden:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusAccepted, out)
}

// Ensure runner package referenced for job type constant in docs.
var _ = runner.JobTypeContainerImageScan
