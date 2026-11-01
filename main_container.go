package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/containers"
	"git.commsnet.org/commstech/repository-detective/runner"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

type containerScanBridge struct{}

func (containerScanBridge) Config(_ *gin.Context) (containers.Config, error) {
	return config.ContainerScan.Normalized(), nil
}

func (containerScanBridge) ListReferences(c *gin.Context, repoID int64) ([]store.ContainerImageReference, error) {
	if rdStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	return rdStore.ListContainerImageReferences(c.Request.Context(), repoID)
}

func (containerScanBridge) ListScans(c *gin.Context, repoID int64) ([]store.ContainerImageScan, error) {
	if rdStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	return rdStore.ListContainerImageScans(c.Request.Context(), repoID, 50)
}

func (containerScanBridge) Discover(c *gin.Context, repoID int64) ([]containers.ImageReference, error) {
	ctx := c.Request.Context()
	if rdStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	repo, err := rdStore.GetRepository(ctx, repoID)
	if err != nil {
		return nil, err
	}
	paths := []string{"Dockerfile", "docker-compose.yml", "compose.yaml", "compose.yml"}
	var files []containers.FileInput
	ref := repo.DefaultBranch
	if ref == "" {
		ref = "main"
	}
	if giteaClient != nil {
		for _, p := range paths {
			content, err := giteaClient.GetFileContent(ctx, repo.Owner, repo.Name, ref, p)
			if err != nil || strings.TrimSpace(content) == "" {
				continue
			}
			files = append(files, containers.FileInput{Path: p, Content: content})
		}
	}
	refs := containers.DiscoverImages(files, repoID)
	for _, r := range refs {
		_, _ = rdStore.UpsertContainerImageReference(ctx, store.ContainerImageReference{
			RepositoryID: repoID, Image: r.Image, Tag: r.Tag, Digest: r.Digest,
			TargetType: string(r.TargetType), FilePath: r.FilePath, Line: r.Line,
			ServiceName: r.ServiceName, MutableTag: r.MutableTag, PrivateRegistry: r.PrivateRegistry,
		})
	}
	return refs, nil
}

func (containerScanBridge) EnqueueScan(c *gin.Context, repoID int64, image string) (map[string]any, error) {
	ctx := c.Request.Context()
	cfg := config.ContainerScan.Normalized()
	if rdStore == nil {
		return nil, fmt.Errorf("database disabled")
	}
	if runnerDispatcher == nil {
		return nil, containers.ErrRunnerRequired
	}
	// Enqueue delegates to a native runner; not a core-local Docker socket scan.
	if err := containers.ValidateEnqueue(cfg, image, false); err != nil {
		return nil, err
	}
	repo, err := rdStore.GetRepository(ctx, repoID)
	if err != nil {
		return nil, err
	}
	scanID, err := newContainerScanID()
	if err != nil {
		return nil, err
	}
	payload := runner.ContainerScanPayload{
		TargetType: string(containers.TargetRegistryImage), Image: image,
		RepositoryID: repoID, ScanID: scanID, PullPolicy: string(cfg.PullPolicy),
		Tools: cfg.ToolList(), GenerateSBOM: cfg.GenerateSBOM, TimeoutSeconds: cfg.TimeoutSeconds,
	}
	policy := analyzers.PolicySnapshot{ScanProfile: "container_image_scan"}
	job, err := runnerDispatcher.CreateContainerImageScanJob(ctx, repo, scanID, payload, policy)
	if err != nil {
		return nil, err
	}
	cov, _ := json.Marshal(map[string]string{})
	warn, _ := json.Marshal([]string{})
	_, _ = rdStore.CreateContainerImageScan(ctx, store.ContainerImageScan{
		RepositoryID: repoID, ScanID: scanID, RunnerJobID: job.JobID, Image: image,
		Status: store.ContainerScanStatusQueued, CoverageJSON: cov, WarningsJSON: warn,
		StartedAt: time.Now().UTC(),
	})
	return map[string]any{
		"scan_id": scanID, "job_id": job.JobID, "image": image, "status": "queued",
		"require_runner": cfg.RequireRunner, "create_issues": cfg.CreateIssues,
	}, nil
}

func newContainerScanID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "cis-" + hex.EncodeToString(buf), nil
}

func applyContainerScanDefaults(cfg *Config) {
	def := containers.DefaultConfig()
	c := &cfg.ContainerScan
	if c.DefaultPolicy == "" {
		c.DefaultPolicy = def.DefaultPolicy
	}
	if c.PullPolicy == "" {
		c.PullPolicy = def.PullPolicy
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = def.TimeoutSeconds
	}
	if c.MaxImageSizeMB <= 0 {
		c.MaxImageSizeMB = def.MaxImageSizeMB
	}
	if len(c.RegistryCredentialsEnv) == 0 {
		c.RegistryCredentialsEnv = append([]string(nil), def.RegistryCredentialsEnv...)
	}
	if len(c.AllowedRunnerLabels) == 0 {
		c.AllowedRunnerLabels = append([]string(nil), def.AllowedRunnerLabels...)
	}
}

func ingestContainerScanResult(ctx context.Context, job store.RunnerJob, result runner.JobResult, repo store.Repository) {
	if result.ContainerScan == nil || rdStore == nil {
		return
	}
	cov, _ := json.Marshal(result.ContainerScan.Coverage)
	warn, _ := json.Marshal(result.ContainerScan.Warnings)
	status := store.ContainerScanStatusCompleted
	if result.Status == runner.JobStatusFailed {
		status = store.ContainerScanStatusFailed
	}
	scans, _ := rdStore.ListContainerImageScans(ctx, repo.ID, 5)
	for _, sc := range scans {
		if sc.RunnerJobID == job.JobID {
			_ = rdStore.UpdateContainerImageScan(ctx, sc.ID, status, result.ContainerScan.Digest,
				result.ContainerScan.VulnCount, cov, warn, time.Now().UTC())
			break
		}
	}
}
