package containers_test

import (
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/containers"
)

func TestDiscoverComposeImages(t *testing.T) {
	content := `services:
  api:
    image: ghcr.io/org/api:1.2.3
  web:
    image: nginx:latest
`
	refs := containers.DiscoverImages([]containers.FileInput{{
		Path: "docker-compose.yml", Content: content,
	}}, 1)
	if len(refs) < 2 {
		t.Fatalf("expected >=2 refs, got %d", len(refs))
	}
	var nginx bool
	for _, r := range refs {
		if strings.Contains(r.Image, "nginx") {
			nginx = true
			if !r.MutableTag {
				t.Fatal("nginx:latest should be mutable")
			}
		}
		if strings.Contains(r.Image, "ghcr.io") && r.Digest == "" {
			if !r.MutableTag && !strings.Contains(r.Image, "@") {
				// tagged semver is ok
			}
		}
	}
	if !nginx {
		t.Fatal("nginx image not discovered")
	}
}

func TestDiscoverDockerfileFROM(t *testing.T) {
	content := "FROM alpine:3.20\nRUN echo hi\n"
	refs := containers.DiscoverImages([]containers.FileInput{{
		Path: "Dockerfile", Content: content,
	}}, 1)
	if len(refs) != 1 || !strings.Contains(refs[0].Image, "alpine") {
		t.Fatalf("refs %+v", refs)
	}
}

func TestDiscoverKubernetesImage(t *testing.T) {
	content := `apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: app
        image: registry.example.com/app@sha256:abc123
`
	refs := containers.DiscoverImages([]containers.FileInput{{
		Path: "deploy/k8s/app.yaml", Content: content,
	}}, 1)
	if len(refs) != 1 {
		t.Fatalf("refs %+v", refs)
	}
	if refs[0].Digest == "" {
		t.Fatal("expected digest")
	}
}

func TestValidateEnqueueDisabled(t *testing.T) {
	cfg := containers.DefaultConfig()
	if err := containers.ValidateEnqueue(cfg, "alpine:3.20", true); err != containers.ErrScanningDisabled {
		t.Fatalf("got %v", err)
	}
}

func TestValidateEnqueueRunnerRequired(t *testing.T) {
	cfg := containers.DefaultConfig()
	cfg.Enabled = true
	if err := containers.ValidateEnqueue(cfg, "alpine:3.20", true); err != containers.ErrRunnerRequired {
		t.Fatalf("got %v", err)
	}
}

func TestRegistryAllowBlock(t *testing.T) {
	cfg := containers.DefaultConfig()
	cfg.Enabled = true
	cfg.AllowedRegistries = []string{"ghcr.io"}
	if !cfg.RegistryAllowed("ghcr.io/org/app:1") {
		t.Fatal("should allow ghcr")
	}
	if cfg.RegistryAllowed("evil.io/app:1") {
		t.Fatal("should deny unlisted registry")
	}
	cfg.AllowedRegistries = []string{"docker.io"}
	if !cfg.RegistryAllowed("alpine:3.20") {
		t.Fatal("implicit docker hub image should match docker.io allowlist")
	}
	cfg.AllowedRegistries = nil
	cfg.BlockedRegistries = []string{"evil.io"}
	if cfg.RegistryAllowed("evil.io/app:1") {
		t.Fatal("blocked registry")
	}
}

func TestRedactLogLine(t *testing.T) {
	out := containers.RedactLogLine("token=supersecret123")
	if strings.Contains(out, "supersecret") {
		t.Fatalf("not redacted: %q", out)
	}
}
